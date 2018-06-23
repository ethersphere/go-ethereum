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

// Handler is the API for Mutable Resources
// It enables creating, updating, syncing and retrieving resources and their update data
package mru

import (
	"bytes"
	"context"
	"sync"
	"time"
	"unsafe"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/swarm/log"
	"github.com/ethereum/go-ethereum/swarm/storage"
)

type Handler struct {
	chunkStore        *storage.NetStore
	HashSize          int
	timestampProvider timestampProvider
	resources         map[uint64]*resource
	resourceLock      sync.RWMutex
	storeTimeout      time.Duration
	queryMaxPeriods   uint32
}

// HandlerParams pass parameters to the Handler constructor NewHandler
// Signer and TimestampProvider are mandatory parameters
type HandlerParams struct {
	QueryMaxPeriods   uint32
	TimestampProvider timestampProvider
}

// hashPool contains a pool of ready hashers
var hashPool sync.Pool

// init initializes the package and hashPool
func init() {
	hashPool = sync.Pool{
		New: func() interface{} {
			return storage.MakeHashFunc(resourceHashAlgorithm)()
		},
	}
}

// NewHandler creates a new Mutable Resource API
func NewHandler(params *HandlerParams) (*Handler, error) {

	rh := &Handler{
		timestampProvider: params.TimestampProvider,
		resources:         make(map[uint64]*resource),
		storeTimeout:      defaultStoreTimeout,
		queryMaxPeriods:   params.QueryMaxPeriods,
	}

	if rh.timestampProvider == nil {
		rh.timestampProvider = NewDefaultTimestampProvider()
	}

	for i := 0; i < hasherCount; i++ {
		hashfunc := storage.MakeHashFunc(resourceHashAlgorithm)()
		if rh.HashSize == 0 {
			rh.HashSize = hashfunc.Size()
		}
		hashPool.Put(hashfunc)
	}

	return rh, nil
}

// SetStore sets the store backend for the Mutable Resource API
func (h *Handler) SetStore(store *storage.NetStore) {
	h.chunkStore = store
}

// Validate is a chunk validation method
// If it's a resource update, the chunk address is checked against the ownerAddr of the update's signature
// It implements the storage.ChunkValidator interface
func (h *Handler) Validate(chunkAddr storage.Address, data []byte) bool {

	dataLength := len(data)
	if dataLength < 2 {
		return false
	}

	//metadata chunks have the first two bytes set to zero
	if data[0] == 0 && data[1] == 0 && dataLength > common.AddressLength {
		//metadata chunk
		rootAddr, _ := metadataHash(data)
		valid := bytes.Equal(chunkAddr, rootAddr)
		if !valid {
			log.Warn("Invalid root metadata chunk")
		}
		return valid
	}

	// if it is not a metadata chunk, check if it is an update chunk with
	// valid signature and proof of ownership of the resource it is trying
	// to update

	var r SignedResourceUpdate
	if err := r.parseUpdateChunk(chunkAddr, data); err != nil {
		log.Warn("Invalid resource chunk: " + err.Error())
		return false
	}

	return true
}

// GetContent retrieves the data payload of the last synced update of the Mutable Resource
func (h *Handler) GetContent(rootAddr storage.Address) (storage.Address, []byte, error) {
	rsrc := h.get(rootAddr)
	if rsrc == nil || !rsrc.isSynced() {
		return nil, nil, NewError(ErrNotFound, " does not exist or is not synced")
	}
	return rsrc.lastKey, rsrc.data, nil
}

// GetLastPeriod retrieves the period of the last synced update of the Mutable Resource
func (h *Handler) GetLastPeriod(rootAddr storage.Address) (uint32, error) {
	rsrc := h.get(rootAddr)
	if rsrc == nil {
		return 0, NewError(ErrNotFound, " does not exist")
	} else if !rsrc.isSynced() {
		return 0, NewError(ErrNotSynced, " is not synced")
	}
	return rsrc.period, nil
}

// GetVersion retrieves the period of the last synced update of the Mutable Resource
func (h *Handler) GetVersion(rootAddr storage.Address) (uint32, error) {
	rsrc := h.get(rootAddr)
	if rsrc == nil {
		return 0, NewError(ErrNotFound, " does not exist")
	} else if !rsrc.isSynced() {
		return 0, NewError(ErrNotSynced, " is not synced")
	}
	return rsrc.version, nil
}

// \TODO should be hashsize * branches from the chosen chunker, implement with FileStore
func (h *Handler) chunkSize() int64 {
	return chunkSize
}

