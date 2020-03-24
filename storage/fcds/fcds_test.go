// Copyright 2019 The Swarm Authors
// This file is part of the Swarm library.
//
// The Swarm library is free software: you can redistribute it and/or modify
// it under the terms of the GNU Lesser General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// The Swarm library is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Lesser General Public License for more details.
//
// You should have received a copy of the GNU Lesser General Public License
// along with the Swarm library. If not, see <http://www.gnu.org/licenses/>.

package fcds

import (
	"fmt"
	"io/ioutil"
	"math/rand"
	"os"
	"sync"
	"testing"
	"time"

	"github.com/ethersphere/swarm/chunk"
)

const (
	ConcurrentThreads = 128
)

func newBadger(b *testing.B) (db Storer, clean func()) {
	b.Helper()

	path, err := ioutil.TempDir("", "swarm-shed")
	if err != nil {
		b.Fatal(err)
	}

	fmt.Println("Creating new store in path ", path)
	db, err = NewBadgerStore(path)
	if err != nil {
		os.RemoveAll(path)
		b.Fatal(err)
	}
	return db, func() {
		db.Close()
		os.RemoveAll(path)
		fmt.Println("Removing path ", path)
	}
}

func getChunks(count int, chunkCache []chunk.Chunk) []chunk.Chunk {
	l := len(chunkCache)
	if l == 0 {
		chunkCache = make([]chunk.Chunk, count)
		for i := 0; i < count; i++ {
			chunkCache[i] = GenerateTestRandomChunk()
		}
		return chunkCache
	}
	if l < count {
		for i := 0; i < count-l; i++ {
			chunkCache = append(chunkCache, GenerateTestRandomChunk())
		}
		return chunkCache
	}
	return chunkCache[:count]
}

func init() {
	rand.Seed(time.Now().UnixNano())
}

func GenerateTestRandomChunk() chunk.Chunk {
	data := make([]byte, chunk.DefaultSize)
	rand.Read(data)
	key := make([]byte, 32)
	rand.Read(key)
	return chunk.NewChunk(key, data)
}

// Benchmarkings

func runBenchmark(b *testing.B, baseChunksCount int, writeChunksCount int, readChunksCount int, deleteChunksCount int, iterationCount int) {
	b.Helper()

	var writeElapsed time.Duration
	var readElapsed time.Duration
	var deleteElapsed time.Duration

	for i := 0; i < iterationCount; i++ {
		db, clean := newBadger(b)
		var basechunks []chunk.Chunk

		if baseChunksCount > 0 {
			basechunks = getChunks(baseChunksCount, basechunks)
			start := time.Now()
			sem := make(chan struct{}, ConcurrentThreads)
			var wg sync.WaitGroup
			wg.Add(baseChunksCount)
			for i, ch := range basechunks {
				sem <- struct{}{}
				go func(i int, ch chunk.Chunk) {
					defer func() {
						<-sem
						wg.Done()
					}()
					if err := db.Put(ch); err != nil {
						panic(err)
					}
				}(i, ch)
			}
			wg.Wait()
			elapsed := time.Since(start)
			fmt.Println("-- adding base chunks took, ", elapsed)
		}

		rand.Shuffle(baseChunksCount, func(i, j int) {
			basechunks[i], basechunks[j] = basechunks[j], basechunks[i]
		})

		var jobWg sync.WaitGroup
		if writeChunksCount > 0 {
			jobWg.Add(1)
			go func() {
				var writeChunks []chunk.Chunk
				writeChunks = getChunks(writeChunksCount, writeChunks)
				start := time.Now()
				sem := make(chan struct{}, ConcurrentThreads)
				var wg sync.WaitGroup
				wg.Add(writeChunksCount)
				for i, ch := range writeChunks {
					sem <- struct{}{}
					go func(i int, ch chunk.Chunk) {
						defer func() {
							<-sem
							wg.Done()
						}()
						if err := db.Put(ch); err != nil {
							panic(err)
						}
					}(i, ch)
				}
				wg.Wait()
				elapsed := time.Since(start)
				fmt.Println("-- writing chunks took , ", elapsed)
				writeElapsed += elapsed
				jobWg.Done()
			}()
		}

		if readChunksCount > 0 {
			jobWg.Add(1)
			go func() {
				errCount := 0
				start := time.Now()
				sem := make(chan struct{}, ConcurrentThreads*4)
				var wg sync.WaitGroup
				wg.Add(readChunksCount)
				for i, ch := range basechunks {
					if i >= readChunksCount {
						break
					}
					sem <- struct{}{}
					go func(i int, ch chunk.Chunk) {
						defer func() {
							<-sem
							wg.Done()
						}()
						_, err := db.Get(ch.Address())
						if err != nil {
							//panic(err)
							errCount++
						}
					}(i, ch)
				}
				wg.Wait()
				elapsed := time.Since(start)
				fmt.Println("-- reading chunks took , ", elapsed)
				readElapsed += elapsed
				jobWg.Done()
			}()
		}

		if deleteChunksCount > 0 {
			jobWg.Add(1)
			go func() {
				start := time.Now()
				sem := make(chan struct{}, ConcurrentThreads)
				var wg sync.WaitGroup
				wg.Add(deleteChunksCount)
				for i, ch := range basechunks {
					if i >= deleteChunksCount {
						break
					}
					sem <- struct{}{}
					go func(i int, ch chunk.Chunk) {
						defer func() {
							<-sem
							wg.Done()
						}()
						if err := db.Delete(ch.Address()); err != nil {
							panic(err)
						}
					}(i, ch)
				}
				wg.Wait()
				elapsed := time.Since(start)
				fmt.Println("-- deleting chunks took , ", elapsed)
				deleteElapsed += elapsed
				jobWg.Done()
			}()
		}

		jobWg.Wait()

		clean()
	}

	if writeElapsed > 0 {
		fmt.Println("- Average write  time : ", writeElapsed.Nanoseconds()/int64(iterationCount), " ns/op")
	}
	if readElapsed > 0 {
		fmt.Println("- Average read time : ", readElapsed.Nanoseconds()/int64(iterationCount), " ns/op")
	}
	if deleteElapsed > 0 {
		fmt.Println("- Average delete time : ", deleteElapsed.Nanoseconds()/int64(iterationCount), " ns/op")
	}
}

