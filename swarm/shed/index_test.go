// Copyright 2018 The go-ethereum Authors
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

package shed

import (
	"bytes"
	"context"
	"encoding/binary"
	"fmt"
	"sort"
	"sync"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/swarm/storage"
	"github.com/syndtr/goleveldb/leveldb"
)

// Index functions for the index that is used in tests in this file.
var (
	// Address->StoreTimestamp|Data
	retrievalIndexFuncs = IndexFuncs{
		EncodeKey: func(fields IndexItem) (key []byte, err error) {
			return fields.Address, nil
		},
		DecodeKey: func(key []byte) (e IndexItem, err error) {
			e.Address = key
			return e, nil
		},
		EncodeValue: func(fields IndexItem) (value []byte, err error) {
			b := make([]byte, 8)
			binary.BigEndian.PutUint64(b, uint64(fields.StoreTimestamp))
			value = append(b, fields.Data...)
			return value, nil
		},
		DecodeValue: func(value []byte) (e IndexItem, err error) {
			e.StoreTimestamp = int64(binary.BigEndian.Uint64(value[:8]))
			e.Data = value[8:]
			return e, nil
		},
	}
	// StoredTimestamp|Address->Data
	storedIndexFuncs = IndexFuncs{
		EncodeKey: func(fields IndexItem) (key []byte, err error) {
			b := make([]byte, 8, 8+len(fields.Address))
			binary.BigEndian.PutUint64(b[:8], uint64(fields.StoreTimestamp))
			key = append(b, fields.Address...)
			return key, nil
		},
		DecodeKey: func(key []byte) (e IndexItem, err error) {
			e.StoreTimestamp = int64(binary.BigEndian.Uint64(key[:8]))
			e.Address = key[8:]
			return e, nil
		},
		EncodeValue: func(fields IndexItem) (value []byte, err error) {
			return fields.Data, nil
		},
		DecodeValue: func(value []byte) (e IndexItem, err error) {
			e.Data = value
			return e, nil
		},
	}
)

// TestIndex validates put, get and delete functions of the Index implementation.
func TestIndex(t *testing.T) {
	db, cleanupFunc := newTestDB(t)
	defer cleanupFunc()

	index, err := db.NewIndex("retrieval", retrievalIndexFuncs)
	if err != nil {
		t.Fatal(err)
	}

	t.Run("put", func(t *testing.T) {
		want := IndexItem{
			Address:        []byte("put-hash"),
			Data:           []byte("DATA"),
			StoreTimestamp: time.Now().UTC().UnixNano(),
		}

		err := index.Put(want)
		if err != nil {
			t.Fatal(err)
		}
		got, err := index.Get(IndexItem{
			Address: want.Address,
		})
		if err != nil {
			t.Fatal(err)
		}
		checkIndexItem(t, got, want)

		t.Run("overwrite", func(t *testing.T) {
			want := IndexItem{
				Address:        []byte("put-hash"),
				Data:           []byte("New DATA"),
				StoreTimestamp: time.Now().UTC().UnixNano(),
			}

			err = index.Put(want)
			if err != nil {
				t.Fatal(err)
			}
			got, err := index.Get(IndexItem{
				Address: want.Address,
			})
			if err != nil {
				t.Fatal(err)
			}
			checkIndexItem(t, got, want)
		})
	})

	t.Run("put in batch", func(t *testing.T) {
		want := IndexItem{
			Address:        []byte("put-in-batch-hash"),
			Data:           []byte("DATA"),
			StoreTimestamp: time.Now().UTC().UnixNano(),
		}

		batch := new(leveldb.Batch)
		index.PutInBatch(batch, want)
		err := db.WriteBatch(batch)
		if err != nil {
			t.Fatal(err)
		}
		got, err := index.Get(IndexItem{
			Address: want.Address,
		})
		if err != nil {
			t.Fatal(err)
		}
		checkIndexItem(t, got, want)

		t.Run("overwrite", func(t *testing.T) {
			want := IndexItem{
				Address:        []byte("put-in-batch-hash"),
				Data:           []byte("New DATA"),
				StoreTimestamp: time.Now().UTC().UnixNano(),
			}

			batch := new(leveldb.Batch)
			index.PutInBatch(batch, want)
			db.WriteBatch(batch)
			if err != nil {
				t.Fatal(err)
			}
			got, err := index.Get(IndexItem{
				Address: want.Address,
			})
			if err != nil {
				t.Fatal(err)
			}
			checkIndexItem(t, got, want)
		})
	})

	t.Run("delete", func(t *testing.T) {
		want := IndexItem{
			Address:        []byte("delete-hash"),
			Data:           []byte("DATA"),
			StoreTimestamp: time.Now().UTC().UnixNano(),
		}

		err := index.Put(want)
		if err != nil {
			t.Fatal(err)
		}
		got, err := index.Get(IndexItem{
			Address: want.Address,
		})
		if err != nil {
			t.Fatal(err)
		}
		checkIndexItem(t, got, want)

		err = index.Delete(IndexItem{
			Address: want.Address,
		})
		if err != nil {
			t.Fatal(err)
		}

		wantErr := leveldb.ErrNotFound
		got, err = index.Get(IndexItem{
			Address: want.Address,
		})
		if err != wantErr {
			t.Fatalf("got error %v, want %v", err, wantErr)
		}
	})

	t.Run("delete in batch", func(t *testing.T) {
		want := IndexItem{
			Address:        []byte("delete-in-batch-hash"),
			Data:           []byte("DATA"),
			StoreTimestamp: time.Now().UTC().UnixNano(),
		}

		err := index.Put(want)
		if err != nil {
			t.Fatal(err)
		}
		got, err := index.Get(IndexItem{
			Address: want.Address,
		})
		if err != nil {
			t.Fatal(err)
		}
		checkIndexItem(t, got, want)

		batch := new(leveldb.Batch)
		index.DeleteInBatch(batch, IndexItem{
			Address: want.Address,
		})
		err = db.WriteBatch(batch)
		if err != nil {
			t.Fatal(err)
		}

		wantErr := leveldb.ErrNotFound
		got, err = index.Get(IndexItem{
			Address: want.Address,
		})
		if err != wantErr {
			t.Fatalf("got error %v, want %v", err, wantErr)
		}
	})
}