// New creates a new metadata chunk out of the request passed in.
func (h *Handler) New(ctx context.Context, request *Request) error {

	// frequency 0 is invalid
	if request.frequency == 0 {
		return NewError(ErrInvalidValue, "frequency cannot be 0 when creating a resource")
	}

	// make sure name only contains ascii values
	if !isSafeName(request.name) {
		return NewErrorf(ErrInvalidValue, "invalid name: '%s'", request.name)
	}

	// make sure owner is set to something
	var zeroAddr = common.Address{}
	if request.ownerAddr == zeroAddr {
		return NewError(ErrInvalidValue, "ownerAddr must be set to create a new metadata chunk")
	}

	// create the meta chunk and store it in swarm
	chunk, metaHash, err := request.resourceMetadata.newChunk()
	if err != nil {
		return err
	}
	if request.metaHash != nil && !bytes.Equal(request.metaHash, metaHash) ||
		request.rootAddr != nil && !bytes.Equal(request.rootAddr, chunk.Addr) {
		return NewError(ErrInvalidValue, "metaHash in UpdateRequest does not match actual metadata")
	}

	request.metaHash = metaHash
	request.rootAddr = chunk.Addr

	h.chunkStore.Put(chunk)
	log.Debug("new resource", "name", request.name, "startBlock", request.startTime, "frequency", request.frequency, "owner", request.ownerAddr)

	// create the internal index for the resource and populate it with the data of the first version
	rsrc := &resource{
		resourceUpdate: resourceUpdate{
			updateHeader: updateHeader{
				UpdateLookup: UpdateLookup{
					rootAddr: chunk.Addr,
				},
			},
		},
		resourceMetadata: resourceMetadata{
			name:      request.name,
			startTime: request.startTime,
			frequency: request.frequency,
		},
		updated: time.Now(),
	}
	copy(rsrc.ownerAddr[:], request.ownerAddr[:])
	h.set(chunk.Addr, rsrc)

	return nil
}

// NewUpdateRequest prepares an UpdateRequest structure with all the necessary information to
// just add the desired data and sign it.
// The resulting structure can then be signed and passed to Handler.Update to be verified and sent
func (h *Handler) NewUpdateRequest(ctx context.Context, rootAddr storage.Address) (*Request, error) {

	if rootAddr == nil {
		return nil, NewError(ErrInvalidValue, "rootAddr cannot be nil")
	}

	// Make sure we have a cache of the metadata chunk
	rsrc, err := h.Load(rootAddr)
	if err != nil {
		return nil, err
	}

	now := h.getCurrentTime(ctx)

	updateRequest := new(Request)
	updateRequest.period, err = getNextPeriod(rsrc.startTime.Time, now.Time, rsrc.frequency)
	if err != nil {
		return nil, err
	}
	if _, err = h.lookup(rsrc, LookupLatestVersionInPeriod(rsrc.rootAddr, updateRequest.period)); err != nil {
		return nil, err
	}

	if !rsrc.isSynced() {
		return nil, NewErrorf(ErrNotSynced, "Handler.NewUpdateRequest: object '%s' not in sync", rootAddr.Hex())
	}

	updateRequest.multihash = rsrc.multihash
	updateRequest.rootAddr = rsrc.rootAddr
	updateRequest.metaHash = rsrc.metaHash
	updateRequest.resourceMetadata = rsrc.resourceMetadata

	// if we already have an update for this period then increment version
	// resource object MUST be in sync for version to be correct, but we checked this earlier in the method already
	if h.hasUpdate(rootAddr, updateRequest.period) {
		updateRequest.version = rsrc.version + 1
	} else {
		updateRequest.version = 1
	}

	return updateRequest, nil
}

// LookupLatest retrieves the latest version of the resource update with metadata chunk at params.Root
// It starts at the next period after the current block height, and upon failure
// tries the corresponding keys of each previous period until one is found
// (or startBlock is reached, in which case there are no updates).
func (h *Handler) Lookup(ctx context.Context, params *LookupParams) (*resource, error) {

	rsrc := h.get(params.rootAddr)
	if rsrc == nil {
		return nil, NewError(ErrNothingToReturn, "resource not loaded")
	}
	if params.period == 0 {
		// get our blockheight at this time and the next block of the update period
		now := h.getCurrentTime(ctx)

		var period uint32
		period, err := getNextPeriod(rsrc.startTime.Time, now.Time, rsrc.frequency)
		if err != nil {
			return nil, err
		}
		params.period = period
	}
	return h.lookup(rsrc, params)
}

