package docstore

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/url"
	"strings"
	"sync"

	"github.com/aws/aws-sdk-go-v2/aws"
	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/s3/types"
)

// S3API is the subset of the S3 client used by S3Backend (allows fakes in tests).
type S3API interface {
	GetObject(ctx context.Context, in *s3.GetObjectInput, opts ...func(*s3.Options)) (*s3.GetObjectOutput, error)
	PutObject(ctx context.Context, in *s3.PutObjectInput, opts ...func(*s3.Options)) (*s3.PutObjectOutput, error)
	DeleteObject(ctx context.Context, in *s3.DeleteObjectInput, opts ...func(*s3.Options)) (*s3.DeleteObjectOutput, error)
	ListObjectsV2(ctx context.Context, in *s3.ListObjectsV2Input, opts ...func(*s3.Options)) (*s3.ListObjectsV2Output, error)
}

// S3Config configures the S3 document backend. Credentials come from the
// default AWS chain (env, shared config, IAM role...).
type S3Config struct {
	Bucket string // required
	Prefix string // optional key prefix, e.g. "plexus/docs"
	Region string // optional, falls back to AWS config
	// Endpoint overrides the S3 endpoint (MinIO, Ceph, localstack...).
	Endpoint string
	// PathStyle forces path-style addressing (required by most S3-compatible servers).
	PathStyle bool
	// Concurrency bounds parallel object requests per batch (default 16).
	Concurrency int
	// Client, if set, is used instead of constructing one.
	Client S3API
}

// S3Backend stores one JSON object per document at <prefix>/<index>/<id>
// (both path-escaped).
type S3Backend struct {
	client S3API
	bucket string
	prefix string
	conc   int
}

// NewS3Backend creates an S3 backend.
func NewS3Backend(cfg S3Config) (*S3Backend, error) {
	if cfg.Bucket == "" {
		return nil, errors.New("docstore s3: bucket is required")
	}
	client := cfg.Client
	if client == nil {
		opts := []func(*awsconfig.LoadOptions) error{}
		if cfg.Region != "" {
			opts = append(opts, awsconfig.WithRegion(cfg.Region))
		}
		awsCfg, err := awsconfig.LoadDefaultConfig(context.Background(), opts...)
		if err != nil {
			return nil, fmt.Errorf("load aws config: %w", err)
		}
		client = s3.NewFromConfig(awsCfg, func(o *s3.Options) {
			if cfg.Endpoint != "" {
				o.BaseEndpoint = aws.String(cfg.Endpoint)
			}
			o.UsePathStyle = cfg.PathStyle
		})
	}
	prefix := strings.Trim(cfg.Prefix, "/")
	if prefix != "" {
		prefix += "/"
	}
	conc := cfg.Concurrency
	if conc <= 0 {
		conc = 16
	}
	return &S3Backend{client: client, bucket: cfg.Bucket, prefix: prefix, conc: conc}, nil
}

func (s *S3Backend) key(index, id string) string {
	return s.prefix + url.PathEscape(index) + "/" + url.PathEscape(id)
}

func (s *S3Backend) parseKey(k string) (index, id string, ok bool) {
	rest := strings.TrimPrefix(k, s.prefix)
	i := strings.IndexByte(rest, '/')
	if i < 0 {
		return "", "", false
	}
	idx, err1 := url.PathUnescape(rest[:i])
	did, err2 := url.PathUnescape(rest[i+1:])
	if err1 != nil || err2 != nil {
		return "", "", false
	}
	return idx, did, true
}

func (s *S3Backend) Get(index, id string) (*Document, error) {
	out, err := s.client.GetObject(context.Background(), &s3.GetObjectInput{
		Bucket: aws.String(s.bucket),
		Key:    aws.String(s.key(index, id)),
	})
	if err != nil {
		var nf *types.NoSuchKey
		if errors.As(err, &nf) {
			return nil, nil
		}
		return nil, err
	}
	defer out.Body.Close()
	body, err := io.ReadAll(out.Body)
	if err != nil {
		return nil, err
	}
	var d Document
	if err := json.Unmarshal(body, &d); err != nil {
		return nil, fmt.Errorf("decode doc %s/%s: %w", index, id, err)
	}
	return &d, nil
}

// Write issues the object requests in parallel. S3 has no multi-object
// transactions, so a failed batch may be partially applied.
func (s *S3Backend) Write(puts []*Document, deletes []DocKey) error {
	ctx := context.Background()
	var (
		wg       sync.WaitGroup
		mu       sync.Mutex
		firstErr error
		sem      = make(chan struct{}, s.conc)
	)
	run := func(fn func() error) {
		sem <- struct{}{}
		wg.Add(1)
		go func() {
			defer wg.Done()
			defer func() { <-sem }()
			if err := fn(); err != nil {
				mu.Lock()
				if firstErr == nil {
					firstErr = err
				}
				mu.Unlock()
			}
		}()
	}
	for _, d := range puts {
		val, err := json.Marshal(d)
		if err != nil {
			wg.Wait()
			return err
		}
		key := s.key(d.Index, d.ID)
		run(func() error {
			_, err := s.client.PutObject(ctx, &s3.PutObjectInput{
				Bucket:      aws.String(s.bucket),
				Key:         aws.String(key),
				Body:        bytes.NewReader(val),
				ContentType: aws.String("application/json"),
			})
			return err
		})
	}
	for _, k := range deletes {
		key := s.key(k.Index, k.ID)
		run(func() error {
			_, err := s.client.DeleteObject(ctx, &s3.DeleteObjectInput{
				Bucket: aws.String(s.bucket),
				Key:    aws.String(key),
			})
			return err
		})
	}
	wg.Wait()
	return firstErr
}

func (s *S3Backend) listKeys(fn func(key string) error) error {
	var token *string
	for {
		out, err := s.client.ListObjectsV2(context.Background(), &s3.ListObjectsV2Input{
			Bucket:            aws.String(s.bucket),
			Prefix:            aws.String(s.prefix),
			ContinuationToken: token,
		})
		if err != nil {
			return err
		}
		for _, o := range out.Contents {
			if err := fn(aws.ToString(o.Key)); err != nil {
				return err
			}
		}
		if !aws.ToBool(out.IsTruncated) || out.NextContinuationToken == nil {
			return nil
		}
		token = out.NextContinuationToken
	}
}

func (s *S3Backend) Scan(fn func(*Document) error) error {
	return s.listKeys(func(k string) error {
		idx, id, ok := s.parseKey(k)
		if !ok {
			return nil
		}
		d, err := s.Get(idx, id)
		if err != nil || d == nil {
			return err
		}
		return fn(d)
	})
}

func (s *S3Backend) Counts() (map[string]int, error) {
	out := make(map[string]int)
	err := s.listKeys(func(k string) error {
		if idx, _, ok := s.parseKey(k); ok {
			out[idx]++
		}
		return nil
	})
	return out, err
}

func (s *S3Backend) Reset() error {
	var keys []DocKey
	if err := s.listKeys(func(k string) error {
		if idx, id, ok := s.parseKey(k); ok {
			keys = append(keys, DocKey{idx, id})
		}
		return nil
	}); err != nil {
		return err
	}
	return s.Write(nil, keys)
}

func (s *S3Backend) Close() error { return nil }