//func TestStorage (b *testing.T) {
//	runBenchmark(b, 0, 1000000, 0, 0)
//}

//func BenchmarkWriteOverClean_10000(t *testing.B) { runBenchmark(t, 0, 10000, 0, 0,8) }
//func BenchmarkWriteOverClean_100000(t *testing.B) { runBenchmark(t, 0, 100000, 0, 0, 6) }
func BenchmarkWriteOverClean_1000000(t *testing.B) { runBenchmark(t, 0, 1000000, 0, 0, 4) }

//func BenchmarkWriteOver1Million_10000(t *testing.B) { runBenchmark(t, 1000000, 10000, 0, 0,8) }
//func BenchmarkWriteOver1Million_100000(t *testing.B) { runBenchmark(t, 1000000, 100000, 0, 0,6) }
func BenchmarkWriteOver1Million_1000000(t *testing.B) { runBenchmark(t, 1000000, 1000000, 0, 0, 4) }

//func BenchmarkReadOver1Million_10000(t *testing.B) { runBenchmark(t, 1000000, 0, 10000, 0,8) }
//func BenchmarkReadOver1Million_100000(t *testing.B) { runBenchmark(t, 1000000, 0, 100000, 0, 6) }
func BenchmarkReadOver1Million_1000000(t *testing.B) { runBenchmark(t, 1000000, 0, 1000000, 0, 4) }

//func BenchmarkDeleteOver1Million_10000(t *testing.B) { runBenchmark(t, 1000000, 0, 0, 10000,8) }
//func BenchmarkDeleteOver1Million_100000(t *testing.B) { runBenchmark(t, 1000000, 0, 0, 100000,6) }
func BenchmarkDeleteOver1Million_1000000(t *testing.B) { runBenchmark(t, 1000000, 0, 0, 1000000, 4) }

//func BenchmarkWriteReadOver1Million_10000(t *testing.B) { runBenchmark(t, 1000000, 10000, 10000, 0,8) }
//func BenchmarkWriteReadOver1Million_100000(t *testing.B) { runBenchmark(t, 1000000, 100000, 100000, 0,6) }
func BenchmarkWriteReadOver1Million_1000000(t *testing.B) {
	runBenchmark(t, 1000000, 1000000, 1000000, 0, 4)
}

//func BenchmarkWriteReadDeleteOver1Million_10000(t *testing.B) { runBenchmark(t, 1000000, 10000, 10000, 10000,8) }
//func BenchmarkWriteReadDeleteOver1Million_100000(t *testing.B) { runBenchmark(t, 1000000, 100000, 100000, 100000,6) }
func BenchmarkWriteReadDeleteOver1Million_1000000(t *testing.B) {
	runBenchmark(t, 1000000, 1000000, 1000000, 1000000, 4)
}
