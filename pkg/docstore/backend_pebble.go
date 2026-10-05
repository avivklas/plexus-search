package docstore

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/cockroachdb/pebble"
)

// PebbleConfig configures the Pebble document backend.
type PebbleConfig struct {
	// Dir is the Pebble data directory (required).
	Dir string
	// Sync fsyncs every write batch. The Raft log already provides durability,
	// so this is off by default.
	Sync bool
}

// PebbleBackend stores documents in an embedded Pebble LSM.
// Key layout: <index> 0x00 <id>; value: JSON-encoded Document.
type PebbleBackend struct {
	db   *pebble.DB
	wopt *pebble.WriteOptions
}

// NewPebbleBackend opens (or creates) a Pebble database at cfg.Dir.
func NewPebbleBackend(cfg PebbleConfig) (*PebbleBackend, error) {
	if cfg.Dir == "" {
		return nil, errors.New("docstore pebble: dir is required")
	}
	db, err := pebble.Open(cfg.Dir, &pebble.Options{})
	if err != nil {
		return nil, fmt.Errorf("open pebble %s: %w", cfg.Dir, err)
	}
	w := pebble.NoSync
	if cfg.Sync {
		w = pebble.Sync
	}
	return &PebbleBackend{db: db, wopt: w}, nil
}

func pebbleKey(index, id string) []byte {
	k := make([]byte, 0, len(index)+1+len(id))
	k = append(k, index...)
	k = append(k, 0)
	return append(k, id...)
}

func splitPebbleKey(k []byte) (index, id string) {
	i := bytes.IndexByte(k, 0)
	if i < 0 {
		return string(k), ""
	}
	return string(k[:i]), string(k[i+1:])
}

func (p *PebbleBackend) Get(index, id string) (*Document, error) {
	v, closer, err := p.db.Get(pebbleKey(index, id))
	if errors.Is(err, pebble.ErrNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	defer closer.Close()
	var d Document
	if err := json.Unmarshal(v, &d); err != nil {
		return nil, fmt.Errorf("decode doc %s/%s: %w", index, id, err)
	}
	return &d, nil
}

func (p *PebbleBackend) Write(puts []*Document, deletes []DocKey) error {
	b := p.db.NewBatch()
	defer b.Close()
	for _, d := range puts {
		val, err := json.Marshal(d)
		if err != nil {
			return err
		}
		if err := b.Set(pebbleKey(d.Index, d.ID), val, nil); err != nil {
			return err
		}
	}
	for _, k := range deletes {
		if err := b.Delete(pebbleKey(k.Index, k.ID), nil); err != nil {
			return err
		}
	}
	return b.Commit(p.wopt)
}

func (p *PebbleBackend) Scan(fn func(*Document) error) error {
	it, err := p.db.NewIter(nil)
	if err != nil {
		return err
	}
	defer it.Close()
	for ok := it.First(); ok; ok = it.Next() {
		var d Document
		if err := json.Unmarshal(it.Value(), &d); err != nil {
			return err
		}
		if err := fn(&d); err != nil {
			return err
		}
	}
	return it.Error()
}

// Counts walks keys only (values are not decoded).
func (p *PebbleBackend) Counts() (map[string]int, error) {
	out := make(map[string]int)
	it, err := p.db.NewIter(nil)
	if err != nil {
		return nil, err
	}
	defer it.Close()
	for ok := it.First(); ok; ok = it.Next() {
		idx, _ := splitPebbleKey(it.Key())
		out[idx]++
	}
	return out, it.Error()
}

func (p *PebbleBackend) Reset() error {
	return p.db.DeleteRange([]byte{}, []byte{0xff, 0xff, 0xff, 0xff}, p.wopt)
}

func (p *PebbleBackend) Close() error { return p.db.Close() }
