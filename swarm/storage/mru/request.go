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

package mru

import (
	"bytes"
	"encoding/json"
	"hash"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/swarm/storage"
	"github.com/ethereum/go-ethereum/swarm/storage/mru/lookup"
)

// Request represents an update and/or resource create message
type Request struct {
	ResourceUpdate // actual content that will be put on the chunk, less signature
	Signature      *Signature
	updateAddr     storage.Address // resulting chunk address for the update (not serialized, for internal use)
	binaryData     []byte          // resulting serialized data (not serialized, for efficiency/internal use)
}

// updateRequestJSON represents a JSON-serialized UpdateRequest
type updateRequestJSON struct {
	UpdateLookup
	Data      string `json:"data,omitempty"`
	Signature string `json:"signature,omitempty"`
}

var zeroAddr = common.Address{}

// Request layout
// resourceUpdate bytes
// SignatureLength bytes
const minimumSignedUpdateLength = minimumUpdateDataLength + signatureLength

// NewFirstRequest returns a ready to sign request to publish a first update
func NewFirstRequest(topic Topic) *Request {

	request := new(Request)

	// get the current time
	now := TimestampProvider.Now().Time
	request.Epoch = lookup.GetFirstEpoch(now)
	request.View.Topic = topic

	return request
}

// SetData stores the payload data the resource will be updated with
func (r *Request) SetData(data []byte) {
	r.data = data
	r.Signature = nil
}

// IsUpdate returns true if this request models a signed update or otherwise it is a signature request
func (r *Request) IsUpdate() bool {
	return r.Signature != nil
}

// Verify checks that signatures are valid and that the signer owns the resource to be updated
func (r *Request) Verify() (err error) {
	if len(r.data) == 0 {
		return NewError(ErrInvalidValue, "Update does not contain data")
	}
	if r.Signature == nil {
		return NewError(ErrInvalidSignature, "Missing signature field")
	}

	digest, err := r.GetDigest()
	if err != nil {
		return err
	}

	// get the address of the signer (which also checks that it's a valid signature)
	r.View.User, err = getUserAddr(digest, *r.Signature)
	if err != nil {
		return err
	}

	// check that the lookup information contained in the chunk matches the updateAddr (chunk search key)
	// that was used to retrieve this chunk
	// if this validation fails, someone forged a chunk.
	if !bytes.Equal(r.updateAddr, r.UpdateAddr()) {
		return NewError(ErrInvalidSignature, "Signature address does not match with update user address")
	}

	return nil
}

// Sign executes the signature to validate the resource
func (r *Request) Sign(signer Signer) error {
	r.View.User = signer.Address()
	r.binaryData = nil           //invalidate serialized data
	digest, err := r.GetDigest() // computes digest and serializes into .binaryData
	if err != nil {
		return err
	}

	signature, err := signer.Sign(digest)
	if err != nil {
		return err
	}

	// Although the Signer interface returns the public address of the signer,
	// recover it from the signature to see if they match
	userAddr, err := getUserAddr(digest, signature)
	if err != nil {
		return NewError(ErrInvalidSignature, "Error verifying signature")
	}

	if userAddr != signer.Address() { // sanity check to make sure the Signer is declaring the same address used to sign!
		return NewError(ErrInvalidSignature, "Signer address does not match update user address")
	}

	r.Signature = &signature
	r.updateAddr = r.UpdateAddr()
	return nil
}

// GetDigest creates the resource update digest used in signatures
// the serialized payload is cached in .binaryData
func (r *Request) GetDigest() (result common.Hash, err error) {
	hasher := hashPool.Get().(hash.Hash)
	defer hashPool.Put(hasher)
	hasher.Reset()
	dataLength := r.ResourceUpdate.binaryLength()
	if r.binaryData == nil {
		r.binaryData = make([]byte, dataLength+signatureLength)
		if err := r.ResourceUpdate.binaryPut(r.binaryData[:dataLength]); err != nil {
			return result, err
		}
	}
	hasher.Write(r.binaryData[:dataLength]) //everything except the signature.

	return common.BytesToHash(hasher.Sum(nil)), nil
}