// TestIndex_iterate validates index iterator functions for correctness.
func TestIndex_iterate(t *testing.T) {
	db, cleanupFunc := newTestDB(t)
	defer cleanupFunc()

	index, err := db.NewIndex("retrieval", retrievalIndexFuncs)
	if err != nil {
		t.Fatal(err)
	}

	items := []IndexItem{
		{
			Address: []byte("iterate-hash-01"),
			Data:    []byte("data80"),
		},
		{
			Address: []byte("iterate-hash-03"),
			Data:    []byte("data22"),
		},
		{
			Address: []byte("iterate-hash-05"),
			Data:    []byte("data41"),
		},
		{
			Address: []byte("iterate-hash-02"),
			Data:    []byte("data84"),
		},
		{
			Address: []byte("iterate-hash-06"),
			Data:    []byte("data1"),
		},
	}
	batch := new(leveldb.Batch)
	for _, i := range items {
		index.PutInBatch(batch, i)
	}
	err = db.WriteBatch(batch)
	if err != nil {
		t.Fatal(err)
	}
	item04 := IndexItem{
		Address: []byte("iterate-hash-04"),
		Data:    []byte("data0"),
	}
	err = index.Put(item04)
	if err != nil {
		t.Fatal(err)
	}
	items = append(items, item04)

	sort.SliceStable(items, func(i, j int) bool {
		return bytes.Compare(items[i].Address, items[j].Address) < 0
	})

	t.Run("all", func(t *testing.T) {
		var i int
		err := index.IterateAll(func(item IndexItem) (stop bool, err error) {
			if i > len(items)-1 {
				return true, fmt.Errorf("got unexpected index item: %#v", item)
			}
			want := items[i]
			checkIndexItem(t, item, want)
			i++
			return false, nil
		})
		if err != nil {
			t.Fatal(err)
		}
	})

	t.Run("from", func(t *testing.T) {
		startIndex := 2
		i := startIndex
		err := index.IterateFrom(items[startIndex], func(item IndexItem) (stop bool, err error) {
			if i > len(items)-1 {
				return true, fmt.Errorf("got unexpected index item: %#v", item)
			}
			want := items[i]
			checkIndexItem(t, item, want)
			i++
			return false, nil
		})
		if err != nil {
			t.Fatal(err)
		}
	})

	t.Run("stop", func(t *testing.T) {
		var i int
		stopIndex := 3
		var count int
		err := index.IterateAll(func(item IndexItem) (stop bool, err error) {
			if i > len(items)-1 {
				return true, fmt.Errorf("got unexpected index item: %#v", item)
			}
			want := items[i]
			checkIndexItem(t, item, want)
			count++
			if i == stopIndex {
				return true, nil
			}
			i++
			return false, nil
		})
		if err != nil {
			t.Fatal(err)
		}
		wantItemsCount := stopIndex + 1
		if count != wantItemsCount {
			t.Errorf("got %v items, expected %v", count, wantItemsCount)
		}
	})

	t.Run("no overflow", func(t *testing.T) {
		secondIndex, err := db.NewIndex("second-index", retrievalIndexFuncs)
		if err != nil {
			t.Fatal(err)
		}

		secondIndexItem := IndexItem{
			Address: []byte("iterate-hash-10"),
			Data:    []byte("data-second"),
		}
		err = secondIndex.Put(secondIndexItem)
		if err != nil {
			t.Fatal(err)
		}

		var i int
		err = index.IterateAll(func(item IndexItem) (stop bool, err error) {
			if i > len(items)-1 {
				return true, fmt.Errorf("got unexpected index item: %#v", item)
			}
			want := items[i]
			checkIndexItem(t, item, want)
			i++
			return false, nil
		})
		if err != nil {
			t.Fatal(err)
		}

		i = 0
		err = secondIndex.IterateAll(func(item IndexItem) (stop bool, err error) {
			if i > 1 {
				return true, fmt.Errorf("got unexpected index item: %#v", item)
			}
			checkIndexItem(t, item, secondIndexItem)
			i++
			return false, nil
		})
		if err != nil {
			t.Fatal(err)
		}
	})
}

