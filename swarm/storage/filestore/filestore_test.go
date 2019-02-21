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

package filestore

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/swarm/constants"
	"github.com/ethereum/go-ethereum/swarm/storage"
	"github.com/ethereum/go-ethereum/swarm/storage/ldbstore"
	"github.com/ethereum/go-ethereum/swarm/storage/lstore"
	"github.com/ethereum/go-ethereum/swarm/storage/mock/mem"
	testi "github.com/ethereum/go-ethereum/swarm/storage/test"
	"github.com/ethereum/go-ethereum/swarm/testutil"
)

const testDataSize = 0x0001000

func TestFileStorerandom(t *testing.T) {
	testFileStoreRandom(false, t)
	testFileStoreRandom(true, t)
}

func testFileStoreRandom(toEncrypt bool, t *testing.T) {
	tdb, cleanup, err := newTestDbStore(false, false, 50000)
	defer cleanup()
	if err != nil {
		t.Fatalf("init dbStore failed: %v", err)
	}
	db := tdb.LDBStore
	//memStore := storage.NewMemStore(NewDefaultStoreParams(), db)
	localStore := &lstore.LocalStore{
		//memStore: memStore,
		DbStore: db,
	}

	fileStore := NewFileStore(localStore, NewFileStoreParams())

	// todo: wtf is this?
	defer os.RemoveAll("/tmp/bzz")

	slice := testutil.RandomBytes(1, testDataSize)
	ctx := context.TODO()
	key, wait, err := fileStore.Store(ctx, bytes.NewReader(slice), testDataSize, toEncrypt)
	if err != nil {
		t.Fatalf("Store error: %v", err)
	}
	err = wait(ctx)
	if err != nil {
		t.Fatalf("Store waitt error: %v", err.Error())
	}
	t.Logf("getting the key now %v", key)
	resultReader, isEncrypted := fileStore.Retrieve(context.TODO(), key)
	if isEncrypted != toEncrypt {
		t.Fatalf("isEncrypted expected %v got %v", toEncrypt, isEncrypted)
	}
	resultSlice := make([]byte, testDataSize)
	n, err := resultReader.ReadAt(resultSlice, 0)
	if err != io.EOF {
		t.Fatalf("Retrieve error: %v", err)
	}
	if n != testDataSize {
		t.Fatalf("Slice size error got %d, expected %d.", n, testDataSize)
	}
	if !bytes.Equal(slice, resultSlice) {
		t.Fatalf("Comparison error.")
	}
	ioutil.WriteFile("/tmp/slice.bzz.16M", slice, 0666)
	ioutil.WriteFile("/tmp/result.bzz.16M", resultSlice, 0666)
	//	localStore.memStore = NewMemStore(NewDefaultStoreParams(), db)
	resultReader, isEncrypted = fileStore.Retrieve(context.TODO(), key)
	if isEncrypted != toEncrypt {
		t.Fatalf("isEncrypted expected %v got %v", toEncrypt, isEncrypted)
	}
	for i := range resultSlice {
		resultSlice[i] = 0
	}
	n, err = resultReader.ReadAt(resultSlice, 0)
	if err != io.EOF {
		t.Fatalf("Retrieve error after removing memStore: %v", err)
	}
	if n != len(slice) {
		t.Fatalf("Slice size error after removing memStore got %d, expected %d.", n, len(slice))
	}
	if !bytes.Equal(slice, resultSlice) {
		t.Fatalf("Comparison error after removing memStore.")
	}
}

func TestFileStoreCapacity(t *testing.T) {
	testFileStoreCapacity(false, t)
	testFileStoreCapacity(true, t)
}