// create an update chunk.
func (r *Request) toChunk() (*storage.Chunk, error) {

	// Check that the update is signed and serialized
	// For efficiency, data is serialized during signature and cached in
	// the binaryData field when computing the signature digest in .getDigest()
	if r.Signature == nil || r.binaryData == nil {
		return nil, NewError(ErrInvalidSignature, "toChunk called without a valid signature or payload data. Call .Sign() first.")
	}

	chunk := storage.NewChunk(r.updateAddr, nil)
	resourceUpdateLength := r.ResourceUpdate.binaryLength()
	chunk.SData = r.binaryData

	// signature is the last item in the chunk data
	copy(chunk.SData[resourceUpdateLength:], r.Signature[:])

	chunk.Size = int64(len(chunk.SData))
	return chunk, nil
}

// fromChunk populates this structure from chunk data. It does not verify the signature is valid.
func (r *Request) fromChunk(updateAddr storage.Address, chunkdata []byte) error {
	// for update chunk layout see Request definition

	//deserialize the resource update portion
	if err := r.ResourceUpdate.binaryGet(chunkdata[:len(chunkdata)-signatureLength]); err != nil {
		return err
	}

	// Extract the signature
	var signature *Signature
	cursor := r.ResourceUpdate.binaryLength()
	sigdata := chunkdata[cursor : cursor+signatureLength]
	if len(sigdata) > 0 {
		signature = &Signature{}
		copy(signature[:], sigdata)
	}

	r.Signature = signature
	r.updateAddr = updateAddr
	r.binaryData = chunkdata

	return nil

}

// FromValues deserializes this instance from a string key-value store
// useful to parse query strings
func (r *Request) FromValues(values Values, data []byte) error {
	signatureBytes, err := hexutil.Decode(values.Get("signature"))
	if err != nil {
		r.Signature = nil
	} else {
		if len(signatureBytes) != signatureLength {
			return NewError(ErrInvalidSignature, "Incorrect signature length")
		}
		r.Signature = new(Signature)
		copy(r.Signature[:], signatureBytes)
	}
	err = r.ResourceUpdate.FromValues(values, data)
	if err != nil {
		return err
	}
	r.updateAddr = r.UpdateAddr()
	return err
}

// ToValues serializes this structure into the provided string key-value store
// useful to build query strings
func (r *Request) ToValues(values Values) []byte {
	if r.Signature != nil {
		values.Set("signature", hexutil.Encode(r.Signature[:]))
	}
	return r.ResourceUpdate.ToValues(values)
}

// fromJSON takes an update request JSON and populates an UpdateRequest
func (r *Request) fromJSON(j *updateRequestJSON) error {

	r.UpdateLookup = j.UpdateLookup

	var err error
	if j.Data != "" {
		r.data, err = hexutil.Decode(j.Data)
		if err != nil {
			return NewError(ErrInvalidValue, "Cannot decode data")
		}
	}

	if j.Signature != "" {
		sigBytes, err := hexutil.Decode(j.Signature)
		if err != nil || len(sigBytes) != signatureLength {
			return NewError(ErrInvalidSignature, "Cannot decode signature")
		}
		r.Signature = new(Signature)
		r.updateAddr = r.UpdateAddr()
		copy(r.Signature[:], sigBytes)
	}
	return nil
}

// UnmarshalJSON takes a JSON structure stored in a byte array and populates the Request object
// Implements json.Unmarshaler interface
func (r *Request) UnmarshalJSON(rawData []byte) error {
	var requestJSON updateRequestJSON
	if err := json.Unmarshal(rawData, &requestJSON); err != nil {
		return err
	}
	return r.fromJSON(&requestJSON)
}

// MarshalJSON takes an update request and encodes it as a JSON structure into a byte array
// Implements json.Marshaler interface
func (r *Request) MarshalJSON() (rawData []byte, err error) {
	var signatureString, dataString string
	if r.Signature != nil {
		signatureString = hexutil.Encode(r.Signature[:])
	}
	if r.data != nil {
		dataString = hexutil.Encode(r.data)
	}

	requestJSON := &updateRequestJSON{
		UpdateLookup: r.UpdateLookup,
		Data:         dataString,
		Signature:    signatureString,
	}

	return json.Marshal(requestJSON)
}
