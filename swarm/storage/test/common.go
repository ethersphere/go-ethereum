// Copyright 2016 The go-ethereum Authors
// This file is part of the go-ethereum library.
//
// The go-ethereum library is free software: you can redistribute it and/or modify
// it under the terms of the GNU Lesser General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// The go-ethereum library is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Lesser General Public License for more details.
//
// You should have received a copy of the GNU Lesser General Public License
// along with the go-ethereum library. If not, see <http://www.gnu.org/licenses/>.

package test

import (
	"bytes"
	"context"
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"sync"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/swarm/constants"
	"github.com/ethereum/go-ethereum/swarm/storage"
	"github.com/ethereum/go-ethereum/swarm/storage/ldbstore"
	"github.com/mattn/go-colorable"
)

var (
	loglevel   = flag.Int("loglevel", 3, "verbosity of logs")
	getTimeout = 30 * time.Second
)

func init() {
	flag.Parse()
	log.PrintOrigins(true)
	log.Root().SetHandler(log.LvlFilterHandler(log.Lvl(*loglevel), log.StreamHandler(colorable.NewColorableStderr(), log.TerminalFormat(true))))
}

type BrokenLimitedReader struct {
	lr    io.Reader
	errAt int
	off   int
	size  int
}

func BrokenLimitReader(data io.Reader, size int, errAt int) *BrokenLimitedReader {
	return &BrokenLimitedReader{
		lr:    data,
		errAt: errAt,
		size:  size,
	}
}

func (r *BrokenLimitedReader) Read(buf []byte) (int, error) {
	if r.off+len(buf) > r.errAt {
		return 0, fmt.Errorf("Broken reader")
	}
	r.off += len(buf)
	return r.lr.Read(buf)
}

func NewLDBStore(t *testing.T) (*ldbstore.LDBStore, func()) {
	dir, err := ioutil.TempDir("", "bzz-storage-test")
	if err != nil {
		t.Fatal(err)
	}
	log.Trace("memstore.tempdir", "dir", dir)

	ldbparams := ldbstore.NewLDBStoreParams(storage.NewDefaultStoreParams(), dir)
	db, err := ldbstore.NewLDBStore(ldbparams)
	if err != nil {
		t.Fatal(err)
	}

	cleanup := func() {
		db.Close()
		err := os.RemoveAll(dir)
		if err != nil {
			t.Fatal(err)
		}
	}

	return db, cleanup
}

func MputRandomChunks(store storage.ChunkStore, n int) ([]storage.Chunk, error) {
	return Mput(store, n, storage.GenerateRandomChunk)
}

func Mput(store storage.ChunkStore, n int, f func(i int64) storage.Chunk) (hs []storage.Chunk, err error) {
	// put to localstore and wait for stored channel
	// does not check delivery error state
	errc := make(chan error)
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Minute)
	defer cancel()
	for i := int64(0); i < int64(n); i++ {
		chunk := f(constants.DefaultChunkSize)
		go func() {
			select {
			case errc <- store.Put(ctx, &chunk):
			case <-ctx.Done():
			}
		}()
		hs = append(hs, chunk)
	}

	// wait for all chunks to be stored
	for i := 0; i < n; i++ {
		err := <-errc
		if err != nil {
			return nil, err
		}
	}
	return hs, nil
}

func Mget(store storage.ChunkStore, hs []storage.Address, f func(h storage.Address, chunk storage.Chunk) error) error {
	wg := sync.WaitGroup{}
	wg.Add(len(hs))
	errc := make(chan error)

	for _, k := range hs {
		go func(h storage.Address) {
			defer wg.Done()
			// TODO: write timeout with context
			chunk, err := store.Get(context.TODO(), h)
			if err != nil {
				errc <- err
				return
			}
			if f != nil {
				err = f(h, *chunk)
				if err != nil {
					errc <- err
					return
				}
			}
		}(k)
	}
	go func() {
		wg.Wait()
		close(errc)
	}()
	var err error
	timeout := 10 * time.Second
	select {
	case err = <-errc:
	case <-time.NewTimer(timeout).C:
		err = fmt.Errorf("timed out after %v", timeout)
	}
	return err
}

