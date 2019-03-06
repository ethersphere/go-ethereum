// Copyright 2019 The go-ethereum Authors
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

package localstore

import (
	"archive/tar"
	"context"
	"encoding/hex"
	"io"
	"io/ioutil"

	"github.com/ethereum/go-ethereum/swarm/chunk"
	"github.com/ethereum/go-ethereum/swarm/log"
	"github.com/ethereum/go-ethereum/swarm/shed"
)

// Export writes a tar structured data to the writer of
// all chunks in the retrieval data index. It returns the
// number of chunks exported.
func (db *DB) Export(w io.Writer) (count int64, err error) {
	tw := tar.NewWriter(w)
	defer tw.Close()

	err = db.retrievalDataIndex.Iterate(func(item shed.Item) (stop bool, err error) {
		hdr := &tar.Header{
			Name: hex.EncodeToString(item.Address),
			Mode: 0644,
			Size: int64(len(item.Data)),
		}
		if err := tw.WriteHeader(hdr); err != nil {
			return false, err
		}
		if _, err := tw.Write(item.Data); err != nil {
			return false, err
		}
		count++
		return false, nil
	}, nil)

	return count, err
}

// Import reads a tar structured data from the reader and
// stores chunks in the database. It returns the number of
// chunks imported.
func (db *DB) Import(r io.Reader) (count int64, err error) {
	tr := tar.NewReader(r)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	countC := make(chan int64)
	errC := make(chan error)
	go func() {
		for {
			hdr, err := tr.Next()
			if err != nil {
				if err == io.EOF {
					break
				}
				select {
				case errC <- err:
				case <-ctx.Done():
				}
			}

			if len(hdr.Name) != 64 {
				log.Warn("ignoring non-chunk file", "name", hdr.Name)
				continue
			}

			keybytes, err := hex.DecodeString(hdr.Name)
			if err != nil {
				log.Warn("ignoring invalid chunk file", "name", hdr.Name, "err", err)
				continue
			}

			data, err := ioutil.ReadAll(tr)
			if err != nil {
				select {
				case errC <- err:
				case <-ctx.Done():
				}
			}
			key := chunk.Address(keybytes)
			ch := chunk.NewChunk(key, data)

			go func() {
				select {
				case errC <- db.Put(ctx, chunk.ModePutUpload, ch):
				case <-ctx.Done():
				}
			}()

			count++
		}
		select {
		case countC <- count:
		case <-ctx.Done():
		}
	}()

	// wait for all chunks to be stored
	var i int64
	var total int64
	for {
		select {
		case err := <-errC:
			if err != nil {
				return count, err
			}
			i++
		case total = <-countC:
		case <-ctx.Done():
			return i, ctx.Err()
		}
		if total > 0 && i == total {
			return total, nil
		}
	}
}
