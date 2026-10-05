package docstore

import (
	"bytes"
	"context"
	"io"
	"sort"
	"strings"
	"sync"
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/s3/types"
)

type fakeS3 struct {
	mu   sync.Mutex
	objs map[string][]byte
}

func (f *fakeS3) GetObject(_ context.Context, in *s3.GetObjectInput, _ ...func(*s3.Options)) (*s3.GetObjectOutput, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	b, ok := f.objs[aws.ToString(in.Key)]
	if !ok {
		return nil, &types.NoSuchKey{}
	}
	return &s3.GetObjectOutput{Body: io.NopCloser(bytes.NewReader(b))}, nil
}

func (f *fakeS3) PutObject(_ context.Context, in *s3.PutObjectInput, _ ...func(*s3.Options)) (*s3.PutObjectOutput, error) {
	b, _ := io.ReadAll(in.Body)
	f.mu.Lock()
	f.objs[aws.ToString(in.Key)] = b
	f.mu.Unlock()
	return &s3.PutObjectOutput{}, nil
}

func (f *fakeS3) CreateBucket(_ context.Context, in *s3.CreateBucketInput, _ ...func(*s3.Options)) (*s3.CreateBucketOutput, error) {
	return &s3.CreateBucketOutput{}, nil
}

func (f *fakeS3) DeleteObject(_ context.Context, in *s3.DeleteObjectInput, _ ...func(*s3.Options)) (*s3.DeleteObjectOutput, error) {
	f.mu.Lock()
	delete(f.objs, aws.ToString(in.Key))
	f.mu.Unlock()
	return &s3.DeleteObjectOutput{}, nil
}

// ListObjectsV2 pages two keys at a time to exercise pagination.
func (f *fakeS3) ListObjectsV2(_ context.Context, in *s3.ListObjectsV2Input, _ ...func(*s3.Options)) (*s3.ListObjectsV2Output, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	var keys []string
	for k := range f.objs {
		if strings.HasPrefix(k, aws.ToString(in.Prefix)) && k > aws.ToString(in.ContinuationToken) {
			keys = append(keys, k)
		}
	}
	sort.Strings(keys)
	out := &s3.ListObjectsV2Output{}
	if len(keys) > 2 {
		keys = keys[:2]
		out.IsTruncated = aws.Bool(true)
		out.NextContinuationToken = aws.String(keys[1])
	}
	for _, k := range keys {
		out.Contents = append(out.Contents, types.Object{Key: aws.String(k)})
	}
	return out, nil
}

func TestBackends(t *testing.T) {
	backends := map[string]func(t *testing.T) Backend{
		"mem": func(t *testing.T) Backend { return NewMemBackend() },
		"pebble": func(t *testing.T) Backend {
			b, err := NewPebbleBackend(PebbleConfig{Dir: t.TempDir()})
			if err != nil {
				t.Fatal(err)
			}
			return b
		},
		"s3": func(t *testing.T) Backend {
			b, err := NewS3Backend(S3Config{Bucket: "b", Prefix: "p", Client: &fakeS3{objs: map[string][]byte{}}})
			if err != nil {
				t.Fatal(err)
			}
			return b
		},
	}

	for name, mk := range backends {
		t.Run(name, func(t *testing.T) {
			s, err := NewWithBackend(mk(t))
			if err != nil {
				t.Fatal(err)
			}
			defer s.Close()
			ctx := context.Background()

			for _, id := range []string{"a", "b/c", "d"} {
				if _, err := s.Put(ctx, PutDocRequest{Index: "idx 1", ID: id, Data: []byte(`{"x":1}`)}); err != nil {
					t.Fatal(err)
				}
			}
			d, _ := s.Put(ctx, PutDocRequest{Index: "idx 1", ID: "a", Data: []byte(`{"x":2}`)})
			if d.Revision != 2 {
				t.Fatalf("revision = %d, want 2", d.Revision)
			}
			if s.Count("idx 1") != 3 {
				t.Fatalf("count = %d, want 3", s.Count("idx 1"))
			}
			if got, ok := s.Get("idx 1", "b/c"); !ok || string(got.Data) != `{"x":1}` {
				t.Fatalf("get b/c = %v, %v", got, ok)
			}
			if _, ok := s.Get("idx 1", "zzz"); ok {
				t.Fatal("unexpected doc")
			}

			resp, err := s.Batch(ctx, BatchDocsRequest{
				Index:   "idx 1",
				Puts:    []PutDocRequest{{ID: "e", Data: []byte(`1`)}, {ID: "e", Data: []byte(`2`)}},
				Deletes: []DeleteDocRequest{{ID: "a"}, {ID: "nope"}},
			})
			if err != nil || resp.PutCount != 2 || resp.DeleteCount != 1 {
				t.Fatalf("batch = %+v, %v", resp, err)
			}
			if e, _ := s.Get("idx 1", "e"); e.Revision != 2 {
				t.Fatalf("batch dup revision = %d, want 2", e.Revision)
			}
			if s.Count("idx 1") != 3 { // b/c, d, e
				t.Fatalf("count after batch = %d, want 3", s.Count("idx 1"))
			}

			snap, err := s.Snapshot()
			if err != nil {
				t.Fatal(err)
			}
			if _, err := s.Put(ctx, PutDocRequest{Index: "other", ID: "z", Data: []byte(`1`)}); err != nil {
				t.Fatal(err)
			}
			if err := s.Restore(snap); err != nil {
				t.Fatal(err)
			}
			if s.Count("other") != 0 || s.Count("idx 1") != 3 || len(s.ListIndexes()) != 1 {
				t.Fatalf("bad restore: %v", s.ListIndexes())
			}

			if ok, _ := s.Delete(ctx, DeleteDocRequest{Index: "idx 1", ID: "d"}); !ok {
				t.Fatal("delete failed")
			}
			if s.Count("idx 1") != 2 || len(s.AllDocs("idx 1")) != 2 {
				t.Fatalf("count after delete = %d", s.Count("idx 1"))
			}
		})
	}
}

func TestPebblePersistence(t *testing.T) {
	dir := t.TempDir()
	s, err := Open(Config{Type: BackendPebble, Pebble: PebbleConfig{Dir: dir}})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := s.Put(context.Background(), PutDocRequest{Index: "i", ID: "1", Data: []byte(`{}`)}); err != nil {
		t.Fatal(err)
	}
	s.Close()

	s, err = Open(Config{Type: BackendPebble, Pebble: PebbleConfig{Dir: dir}})
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	if s.Count("i") != 1 {
		t.Fatalf("count after reopen = %d", s.Count("i"))
	}
	if _, ok := s.Get("i", "1"); !ok {
		t.Fatal("doc lost after reopen")
	}
}

func TestOpenUnknownBackend(t *testing.T) {
	if _, err := Open(Config{Type: "nope"}); err == nil {
		t.Fatal("expected error")
	}
}
