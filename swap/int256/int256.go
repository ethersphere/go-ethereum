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

package int256

import (
	"fmt"
	"io"
	"math/big"
	"strconv"

	"github.com/ethereum/go-ethereum/rlp"
)

// Int256 represents an signed integer of 256 bits
type Int256 struct {
	value *big.Int
}

var minInt256 = new(big.Int).Mul(big.NewInt(-1), new(big.Int).Exp(big.NewInt(2), big.NewInt(255), nil)) // -(2^255)
var maxInt256 = new(big.Int).Sub(new(big.Int).Exp(big.NewInt(2), big.NewInt(255), nil), big.NewInt(1))  // 2^255 - 1

// NewInt256 creates a Int256 struct with an initial underlying value of the given param
// returns an error when the value cannot be correctly set
func NewInt256(value *big.Int) (*Int256, error) {
	u := new(Int256)
	return u.set(value)
}

// Int256From creates a Int256 struct based on the given int64 param
// any int64 is valid as a Int256
func Int256From(base int64) *Int256 {
	u := new(Int256)
	u.value = new(big.Int).SetInt64(base)
	return u
}

// Copy creates and returns a new Int256 instance, with its underlying value set matching the receiver
func (u *Int256) Copy() *Int256 {
	v := new(Int256)
	v.value = new(big.Int).Set(u.value)
	return v
}

// Value returns the underlying private value for a Int256 struct
func (u *Int256) Value() *big.Int {
	return new(big.Int).Set(u.value)
}

// set assigns the underlying value of the given Int256 param to u, and returns the modified receiver struct
// returns an error when the value cannot be correctly set
func (u *Int256) set(value *big.Int) (*Int256, error) {
	if err := checkInt256Bounds(value); err != nil {
		return nil, err
	}
	if u.value == nil {
		u.value = new(big.Int)
	}
	u.value.Set(value)
	return u, nil
}

// checkInt256Bounds returns an error when the given value falls outside of the signed 256-bit integer range or is nil
// returns nil otherwise
func checkInt256Bounds(value *big.Int) error {
	if value == nil {
		return fmt.Errorf("cannot set Int256 to a nil value")
	}
	if value.Cmp(maxInt256) == 1 {
		return fmt.Errorf("cannot set Int256 to %v as it overflows max value of %v", value, maxInt256)
	}
	if value.Cmp(minInt256) == -1 {
		return fmt.Errorf("cannot set Int256 to %v as it underflows min value of %v", value, minInt256)
	}
	return nil
}

// Add sets u to augend + addend and returns u as the sum
// returns an error when the value cannot be correctly set
func (u *Int256) Add(augend, addend *Int256) (*Int256, error) {
	sum := new(big.Int).Add(augend.value, addend.value)
	return u.set(sum)
}

// Sub sets u to minuend - subtrahend and returns u as the difference
// returns an error when the value cannot be correctly set
func (u *Int256) Sub(minuend, subtrahend *Int256) (*Int256, error) {
	difference := new(big.Int).Sub(minuend.value, subtrahend.value)
	return u.set(difference)
}

// Mul sets u to multiplicand * multiplier and returns u as the product
// returns an error when the value cannot be correctly set
func (u *Int256) Mul(multiplicand, multiplier *Int256) (*Int256, error) {
	product := new(big.Int).Mul(multiplicand.value, multiplier.value)
	return u.set(product)
}

// cmp calls the underlying Cmp method for the big.Int stored in a Int256 struct as its value field
func (u *Int256) cmp(v *Int256) int {
	return u.value.Cmp(v.value)
}

// Equals returns true if the two Int256 structs have the same underlying values, false otherwise
func (u *Int256) Equals(v *Int256) bool {
	return u.cmp(v) == 0
}

// String returns the string representation for Int256 structs
func (u *Int256) String() string {
	return u.value.String()
}

// MarshalJSON implements the json.Marshaler interface
// it specifies how to marshal a Int256 struct so that it can be written to disk
func (u *Int256) MarshalJSON() ([]byte, error) {
	// number is wrapped in quotes to prevent json number overflowing
	return []byte(strconv.Quote(u.value.String())), nil
}

// UnmarshalJSON implements the json.Unmarshaler interface
// it specifies how to unmarshal a Int256 struct so that it can be reconstructed from disk
func (u *Int256) UnmarshalJSON(b []byte) error {
	var value big.Int
	// value string must be unquoted due to marshaling
	strValue, err := strconv.Unquote(string(b))
	if err != nil {
		return err
	}
	_, ok := (&value).SetString(strValue, 10)
	if !ok {
		return fmt.Errorf("not a valid integer value: %s", b)
	}
	_, err = u.set(&value)
	return err
}

// EncodeRLP implements the rlp.Encoder interface
// it makes sure the value field is encoded even though it is private
func (u *Int256) EncodeRLP(w io.Writer) error {
	return rlp.Encode(w, &u.value)
}

// DecodeRLP implements the rlp.Decoder interface
// it makes sure the value field is decoded even though it is private
func (u *Int256) DecodeRLP(s *rlp.Stream) error {
	if err := s.Decode(&u.value); err != nil {
		return err
	}
	if err := checkInt256Bounds(u.value); err != nil {
		return err
	}
	return nil
}
