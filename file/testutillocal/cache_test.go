package testutillocal

import (
	"bytes"
	"context"
	"testing"

	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethersphere/swarm/bmt"
	"github.com/ethersphere/swarm/file/hasher"
	"github.com/ethersphere/swarm/param"
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

func TestCache(t *testing.T) {
	c := NewCache()
	c.Init(context.Background(), func(error) {})
	_, data := testutil.SerialData(chunkSize, 255, 0)
	c.Write(0, data)
	cachedData := c.Get(0)
	if !bytes.Equal(cachedData, data) {
		t.Fatalf("cache data; expected %x, got %x", data, cachedData)
	}
}

func TestCacheLink(t *testing.T) {
	poolSync := bmt.NewTreePool(sha3.NewLegacyKeccak256, branches, bmt.PoolSize)
	dataHashFunc := func() param.SectionWriter {
		return hasher.NewBMTSyncSectionWriter(bmt.New(poolSync))
	}

	c := NewCache()
	c.Init(context.Background(), func(error) {})
	c.Link(dataHashFunc)
	_, data := testutil.SerialData(chunkSize, 255, 0)
	c.Write(0, data)
	span := bmt.LengthToSpan(chunkSize)
	ref := c.Sum(nil, chunkSize, span)
	refHex := hexutil.Encode(ref)
	correctRefHex := "0xc10090961e7682a10890c334d759a28426647141213abda93b096b892824d2ef"
	if refHex != correctRefHex {
		t.Fatalf("cache link; expected %s, got %s", correctRefHex, refHex)
	}
}