// LookupPreviousByName returns the resource before the one currently loaded in the resource index
// This is useful where resource updates are used incrementally in contrast to
// merely replacing content.
// Requires a synced resource object
func (h *Handler) LookupPrevious(ctx context.Context, params *LookupParams) (*resource, error) {
	rsrc := h.get(params.rootAddr)
	if rsrc == nil {
		return nil, NewError(ErrNothingToReturn, "resource not loaded")
	}
	if !rsrc.isSynced() {
		return nil, NewError(ErrNotSynced, "LookupPrevious requires synced resource.")
	} else if rsrc.period == 0 {
		return nil, NewError(ErrNothingToReturn, " not found")
	}
	if rsrc.version > 1 {
		rsrc.version--
	} else if rsrc.period == 1 {
		return nil, NewError(ErrNothingToReturn, "Current update is the oldest")
	} else {
		rsrc.version = 0
		rsrc.period--
	}
	return h.lookup(rsrc, NewLookupParams(rsrc.rootAddr, rsrc.period, rsrc.version, params.Limit))
}

// base code for public lookup methods
func (h *Handler) lookup(rsrc *resource, params *LookupParams) (*resource, error) {

	lp := *params
	// we can't look for anything without a store
	if h.chunkStore == nil {
		return nil, NewError(ErrInit, "Call Handler.SetStore() before performing lookups")
	}

	// period 0 does not exist
	if lp.period == 0 {
		return nil, NewError(ErrInvalidValue, "period must be >0")
	}

	// start from the last possible block period, and iterate previous ones until we find a match
	// if we hit startBlock we're out of options
	var specificversion bool
	if lp.version > 0 {
		specificversion = true
	} else {
		lp.version = 1
	}

	var hops uint32
	if lp.Limit == 0 {
		lp.Limit = h.queryMaxPeriods
	}
	log.Trace("resource lookup", "period", lp.period, "version", lp.version, "limit", lp.Limit)
	for lp.period > 0 {
		if lp.Limit != 0 && hops > lp.Limit {
			return nil, NewErrorf(ErrPeriodDepth, "Lookup exceeded max period hops (%d)", lp.Limit)
		}
		updateAddr := lp.GetUpdateAddr()
		chunk, err := h.chunkStore.GetWithTimeout(updateAddr, defaultRetrieveTimeout)
		if err == nil {
			if specificversion {
				return h.updateIndex(rsrc, chunk)
			}
			// check if we have versions > 1. If a version fails, the previous version is used and returned.
			log.Trace("rsrc update version 1 found, checking for version updates", "period", lp.period, "updateAddr", updateAddr)
			for {
				newversion := lp.version + 1
				updateAddr := lp.GetUpdateAddr()
				newchunk, err := h.chunkStore.GetWithTimeout(updateAddr, defaultRetrieveTimeout)
				if err != nil {
					return h.updateIndex(rsrc, chunk)
				}
				chunk = newchunk
				lp.version = newversion
				log.Trace("version update found, checking next", "version", lp.version, "period", lp.period, "updateAddr", updateAddr)
			}
		}
		log.Trace("rsrc update not found, checking previous period", "period", lp.period, "updateAddr", updateAddr)
		lp.period--
		hops++
	}
	return nil, NewError(ErrNotFound, "no updates found")
}

// Load retrieves the Mutable Resource metadata chunk stored at rootAddr
// Upon retrieval it creates/updates the index entry for it with metadata corresponding to the chunk contents
func (h *Handler) Load(rootAddr storage.Address) (*resource, error) {
	chunk, err := h.chunkStore.GetWithTimeout(rootAddr, defaultRetrieveTimeout)
	if err != nil {
		return nil, NewError(ErrNotFound, err.Error())
	}

	// create the index entry
	rsrc := &resource{}

	if err := rsrc.resourceMetadata.binaryGet(chunk.SData); err != nil { // Will fail if this is not really a metadata chunk
		return nil, err
	}

	rsrc.rootAddr, rsrc.metaHash = metadataHash(chunk.SData)
	if !bytes.Equal(rsrc.rootAddr, rootAddr) {
		return nil, NewError(ErrCorruptData, "Corrupt metadata chunk")
	}
	h.set(rootAddr, rsrc)
	log.Trace("resource index load", "rootkey", rootAddr, "name", rsrc.name, "startblock", rsrc.startTime, "frequency", rsrc.frequency)
	return rsrc, nil
}

