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

package storage

import (
	"bytes"
	"crypto"
	"crypto/rand"
	"encoding/binary"
	"fmt"
	"hash"
	"io"

	"github.com/ethereum/go-ethereum/bmt"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto/sha3"
)

const MaxPO = 7

type Hasher func() hash.Hash
type SwarmHasher func() SwarmHash

// Peer is the recorded as Source on the chunk
// should probably not be here? but network should wrap chunk object
type Peer interface{}

type Key []byte

func (x Key) Size() uint {
	return uint(len(x))
}

func (x Key) isEqual(y Key) bool {
	return bytes.Equal(x, y)
}

func (h Key) bits(i, j uint) uint {
	ii := i >> 3
	jj := i & 7
	if ii >= h.Size() {
		return 0
	}

	if jj+j <= 8 {
		return uint((h[ii] >> jj) & ((1 << j) - 1))
	}

	res := uint(h[ii] >> jj)
	jj = 8 - jj
	j -= jj
	for j != 0 {
		ii++
		if j < 8 {
			res += uint(h[ii]&((1<<j)-1)) << jj
			return res
		}
		res += uint(h[ii]) << jj
		jj += 8
		j -= 8
	}
	return res
}

func Proximity(one, other []byte) (ret int) {
	b := (MaxPO-1)/8 + 1
	if b > len(one) {
		b = len(one)
	}
	m := 8
	for i := 0; i < b; i++ {
		oxo := one[i] ^ other[i]
		if i == b-1 {
			m = MaxPO % 8
		}
		for j := 0; j < m; j++ {
			if (oxo>>uint8(7-j))&0x01 != 0 {
				return i*8 + j
			}
		}
	}
	return MaxPO
}

func IsZeroKey(key Key) bool {
	return len(key) == 0 || bytes.Equal(key, ZeroKey)
}

var ZeroKey = Key(common.Hash{}.Bytes())

func MakeHashFunc(hash string) SwarmHasher {
	switch hash {
	case "SHA256":
		return func() SwarmHash { return &HashWithLength{crypto.SHA256.New()} }
	case "SHA3":
		return func() SwarmHash { return &HashWithLength{sha3.NewKeccak256()} }
	case "BMT":
		return func() SwarmHash {
			hasher := sha3.NewKeccak256
			pool := bmt.NewTreePool(hasher, bmt.DefaultSegmentCount, bmt.DefaultPoolSize)
			return bmt.New(pool)
		}
	}
	return nil
}

func (key Key) Hex() string {
	return fmt.Sprintf("%064x", []byte(key[:]))
}

func (key Key) Log() string {
	if len(key[:]) < 8 {
		return fmt.Sprintf("%x", []byte(key[:]))
	}
	return fmt.Sprintf("%016x", []byte(key[:8]))
}

func (key Key) String() string {
	return fmt.Sprintf("%064x", []byte(key)[:])
}

func (key Key) MarshalJSON() (out []byte, err error) {
	return []byte(`"` + key.String() + `"`), nil
}

func (key *Key) UnmarshalJSON(value []byte) error {
	s := string(value)
	*key = make([]byte, 32)
	h := common.Hex2Bytes(s[1 : len(s)-1])
	copy(*key, h)
	return nil
}

type KeyCollection []Key

func NewKeyCollection(l int) KeyCollection {
	return make(KeyCollection, l)
}

func (c KeyCollection) Len() int {
	return len(c)
}

func (c KeyCollection) Less(i, j int) bool {
	return bytes.Compare(c[i], c[j]) == -1
}

func (c KeyCollection) Swap(i, j int) {
	c[i], c[j] = c[j], c[i]
}

// Chunk also serves as a request object passed to ChunkStores
// in case it is a retrieval request, Data is nil and Size is 0
// Note that Size is not the size of the data chunk, which is Data.Size()
// but the size of the subtree encoded in the chunk
// 0 if request, to be supplied by the dpa
type Chunk struct {
	Key   Key    // always
	SData []byte // nil if request, to be supplied by dpa
	Size  int64  // size of the data covered by the subtree encoded in this chunk
	//Source   Peer           // peer
	C        chan bool // to signal data delivery by the dpa
	ReqC     chan bool // to signal the request done
	dbStored chan bool // never remove a chunk from memStore before it is written to dbStore
	errored  bool      // flag which is set when the chunk request has errored or timeouted
}

func NewChunk(key Key, reqC chan bool) *Chunk {
	return &Chunk{Key: key, ReqC: reqC, dbStored: make(chan bool)}
}

