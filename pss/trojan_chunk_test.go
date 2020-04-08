// Copyright 2020 The Swarm Authors
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

package pss

import (
	"bytes"
	"reflect"
	"testing"
)

// arbitrary targets for tests
var testTargets = [][]byte{
	[]byte{57, 120},
	[]byte{209, 156},
	[]byte{156, 38},
	[]byte{89, 19},
	[]byte{22, 129}}

// newTestTrojanMsg creates an arbitrary trojan message for tests
func newTestTrojanMsg(t *testing.T) trojanMsg {
	payload := []byte("foopayload")
	tm, err := newTrojanMsg(newMsgTopic("RECOVERY"), payload)
	if err != nil {
		t.Fatal(err)
	}

	return tm
}

// TODO: add failure tests

// TestNewTrojanChunk tests the creation of a trojan chunk
// its resulting address should have a prefix which matches one of the given targets
// its resulting payload should have a hash that matches its address exactly
func TestNewTrojanChunk(t *testing.T) {
	tc, err := newTrojanChunk(testTargets, newTestTrojanMsg(t))
	if err != nil {
		t.Fatal(err)
	}

	addr := tc.Address()
	addrPrefix := addr[:len(testTargets[0])]

	if !contains(testTargets, addrPrefix) {
		t.Fatal("trojan chunk address prefix does not match any of the targets")
	}

	payload := tc.Data()
	payloadHash, err := hash(payload)
	if err != nil {
		t.Fatal(err)
	}

	if !bytes.Equal(addr, payloadHash) {
		t.Fatal("trojan chunk address does not match its payload hash")
	}
}

// TestTrojanMsgSerialization tests that the trojanMessage type can be correctly serialized and deserialized
func TestTrojanMsgSerialization(t *testing.T) {
	tm := newTestTrojanMsg(t)

	stm, err := tm.MarshalBinary()
	if err != nil {
		t.Fatal(err)
	}

	dtm := new(trojanMsg)
	err = dtm.UnmarshalBinary(stm)
	if err != nil {
		t.Fatal(err)
	}

	if !reflect.DeepEqual(tm, *dtm) {
		t.Fatalf("original trojan message does not match deserialized one")
	}
}
