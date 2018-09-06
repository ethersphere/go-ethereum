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

package lookup_test

import (
	"fmt"
	"testing"

	"github.com/ethereum/go-ethereum/swarm/log"
	"github.com/ethereum/go-ethereum/swarm/storage/mru/lookup"
)

type Data struct {
	Payload uint64
	Time    uint64
}

type Store map[lookup.EpochID]*Data

func write(store Store, epoch lookup.Epoch, value *Data) {
	log.Debug("Write: %d-%d, value='%d'\n", epoch.Base(), epoch.Level, value.Payload)
	store[epoch.ID()] = value
}

func update(store Store, last lookup.Epoch, now uint64, value *Data) lookup.Epoch {
	var epoch lookup.Epoch

	epoch = lookup.GetNextEpoch(last, now)

	write(store, epoch, value)

	return epoch
}

const Day = 60 * 60 * 24
const Year = Day * 365
const Month = Day * 30

func makeReadFunc(store Store, counter *int) lookup.ReadFunc {
	return func(epoch lookup.Epoch, now uint64) (interface{}, error) {
		*counter++
		data := store[epoch.ID()]
		var valueStr string
		if data != nil {
			valueStr = fmt.Sprintf("%d", data.Payload)
		}
		log.Debug("Read: %d-%d, value='%s'\n", epoch.Base(), epoch.Level, valueStr)
		if data != nil && data.Time <= now {
			return data, nil
		}
		return nil, nil
	}
}

func TestLookup(t *testing.T) {

	store := make(Store)
	readCount := 0
	readFunc := makeReadFunc(store, &readCount)

	// write an update every month for 12 months 3 years ago and then silence for two years
	now := uint64(1533799046)
	var epoch lookup.Epoch

	var lastData *Data
	for i := uint64(0); i < 12; i++ {
		t := uint64(now - Year*3 + i*Month)
		data := Data{
			Payload: t, //our "payload" will be the timestamp itself.
			Time:    t,
		}
		epoch = update(store, epoch, t, &data)
		lastData = &data
	}

	// try to get the last value

	value, err := lookup.Lookup(now, lookup.NoClue, readFunc)
	if err != nil {
		t.Fatal(err)
	}

	readCountWithoutHint := readCount

	if value != lastData {
		t.Fatalf("Expected lookup to return the last written value: %v. Got %v", lastData, value)
	}

	// reset the read count for the next test
	readCount = 0
	// Provide a hint to get a faster lookup. In particular, we give the exact location of the last update
	value, err = lookup.Lookup(now, epoch, readFunc)
	if err != nil {
		t.Fatal(err)
	}

	if value != lastData {
		t.Fatalf("Expected lookup to return the last written value: %v. Got %v", lastData, value)
	}

	if readCount > readCountWithoutHint {
		t.Fatalf("Expected lookup to complete with fewer or same reads than %d since we provided a hint. Did %d reads.", readCountWithoutHint, readCount)
	}

	// try to get an intermediate value
	// if we look for a value in now - Year*3 + 6*Month, we should get that value
	// Since the "payload" is the timestamp itself, we can check this.

	expectedTime := now - Year*3 + 6*Month

	value, err = lookup.Lookup(expectedTime, lookup.NoClue, readFunc)
	if err != nil {
		t.Fatal(err)
	}

	data, ok := value.(*Data)

	if !ok {
		t.Fatal("Expected value to contain data")
	}

	if data.Time != expectedTime {
		t.Fatalf("Expected value timestamp to be %d, got %d", data.Time, expectedTime)
	}

}

func TestOneUpdateAt0(t *testing.T) {

	store := make(Store)
	readCount := 0

	readFunc := makeReadFunc(store, &readCount)
	now := uint64(1533903729)

	var epoch lookup.Epoch
	data := Data{
		Payload: 79,
		Time:    0,
	}
	update(store, epoch, 0, &data)

	value, err := lookup.Lookup(now, lookup.NoClue, readFunc)
	if err != nil {
		t.Fatal(err)
	}
	if value != &data {
		t.Fatalf("Expected lookup to return the last written value: %v. Got %v", data, value)
	}
}

// Tests the update is found even when a bad hint is given
func TestBadHint(t *testing.T) {

	store := make(Store)
	readCount := 0

	readFunc := makeReadFunc(store, &readCount)
	now := uint64(1533903729)

	var epoch lookup.Epoch
	data := Data{
		Payload: 79,
		Time:    0,
	}

	// place an update for t=1200
	update(store, epoch, 1200, &data)

	// come up with some evil hint
	badHint := lookup.Epoch{
		Level: 18,
		Time:  1200000000,
	}

	value, err := lookup.Lookup(now, badHint, readFunc)
	if err != nil {
		t.Fatal(err)
	}
	if value != &data {
		t.Fatalf("Expected lookup to return the last written value: %v. Got %v", data, value)
	}
}

