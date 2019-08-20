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

package localstore

import (
	"context"
	"fmt"
	"time"

	"github.com/ethereum/go-ethereum/metrics"
	"github.com/ethersphere/swarm/chunk"
	"github.com/ethersphere/swarm/shed"
	"github.com/syndtr/goleveldb/leveldb"
)

// Put stores Chunks to database and depending
// on the Putter mode, it updates required indexes.
// Put is required to implement chunk.Store
// interface.
func (db *DB) Put(ctx context.Context, mode chunk.ModePut, chs ...chunk.Chunk) (exist []bool, err error) {
	metricName := fmt.Sprintf("localstore.Put.%s", mode)

	metrics.GetOrRegisterCounter(metricName, nil).Inc(1)
	defer totalTimeMetric(metricName, time.Now())

	exist, err = db.put(mode, chs...)
	if err != nil {
		metrics.GetOrRegisterCounter(metricName+".error", nil).Inc(1)
	}
	return exist, err
}

// put stores Chunks to database and updates other
// indexes. It acquires lockAddr to protect two calls
// of this function for the same address in parallel.
// Item fields Address and Data must not be
// with their nil values.
func (db *DB) put(mode chunk.ModePut, chs ...chunk.Chunk) (exist []bool, err error) {
	// protect parallel updates
	db.batchMu.Lock()
	defer db.batchMu.Unlock()

	batch := new(leveldb.Batch)

	// variables that provide information for operations
	// to be done after write batch function successfully executes
	var gcSizeChange int64                      // number to add or subtract from gcSize
	var triggerPushFeed bool                    // signal push feed subscriptions to iterate
	triggerPullFeed := make(map[uint8]struct{}) // signal pull feed subscriptions to iterate

	exist = make([]bool, len(chs))

	// A lazy populated map of bin ids to properly set
	// BinID values for new chunks based on initial value from database
	// and incrementing them.
	// Values from this map are stored with the batch
	binIDs := make(map[uint8]uint64)

	switch mode {
	case chunk.ModePutRequest:
		for i, ch := range chs {
			exists, c, err := db.putRequest(batch, binIDs, chunkToItem(ch))
			if err != nil {
				return nil, err
			}
			exist[i] = exists
			gcSizeChange += c
		}

	case chunk.ModePutUpload:
		for i, ch := range chs {
			exists, err := db.putUpload(batch, binIDs, chunkToItem(ch))
			if err != nil {
				return nil, err
			}
			exist[i] = exists
			if !exists {
				// chunk is new so, trigger subscription feeds
				// after the batch is successfully written
				triggerPullFeed[db.po(ch.Address())] = struct{}{}
				triggerPushFeed = true
			}
		}

	case chunk.ModePutSync:
		for i, ch := range chs {
			exists, err := db.putSync(batch, binIDs, chunkToItem(ch))
			if err != nil {
				return nil, err
			}
			exist[i] = exists
			if !exists {
				// chunk is new so, trigger pull subscription feed
				// after the batch is successfully written
				triggerPullFeed[db.po(ch.Address())] = struct{}{}
			}
		}

	default:
		return nil, ErrInvalidMode
	}

	for po, id := range binIDs {
		db.binIDs.PutInBatch(batch, uint64(po), id)
	}

	err = db.incGCSizeInBatch(batch, gcSizeChange)
	if err != nil {
		return nil, err
	}

	err = db.shed.WriteBatch(batch)
	if err != nil {
		return nil, err
	}
	for po := range triggerPullFeed {
		db.triggerPullSubscriptions(po)
	}
	if triggerPushFeed {
		db.triggerPushSubscriptions()
	}
	return exist, nil
}

// putRequest adds an Item to the batch by updating required indexes:
//  - put to indexes: retrieve, gc
//  - it does not enter the syncpool
// The batch can be written to the database.
// Provided batch and binID map are updated.
func (db *DB) putRequest(batch *leveldb.Batch, binIDs map[uint8]uint64, item shed.Item) (exists bool, gcSizeChange int64, err error) {
	// check if the chunk already is in the database
	// as gc index is updated
	i, err := db.retrievalAccessIndex.Get(item)
	switch err {
	case nil:
		exists = true
		item.AccessTimestamp = i.AccessTimestamp
	case leveldb.ErrNotFound:
		exists = false
		// no chunk accesses
	default:
		return false, 0, err
	}
	i, err = db.retrievalDataIndex.Get(item)
	switch err {
	case nil:
		exists = true
		item.StoreTimestamp = i.StoreTimestamp
		item.BinID = i.BinID
	case leveldb.ErrNotFound:
		// no chunk accesses
		exists = false
	default:
		return false, 0, err
	}
	if item.AccessTimestamp != 0 {
		// delete current entry from the gc index
		db.gcIndex.DeleteInBatch(batch, item)
		gcSizeChange--
	}
	if item.StoreTimestamp == 0 {
		item.StoreTimestamp = now()
	}
	if item.BinID == 0 {
		item.BinID, err = db.incBinID(binIDs, db.po(item.Address))
		if err != nil {
			return false, 0, err
		}
	}
	// update access timestamp
	item.AccessTimestamp = now()
	// update retrieve access index
	db.retrievalAccessIndex.PutInBatch(batch, item)
	// add new entry to gc index
	db.gcIndex.PutInBatch(batch, item)
	gcSizeChange++

	db.retrievalDataIndex.PutInBatch(batch, item)

	return exists, gcSizeChange, nil
}

// putRequest adds an Item to the batch by updating required indexes:
//  - put to indexes: retrieve, push, pull
// The batch can be written to the database.
// Provided batch and binID map are updated.
func (db *DB) putUpload(batch *leveldb.Batch, binIDs map[uint8]uint64, item shed.Item) (exists bool, err error) {
	exists, err = db.retrievalDataIndex.Has(item)
	if err != nil {
		return false, err
	}
	if exists {
		return true, nil
	}

	item.StoreTimestamp = now()
	item.BinID, err = db.incBinID(binIDs, db.po(item.Address))
	if err != nil {
		return false, err
	}
	db.retrievalDataIndex.PutInBatch(batch, item)
	db.pullIndex.PutInBatch(batch, item)
	db.pushIndex.PutInBatch(batch, item)
	return false, nil
}

// putRequest adds an Item to the batch by updating required indexes:
//  - put to indexes: retrieve, pull
// The batch can be written to the database.
// Provided batch and binID map are updated.
func (db *DB) putSync(batch *leveldb.Batch, binIDs map[uint8]uint64, item shed.Item) (exists bool, err error) {
	exists, err = db.retrievalDataIndex.Has(item)
	if err != nil {
		return false, err
	}
	if exists {
		return true, nil
	}

	item.StoreTimestamp = now()
	item.BinID, err = db.incBinID(binIDs, db.po(item.Address))
	if err != nil {
		return false, err
	}
	db.retrievalDataIndex.PutInBatch(batch, item)
	db.pullIndex.PutInBatch(batch, item)
	return false, nil
}

// incBinID is a helper function for db.put* methods that increments bin id
// based on the current value in the database. This function must be called under
// a db.batchMu lock. Provided binID map is updated.
func (db *DB) incBinID(binIDs map[uint8]uint64, po uint8) (id uint64, err error) {
	if _, ok := binIDs[po]; !ok {
		binIDs[po], err = db.binIDs.Get(uint64(po))
		if err != nil {
			return 0, err
		}
	}
	binIDs[po]++
	return binIDs[po], nil
}