// TestIndex_NewSubscription tests one index subscription for iterations
// over existing keys and a newly saved.
func TestIndex_NewSubscription(t *testing.T) {
	db, cleanupFunc := newTestDB(t)
	defer cleanupFunc()

	index, err := db.NewIndex("stored", storedIndexFuncs)
	if err != nil {
		t.Fatal(err)
	}

	// put some items before the subscription is created and provide them for validation
	items := putItems(t, index, 10)

	// cursor counts the number of received items
	var cursor int
	// mu protects the cursor and wantItemsCount
	var mu sync.Mutex
	// wait signals that it is safe to check if
	// all items are iterated on.
	wait := make(chan struct{})

	// wantItemsCount is the expected number of items from subscription
	wantItemsCount := len(items)

	s, err := index.NewSubscription(context.Background(), func(item IndexItem) (stop bool, err error) {
		mu.Lock()
		defer mu.Unlock()

		// validate that the item is the one that is expected by ordering
		if !bytes.Equal(items[cursor].Address, item.Address) {
			return false, fmt.Errorf("got %v address %x, want %x", cursor, items[cursor].Address, item.Address)
		}

		// move the cursor (increase the count of received items)
		cursor++

		// if all expected items are received, signal for a check
		if cursor == wantItemsCount {
			wait <- struct{}{}
		}
		return
	})
	if err != nil {
		t.Fatal(err)
	}
	if s == nil {
		t.Fatal("subscription is nil")
	}
	defer s.Stop()

	t.Run("initial items", func(t *testing.T) {
		// wait until it is safe to check for the number of received items
		select {
		case <-s.Done():
			t.Fatalf("sunscription should not be done: %s", s.Err())
		case <-time.After(30 * time.Second):
			t.Fatalf("index subscription items not received")
		case <-wait:
		}

		mu.Lock()
		// get the current received items count
		gotItemsCount := cursor
		mu.Unlock()

		if gotItemsCount != wantItemsCount {
			t.Fatalf("got items %v, want %v", gotItemsCount, wantItemsCount)
		}
	})

	t.Run("put more items", func(t *testing.T) {
		// add more items after the subscription started
		items = append(items, putItems(t, index, 12)...)

		mu.Lock()
		// increment expected total number of items
		wantItemsCount = len(items)
		mu.Unlock()

		// wait for a second as no items should be received until
		// TriggerSubscriptions is called.
		select {
		case <-s.Done():
			t.Fatalf("sunscription should not be done: %s", s.Err())
		case <-time.After(time.Second):
		case <-wait:
			t.Fatalf("unexpected index subscriptions received")
		}

		mu.Lock()
		// get the current cursor
		gotItemsCount := cursor
		mu.Unlock()

		if gotItemsCount == wantItemsCount {
			t.Fatalf("got items %v, before triggering subscriptions", gotItemsCount)
		}
	})

	t.Run("trigger", func(t *testing.T) {
		index.TriggerSubscriptions()

		// wait until it is safe to check for the number of received items
		select {
		case <-s.Done():
			t.Fatalf("sunscription should not be done: %s", s.Err())
		case <-time.After(30 * time.Second):
			t.Fatalf("index subscription items not received")
		case <-wait:
		}

		mu.Lock()
		// get the current cursor
		gotItemsCount := cursor
		// increment expected total number of items
		wantItemsCount = len(items)
		mu.Unlock()

		if gotItemsCount != wantItemsCount {
			t.Fatalf("got items %v, want %v", gotItemsCount, wantItemsCount)
		}
	})
}

func putItems(t *testing.T, index Index, n int) []IndexItem {
	t.Helper()

	items := make([]IndexItem, 0)
	for i := 0; i < n; i++ {
		c := storage.GenerateRandomChunk(24)
		items = append(items, IndexItem{
			Address:        c.Address(),
			Data:           c.Data(),
			StoreTimestamp: time.Now().UTC().UnixNano(),
		})
	}

	for _, item := range items {
		err := index.Put(item)
		if err != nil {
			t.Fatal(err)
		}
	}
	return items
}

// checkIndexItem is a test helper function that compares if two Index items are the same.
func checkIndexItem(t *testing.T, got, want IndexItem) {
	t.Helper()

	if !bytes.Equal(got.Address, want.Address) {
		t.Errorf("got hash %q, expected %q", string(got.Address), string(want.Address))
	}
	if !bytes.Equal(got.Data, want.Data) {
		t.Errorf("got data %q, expected %q", string(got.Data), string(want.Data))
	}
	if got.StoreTimestamp != want.StoreTimestamp {
		t.Errorf("got store timestamp %v, expected %v", got.StoreTimestamp, want.StoreTimestamp)
	}
	if got.AccessTimestamp != want.AccessTimestamp {
		t.Errorf("got access timestamp %v, expected %v", got.AccessTimestamp, want.AccessTimestamp)
	}
}