func TestLookupFail(t *testing.T) {

	store := make(Store)
	readCount := 0

	readFunc := makeReadFunc(store, &readCount)
	now := uint64(1533903729)

	// don't write anything and try to look up.
	// we're testing we don't get stuck in a loop

	value, err := lookup.Lookup(now, lookup.NoClue, readFunc)
	if err != nil {
		t.Fatal(err)
	}
	if value != nil {
		t.Fatal("Expected value to be nil, since the update should've failed")
	}

	expectedReads := now/(1<<lookup.HighestLevel) + 1
	if uint64(readCount) != expectedReads {
		t.Fatalf("Expected lookup to fail after %d reads. Did %d reads.", expectedReads, readCount)
	}
}

func TestHighFreqUpdates(t *testing.T) {

	store := make(Store)
	readCount := 0

	readFunc := makeReadFunc(store, &readCount)
	now := uint64(1533903729)

	// write an update every second for the last 1000 seconds
	var epoch lookup.Epoch

	var lastData *Data
	for i := uint64(0); i <= 994; i++ {
		T := uint64(now - 1000 + i)
		data := Data{
			Payload: T, //our "payload" will be the timestamp itself.
			Time:    T,
		}
		epoch = update(store, epoch, T, &data)
		lastData = &data
	}

	value, err := lookup.Lookup(lastData.Time, lookup.NoClue, readFunc)
	if err != nil {
		t.Fatal(err)
	}

	if value != lastData {
		t.Fatalf("Expected lookup to return the last written value: %v. Got %v", lastData, value)
	}

	readCountWithoutHint := readCount
	// reset the read count for the next test
	readCount = 0
	// Provide a hint to get a faster lookup. In particular, we give the exact location of the last update
	value, err = lookup.Lookup(now, epoch, readFunc)
	if err != nil {
		t.Fatal(err)
	}

	if value != lastData {
		t.Fatalf("Expected lookup to return the last written value: %v. Got %v", lastData, value)
	}

	if readCount > readCountWithoutHint {
		t.Fatalf("Expected lookup to complete with fewer reads than %d since we provided a hint. Did %d reads.", readCountWithoutHint, readCount)
	}

	for i := uint64(0); i <= 994; i++ {
		T := uint64(now - 1000 + i) // update every second for the last 1000 seconds
		value, err := lookup.Lookup(T, lookup.NoClue, readFunc)
		if err != nil {
			t.Fatal(err)
		}
		data, _ := value.(*Data)
		if data == nil {
			t.Fatalf("Expected lookup to return %d, got nil", T)
		}
		if data.Payload != T {
			t.Fatalf("Expected lookup to return %d, got %d", T, data.Time)
		}
	}
}

func TestSparseUpdates(t *testing.T) {

	store := make(Store)
	readCount := 0
	readFunc := makeReadFunc(store, &readCount)

	// write an update every 5 years 3 times starting in Jan 1st 1970 and then silence

	now := uint64(1533799046)
	var epoch lookup.Epoch

	var lastData *Data
	for i := uint64(0); i < 5; i++ {
		T := uint64(Year * 5 * i) // write an update every 5 years 3 times starting in Jan 1st 1970 and then silence
		data := Data{
			Payload: T, //our "payload" will be the timestamp itself.
			Time:    T,
		}
		epoch = update(store, epoch, T, &data)
		lastData = &data
	}

	// try to get the last value

	value, err := lookup.Lookup(now, lookup.NoClue, readFunc)
	if err != nil {
		t.Fatal(err)
	}

	readCountWithoutHint := readCount

	if value != lastData {
		t.Fatalf("Expected lookup to return the last written value: %v. Got %v", lastData, value)
	}

	// reset the read count for the next test
	readCount = 0
	// Provide a hint to get a faster lookup. In particular, we give the exact location of the last update
	value, err = lookup.Lookup(now, epoch, readFunc)
	if err != nil {
		t.Fatal(err)
	}

	if value != lastData {
		t.Fatalf("Expected lookup to return the last written value: %v. Got %v", lastData, value)
	}

	if readCount > readCountWithoutHint {
		t.Fatalf("Expected lookup to complete with fewer reads than %d since we provided a hint. Did %d reads.", readCountWithoutHint, readCount)
	}

}