func (c *Chunk) WaitToStore() {
	<-c.dbStored
}

func FakeChunk(size int64, count int, chunks []*Chunk) int {
	var i int
	hasher := MakeHashFunc(SHA3Hash)()
	chunksize := getDefaultChunkSize()
	if size > chunksize {
		size = chunksize
	}

	for i = 0; i < count; i++ {
		hasher.Reset()
		chunks[i].SData = make([]byte, size)
		rand.Read(chunks[i].SData)
		binary.LittleEndian.PutUint64(chunks[i].SData[:8], uint64(size))
		hasher.Write(chunks[i].SData)
		chunks[i].Key = make([]byte, 32)
		copy(chunks[i].Key, hasher.Sum(nil))
	}

	return i
}

func getDefaultChunkSize() int64 {
	return DefaultBranches * int64(MakeHashFunc(SHA3Hash)().Size())

}

/*
The ChunkStore interface is implemented by :

- MemStore: a memory cache
- DbStore: local disk/db store
- LocalStore: a combination (sequence of) memStore and dbStore
- NetStore: cloud storage abstraction layer
- DPA: local requests for swarm storage and retrieval
*/
type ChunkStore interface {
	Put(*Chunk) // effectively there is no error even if there is an error
	Get(Key) (*Chunk, error)
	Close()
}

/*
Chunker is the interface to a component that is responsible for disassembling and assembling larger data and indended to be the dependency of a DPA storage system with fixed maximum chunksize.

It relies on the underlying chunking model.

When calling Split, the caller provides a channel (chan *Chunk) on which it receives chunks to store. The DPA delegates to storage layers (implementing ChunkStore interface).

Split returns an error channel, which the caller can monitor.
After getting notified that all the data has been split (the error channel is closed), the caller can safely read or save the root key. Optionally it times out if not all chunks get stored or not the entire stream of data has been processed. By inspecting the errc channel the caller can check if any explicit errors (typically IO read/write failures) occurred during splitting.

When calling Join with a root key, the caller gets returned a seekable lazy reader. The caller again provides a channel on which the caller receives placeholder chunks with missing data. The DPA is supposed to forward this to the chunk stores and notify the chunker if the data has been delivered (i.e. retrieved from memory cache, disk-persisted db or cloud based swarm delivery). As the seekable reader is used, the chunker then puts these together the relevant parts on demand.
*/
type Splitter interface {
	/*
	   When splitting, data is given as a SectionReader, and the key is a hashSize long byte slice (Key), the root hash of the entire content will fill this once processing finishes.
	   New chunks to store are coming to caller via the chunk storage channel, which the caller provides.
	   wg is a Waitgroup (can be nil) that can be used to block until the local storage finishes
	   The caller gets returned an error channel, if an error is encountered during splitting, it is fed to errC error channel.
	   A closed error signals process completion at which point the key can be considered final if there were no errors.
	*/
	Split(io.Reader, int64, chan *Chunk) (Key, func(), error)

	/* This is the first step in making files mutable (not chunks)..
	   Append allows adding more data chunks to the end of the already existsing file.
	   The key for the root chunk is supplied to load the respective tree.
	   Rest of the parameters behave like Split.
	*/
	Append(Key, io.Reader, chan *Chunk) (Key, func(), error)
}

type Joiner interface {
	/*
	   Join reconstructs original content based on a root key.
	   When joining, the caller gets returned a Lazy SectionReader, which is
	   seekable and implements on-demand fetching of chunks as and where it is read.
	   New chunks to retrieve are coming to caller via the Chunk channel, which the caller provides.
	   If an error is encountered during joining, it appears as a reader error.
	   The SectionReader.
	   As a result, partial reads from a document are possible even if other parts
	   are corrupt or lost.
	   The chunks are not meant to be validated by the chunker when joining. This
	   is because it is left to the DPA to decide which sources are trusted.
	*/
	Join(key Key, chunkC chan *Chunk, depth int) LazySectionReader
}

type Chunker interface {
	Joiner
	Splitter
	// returns the key length
	// KeySize() int64
}

// Size, Seek, Read, ReadAt
type LazySectionReader interface {
	Size(chan bool) (int64, error)
	io.Seeker
	io.Reader
	io.ReaderAt
}

type LazyTestSectionReader struct {
	*io.SectionReader
}

func (self *LazyTestSectionReader) Size(chan bool) (int64, error) {
	return self.SectionReader.Size(), nil
}