func testStoreRandom(m storage.ChunkStore, n int, t *testing.T) {
	chunks, err := MputRandomChunks(m, n)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	err = Mget(m, ChunkAddresses(chunks), nil)
	if err != nil {
		t.Fatalf("testStore failed: %v", err)
	}
}

func TestStoreCorrect(m storage.ChunkStore, n int, t *testing.T) {
	chunks, err := MputRandomChunks(m, n)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	f := func(h storage.Address, chunk storage.Chunk) error {
		if !bytes.Equal(h, chunk.Address()) {
			return fmt.Errorf("key does not match retrieved chunk Address")
		}
		hasher := storage.MakeHashFunc(storage.DefaultHash)()
		data := chunk.Data()
		hasher.ResetWithLength(data[:8])
		hasher.Write(data[8:])
		exp := hasher.Sum(nil)
		if !bytes.Equal(h, exp) {
			return fmt.Errorf("key is not hash of chunk data")
		}
		return nil
	}
	err = Mget(m, ChunkAddresses(chunks), f)
	if err != nil {
		t.Fatalf("testStore failed: %v", err)
	}
}

func BenchmarkStorePut(store storage.ChunkStore, n int, b *testing.B) {
	chunks := make([]storage.Chunk, n)
	i := 0
	f := func(dataSize int64) storage.Chunk {
		chunk := storage.GenerateRandomChunk(dataSize)
		chunks[i] = chunk
		i++
		return chunk
	}

	Mput(store, n, f)

	f = func(dataSize int64) storage.Chunk {
		chunk := chunks[i]
		i++
		return chunk
	}

	b.ReportAllocs()
	b.ResetTimer()

	for j := 0; j < b.N; j++ {
		i = 0
		Mput(store, n, f)
	}
}

func BenchmarkStoreGet(store storage.ChunkStore, n int, b *testing.B) {
	chunks, err := MputRandomChunks(store, n)
	if err != nil {
		b.Fatalf("expected no error, got %v", err)
	}
	b.ReportAllocs()
	b.ResetTimer()
	addrs := ChunkAddresses(chunks)
	for i := 0; i < b.N; i++ {
		err := Mget(store, addrs, nil)
		if err != nil {
			b.Fatalf("mget failed: %v", err)
		}
	}
}

// MapChunkStore is a very simple ChunkStore implementation to store chunks in a map in memory.
type MapChunkStore struct {
	chunks map[string]storage.Chunk
	mu     sync.RWMutex
}

func NewMapChunkStore() *MapChunkStore {
	return &MapChunkStore{
		chunks: make(map[string]storage.Chunk),
	}
}

func (m *MapChunkStore) Put(_ context.Context, ch *storage.Chunk) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.chunks[ch.Address().Hex()] = *ch
	return nil
}

func (m *MapChunkStore) Get(_ context.Context, ref storage.Address) (*storage.Chunk, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	chunk, ok := m.chunks[ref.Hex()]
	if !ok {
		return nil, storage.ErrChunkNotFound
	}
	return &chunk, nil
}

// Need to implement Has from SyncChunkStore
func (m *MapChunkStore) Has(ctx context.Context, ref storage.Address) bool {
	m.mu.RLock()
	defer m.mu.RUnlock()

	_, has := m.chunks[ref.Hex()]
	return has
}

func (m *MapChunkStore) Close() {
}

func ChunkAddresses(chunks []storage.Chunk) []storage.Address {
	addrs := make([]storage.Address, len(chunks))
	for i, ch := range chunks {
		addrs[i] = ch.Address()
	}
	return addrs
}

func TestPoFunc(k storage.Address) (ret uint8) {
	basekey := make([]byte, 32)
	return uint8(storage.Proximity(basekey, k[:]))
}