func testFileStoreCapacity(toEncrypt bool, t *testing.T) {
	tdb, cleanup, err := newTestDbStore(false, false, constants.DefaultLDBCapacity)

	defer cleanup()
	if err != nil {
		t.Fatalf("init dbStore failed: %v", err)
	}
	db := tdb.LDBStore
	//memStore := NewMemStore(NewDefaultStoreParams(), db)
	localStore := &lstore.LocalStore{
		//	memStore: memStore,
		DbStore: db,
	}
	fileStore := NewFileStore(localStore, NewFileStoreParams())
	slice := testutil.RandomBytes(1, testDataSize)
	ctx := context.TODO()
	key, wait, err := fileStore.Store(ctx, bytes.NewReader(slice), testDataSize, toEncrypt)
	if err != nil {
		t.Errorf("Store error: %v", err)
	}
	err = wait(ctx)
	if err != nil {
		t.Fatalf("Store error: %v", err)
	}
	resultReader, isEncrypted := fileStore.Retrieve(context.TODO(), key)
	if isEncrypted != toEncrypt {
		t.Fatalf("isEncrypted expected %v got %v", toEncrypt, isEncrypted)
	}
	resultSlice := make([]byte, len(slice))
	n, err := resultReader.ReadAt(resultSlice, 0)
	if err != io.EOF {
		t.Fatalf("Retrieve error: %v", err)
	}
	if n != len(slice) {
		t.Fatalf("Slice size error got %d, expected %d.", n, len(slice))
	}
	if !bytes.Equal(slice, resultSlice) {
		t.Fatalf("Comparison error.")
	}
	// Clear memStore
	//	memStore.setCapacity(0)
	// check whether it is, indeed, empty
	//	fileStore.ChunkStore = memStore
	//	resultReader, isEncrypted = fileStore.Retrieve(context.TODO(), key)
	//	if isEncrypted != toEncrypt {
	//		t.Fatalf("isEncrypted expected %v got %v", toEncrypt, isEncrypted)
	//	}
	//	if _, err = resultReader.ReadAt(resultSlice, 0); err == nil {
	//		t.Fatalf("Was able to read %d bytes from an empty memStore.", len(slice))
	//	}
	// check how it works with localStore
	fileStore.ChunkStore = localStore
	//	localStore.dbStore.setCapacity(0)
	resultReader, isEncrypted = fileStore.Retrieve(context.TODO(), key)
	if isEncrypted != toEncrypt {
		t.Fatalf("isEncrypted expected %v got %v", toEncrypt, isEncrypted)
	}
	for i := range resultSlice {
		resultSlice[i] = 0
	}
	n, err = resultReader.ReadAt(resultSlice, 0)
	if err != io.EOF {
		t.Fatalf("Retrieve error after clearing memStore: %v", err)
	}
	if n != len(slice) {
		t.Fatalf("Slice size error after clearing memStore got %d, expected %d.", n, len(slice))
	}
	if !bytes.Equal(slice, resultSlice) {
		t.Fatalf("Comparison error after clearing memStore.")
	}
}

// TestGetAllReferences only tests that GetAllReferences returns an expected
// number of references for a given file
func TestGetAllReferences(t *testing.T) {
	tdb, cleanup, err := newTestDbStore(false, false, constants.DefaultLDBCapacity)
	defer cleanup()
	if err != nil {
		t.Fatalf("init dbStore failed: %v", err)
	}
	db := tdb.LDBStore
	//	memStore := NewMemStore(NewDefaultStoreParams(), db)
	localStore := &lstore.LocalStore{
		//	memStore: memStore,
		DbStore: db,
	}
	fileStore := NewFileStore(localStore, NewFileStoreParams())

	// testRuns[i] and expectedLen[i] are dataSize and expected length respectively
	testRuns := []int{1024, 8192, 16000, 30000, 1000000}
	expectedLens := []int{1, 3, 5, 9, 248}
	for i, r := range testRuns {
		slice := testutil.RandomBytes(1, r)

		addrs, err := fileStore.GetAllReferences(context.Background(), bytes.NewReader(slice), false)
		if err != nil {
			t.Fatal(err)
		}
		if len(addrs) != expectedLens[i] {
			t.Fatalf("Expected reference array length to be %d, but is %d", expectedLens[i], len(addrs))
		}
	}
}

type testDbStore struct {
	*ldbstore.LDBStore
	dir string
}

func newTestDbStore(mock bool, trusted bool, capacity uint64) (*testDbStore, func(), error) {
	dir, err := ioutil.TempDir("", "bzz-storage-test")
	if err != nil {
		return nil, func() {}, err
	}

	var db *ldbstore.LDBStore
	storeparams := storage.NewDefaultStoreParams()
	if capacity != storeparams.DbCapacity {
		storeparams.DbCapacity = capacity
	}
	params := ldbstore.NewLDBStoreParams(storeparams, dir)
	params.Po = testi.TestPoFunc

	if mock {
		globalStore := mem.NewGlobalStore()
		addr := common.HexToAddress("0x5aaeb6053f3e94c9b9a09f33669435e7ef1beaed")
		mockStore := globalStore.NewNodeStore(addr)

		db, err = ldbstore.NewMockDbStore(params, mockStore)
	} else {
		db, err = ldbstore.NewLDBStore(params)
	}

	cleanup := func() {
		if db != nil {
			db.Close()
		}
		err = os.RemoveAll(dir)
		if err != nil {
			panic(fmt.Sprintf("db cleanup failed: %v", err))
		}
	}

	return &testDbStore{db, dir}, cleanup, err
}
