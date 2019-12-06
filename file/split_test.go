package file

import (
	"context"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethersphere/swarm/bmt"
	"github.com/ethersphere/swarm/file/hasher"
	"github.com/ethersphere/swarm/file/store"
	"github.com/ethersphere/swarm/log"
	"github.com/ethersphere/swarm/param"
	"github.com/ethersphere/swarm/storage"
	"github.com/ethersphere/swarm/testutil"
	"golang.org/x/crypto/sha3"
)

const (
	sectionSize = 32
	branches    = 128
	chunkSize   = 4096
)

func init() {
	testutil.Init()
}

var (
	errFunc = func(err error) {
		log.Error("split writer pipeline error", "err", err)
	}
)

// TestSplit creates a Splitter with a reader with one chunk of serial data and
// a Hasher as the underlying param.SectionWriter
// It verifies the returned result
func TestSplit(t *testing.T) {
	poolAsync := bmt.NewTreePool(sha3.NewLegacyKeccak256, branches, bmt.PoolSize*128)
	refHashFunc := func(_ context.Context) param.SectionWriter {
		return bmt.New(poolAsync).NewAsyncWriter(false)
	}
	h := hasher.New(refHashFunc)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	h.Init(ctx, errFunc)

	r, _ := testutil.SerialData(chunkSize, 255, 0)
	s := NewSplitter(r, h)
	ref, err := s.Split()
	if err != nil {
		t.Fatal(err)
	}
	refHex := hexutil.Encode(ref)
	correctRefHex := "0xc10090961e7682a10890c334d759a28426647141213abda93b096b892824d2ef"
	if refHex != correctRefHex {
		t.Fatalf("split, expected %s, got %s", correctRefHex, refHex)
	}
}

// TestSplitWithDataFileStore verifies chunk.Store sink result for data hashing
func TestSplitWithDataFileStore(t *testing.T) {
	poolAsync := bmt.NewTreePool(sha3.NewLegacyKeccak256, branches, bmt.PoolSize*128)
	refHashFunc := func(_ context.Context) param.SectionWriter {
		return bmt.New(poolAsync).NewAsyncWriter(false)
	}

	ctx, cancel := context.WithTimeout(context.Background(), time.Millisecond)
	defer cancel()
	chunkStore := &storage.FakeChunkStore{}
	storeFunc := func(_ context.Context) param.SectionWriter {
		h := store.New(chunkStore, refHashFunc)
		h.Init(ctx, errFunc)
		return h
	}

	h := hasher.New(storeFunc)
	h.Init(ctx, errFunc)

	r, _ := testutil.SerialData(chunkSize, 255, 0)
	s := NewSplitter(r, h)
	ref, err := s.Split()
	if err != nil {
		t.Fatal(err)
	}
	time.Sleep(time.Second)
	refHex := hexutil.Encode(ref)
	correctRefHex := "0xc10090961e7682a10890c334d759a28426647141213abda93b096b892824d2ef"
	if refHex != correctRefHex {
		t.Fatalf("split, expected %s, got %s", correctRefHex, refHex)
	}
}

// TestSplitWithIntermediateFileStore verifies chunk.Store sink result for intermediate hashing
func TestSplitWithIntermediateFileStore(t *testing.T) {
	poolAsync := bmt.NewTreePool(sha3.NewLegacyKeccak256, branches, bmt.PoolSize*128)
	refHashFunc := func(_ context.Context) param.SectionWriter {
		return bmt.New(poolAsync).NewAsyncWriter(false)
	}

	ctx, cancel := context.WithTimeout(context.Background(), time.Millisecond)
	defer cancel()
	chunkStore := &storage.FakeChunkStore{}
	storeFunc := func(_ context.Context) param.SectionWriter {
		h := store.New(chunkStore, refHashFunc)
		h.Init(ctx, errFunc)
		return h
	}

	h := hasher.New(storeFunc)
	h.Init(ctx, errFunc)

	r, _ := testutil.SerialData(chunkSize*2, 255, 0)
	s := NewSplitter(r, h)
	ref, err := s.Split()
	if err != nil {
		t.Fatal(err)
	}
	time.Sleep(time.Second)
	refHex := hexutil.Encode(ref)
	correctRefHex := "0x29a5fb121ce96194ba8b7b823a1f9c6af87e1791f824940a53b5a7efe3f790d9"
	if refHex != correctRefHex {
		t.Fatalf("split, expected %s, got %s", correctRefHex, refHex)
	}
}

// TestSplitWithBothFileStore verifies chunk.Store sink result for both data and intermediate hashing
func TestSplitWithBothFileStore(t *testing.T) {
	poolAsync := bmt.NewTreePool(sha3.NewLegacyKeccak256, branches, bmt.PoolSize*128)
	refHashFunc := func(_ context.Context) param.SectionWriter {
		return bmt.New(poolAsync).NewAsyncWriter(false)
	}

	ctx, cancel := context.WithTimeout(context.Background(), time.Millisecond)
	defer cancel()
	chunkStore := &storage.FakeChunkStore{}
	refStoreFunc := func(_ context.Context) param.SectionWriter {
		h := store.New(chunkStore, refHashFunc)
		h.Init(ctx, errFunc)
		return h
	}

	h := hasher.New(refStoreFunc)
	h.Init(ctx, errFunc)

	r, _ := testutil.SerialData(chunkSize*128, 255, 0)
	s := NewSplitter(r, h)
	ref, err := s.Split()
	if err != nil {
		t.Fatal(err)
	}
	time.Sleep(time.Second)
	refHex := hexutil.Encode(ref)
	correctRefHex := "0x3047d841077898c26bbe6be652a2ec590a5d9bd7cd45d290ea42511b48753c09"
	if refHex != correctRefHex {
		t.Fatalf("split, expected %s, got %s", correctRefHex, refHex)
	}
}