// update mutable resource index map with specified content
func (h *Handler) updateIndex(rsrc *resource, chunk *storage.Chunk) (*resource, error) {

	// retrieve metadata from chunk data and check that it matches this mutable resource
	var r SignedResourceUpdate
	if err := r.parseUpdateChunk(chunk.Addr, chunk.SData); err != nil {
		return nil, NewErrorf(ErrInvalidSignature, "Invalid resource chunk: %s", err)
	}
	log.Trace("resource index update", "name", rsrc.name, "updatekey", chunk.Addr, "period", r.period, "version", r.version)

	// update our rsrcs entry map
	rsrc.lastKey = chunk.Addr
	rsrc.period = r.period
	rsrc.version = r.version
	rsrc.updated = time.Now()
	rsrc.data = make([]byte, len(r.data))
	rsrc.multihash = r.multihash
	rsrc.Reader = bytes.NewReader(rsrc.data)
	copy(rsrc.data, r.data)
	log.Debug(" synced", "name", rsrc.name, "updateAddr", chunk.Addr, "period", rsrc.period, "version", rsrc.version)
	h.set(chunk.Addr, rsrc)
	return rsrc, nil
}

// Update adds an actual data update
// Uses the Mutable Resource metadata currently loaded in the resources map entry.
// It is the caller's responsibility to make sure that this data is not stale.
// Note that a Mutable Resource update cannot span chunks, and thus has a MAX NET LENGTH 4096, INCLUDING update header data and signature. An error will be returned if the total length of the chunk payload will exceed this limit.
func (h *Handler) Update(ctx context.Context, rootAddr storage.Address, r *SignedResourceUpdate) (storage.Address, error) {
	return h.update(ctx, rootAddr, r)
}

// create and commit an update
func (h *Handler) update(ctx context.Context, rootAddr storage.Address, r *SignedResourceUpdate) (storage.Address, error) {

	// we can't update anything without a store
	if h.chunkStore == nil {
		return nil, NewError(ErrInit, "Call Handler.SetStore() before updating")
	}

	// get the cached information

	rsrc := h.get(rootAddr)
	if rsrc == nil {
		return nil, NewErrorf(ErrNotFound, " object '%s' not in index", rsrc.name)
	} else if !rsrc.isSynced() {
		return nil, NewError(ErrNotSynced, " object not in sync")
	}

	// an update can be only one chunk long; data length less header and signature data
	if int64(len(r.data)) > maxUpdateDataLength {
		return nil, NewErrorf(ErrDataOverflow, "Data overflow: %d / %d bytes", len(r.data), maxUpdateDataLength)
	}

	// Check that the update is consecutive, i.e., that the proposed update
	// occurs exactly after the latest one we know.
	if rsrc.period == r.period {
		if r.version != rsrc.version+1 {
			return nil, NewErrorf(ErrInvalidValue, "Invalid version for this period. Expected version=%d", rsrc.version+1)
		}
	} else {
		if !(r.period > rsrc.period && r.version == 1) {
			return nil, NewErrorf(ErrInvalidValue, "Invalid version,period. Expected version=1 and period > %d", rsrc.period)
		}
	}

	chunk, err := r.newUpdateChunk() // Serialize the update into a chunk
	if err != nil {
		return nil, err
	}

	// send the chunk
	h.chunkStore.Put(chunk)
	log.Trace("resource update", "updateAddr", r.updateAddr, "lastperiod", r.period, "version", r.version, "data", chunk.SData, "multihash", r.multihash)

	// update our resources map entry and return the new updateAddr
	rsrc.period = r.period
	rsrc.version = r.version
	rsrc.data = make([]byte, len(r.data))
	copy(rsrc.data, r.data)
	return r.updateAddr, nil
}

// gets the current time
func (h *Handler) getCurrentTime(ctx context.Context) Timestamp {
	return h.timestampProvider.GetCurrentTimestamp()
}

// Retrieves the resource index value for the given nameHash
func (h *Handler) get(rootAddr storage.Address) *resource {
	if len(rootAddr) < storage.KeyLength {
		log.Warn("Handler.get with invalid rootAddr")
		return nil
	}
	hashKey := *(*uint64)(unsafe.Pointer(&rootAddr[0]))
	h.resourceLock.RLock()
	defer h.resourceLock.RUnlock()
	rsrc := h.resources[hashKey]
	return rsrc
}

// Sets the resource index value for the given nameHash
func (h *Handler) set(rootAddr storage.Address, rsrc *resource) {
	if len(rootAddr) < storage.KeyLength {
		log.Warn("Handler.set with invalid rootAddr")
		return
	}
	hashKey := *(*uint64)(unsafe.Pointer(&rootAddr[0]))
	h.resourceLock.Lock()
	defer h.resourceLock.Unlock()
	h.resources[hashKey] = rsrc
}

// Checks if we already have an update on this resource, according to the value in the current state of the resource index
func (h *Handler) hasUpdate(rootAddr storage.Address, period uint32) bool {
	rsrc := h.get(rootAddr)
	return rsrc != nil && rsrc.period == period
}
