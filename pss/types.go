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

package pss

import (
	"encoding/json"

	"github.com/ethereum/go-ethereum/common/hexutil"
)

// PssAddress is an alias for []byte. It represents a variable length address
type PssAddress []byte

// MarshalJSON implements the json.Marshaler interface
func (a PssAddress) MarshalJSON() ([]byte, error) {
	return json.Marshal(hexutil.Encode(a[:]))
}

// UnmarshalJSON implements the json.Marshaler interface
func (a *PssAddress) UnmarshalJSON(input []byte) error {
	b, err := hexutil.Decode(string(input[1 : len(input)-1]))
	if err != nil {
		return err
	}
	for _, bb := range b {
		*a = append(*a, bb)
	}
	return nil
}
