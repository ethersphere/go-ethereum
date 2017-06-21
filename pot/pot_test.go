// Copyright 2017 The go-ethereum Authors
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
package pot

import (
	"errors"
	"fmt"
	"math/rand"
	"os"
	"runtime"
	"sync"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/log"
)

const (
	maxEachNeighbourTests = 420
	maxEachNeighbour      = 420
)

func init() {
	log.Root().SetHandler(log.LvlFilterHandler(log.LvlError, log.StreamHandler(os.Stderr, log.TerminalFormat(false))))
}

type testAddr struct {
	a []byte
	i int
}

//
// func newTestAddr(s string, i int) *testAddr {
// 	b := NewAddressFromString(s)
// 	h := &common.Hash{}
// 	copy((*h)[:], b)
// 	a := Address(*h)
// 	return &testAddr{&a, i}
// }

func newTestAddr(s string, i int) *testAddr {
	return &testAddr{NewAddressFromString(s), i}
}

func (a *testAddr) Address() []byte {
	return a.a
}

func (a *testAddr) String() string {
	return Label(a.a)
	// return a.Address.String()[:keylen]
}

func randomTestAddr(n int, i int) *testAddr {
	v := RandomAddress().Bin()[:n]
	return newTestAddr(v, i)
}

func randomtestAddr(n int, i int) *testAddr {
	v := RandomAddress().Bin()[:n]
	return newTestAddr(v, i)
}

func indexes(t *Pot) (i []int, po []int) {
	t.Each(func(v Val, p int) bool {
		a := v.(*testAddr)
		i = append(i, a.i)
		po = append(po, p)
		return true
	})
	return i, po
}

func testAdd(t *Pot, n int, values ...string) {
	for i, val := range values {
		t.Add(newTestAddr(val, i+n))
	}
}

// func RandomBoolAddress()
func TestPotAdd(t *testing.T) {
	n := NewPot(newTestAddr("00111100", 0), 0, DefaultPof(8))
	// Pin set correctly
	exp := "00111100"
	got := Label(n.Pin())[:8]
	if got != exp {
		t.Fatalf("incorrect pinned value. Expected %v, got %v", exp, got)
	}
	// check size
	goti := n.Size()
	expi := 1
	if goti != expi {
		t.Fatalf("incorrect number of elements in Pot. Expected %v, got %v", expi, goti)
	}

	testAdd(n, 1, "01111100", "00111100", "01111100", "00011100")
	// check size
	goti = n.Size()
	expi = 3
	if goti != expi {
		t.Fatalf("incorrect number of elements in Pot. Expected %v, got %v", expi, goti)
	}
	inds, po := indexes(n)
	got = fmt.Sprintf("%v", inds)
	exp = "[3 4 2]"
	if got != exp {
		t.Fatalf("incorrect indexes in iteration over Pot. Expected %v, got %v", exp, got)
	}
	got = fmt.Sprintf("%v", po)
	exp = "[1 2 0]"
	if got != exp {
		t.Fatalf("incorrect po-s in iteration over Pot. Expected %v, got %v", exp, got)
	}
}

// func RandomBoolAddress()
func TestPotRemove(t *testing.T) {
	n := NewPot(newTestAddr("00111100", 0), 0, DefaultPof(8))
	n.Remove(newTestAddr("00111100", 0))
	exp := "<nil>"
	got := Label(n.Pin())
	if got != exp {
		t.Fatalf("incorrect pinned value. Expected %v, got %v", exp, got)
	}
	testAdd(n, 1, "00000000", "01111100", "00111100", "00011100")
	n.Remove(newTestAddr("00111100", 0))
	goti := n.Size()
	expi := 3
	if goti != expi {
		t.Fatalf("incorrect number of elements in Pot. Expected %v, got %v", expi, goti)
	}
	inds, po := indexes(n)
	got = fmt.Sprintf("%v", inds)
	exp = "[2 4 0]"
	if got != exp {
		t.Fatalf("incorrect indexes in iteration over Pot. Expected %v, got %v", exp, got)
	}
	got = fmt.Sprintf("%v", po)
	exp = "[1 3 0]"
	if got != exp {
		t.Fatalf("incorrect po-s in iteration over Pot. Expected %v, got %v", exp, got)
	}
	// remove again
	n.Remove(newTestAddr("00111100", 0))
	inds, po = indexes(n)
	got = fmt.Sprintf("%v", inds)
	exp = "[2 4]"
	if got != exp {
		t.Fatalf("incorrect indexes in iteration over Pot. Expected %v, got %v", exp, got)
	}

}

func TestPotSwap(t *testing.T) {
	// t.Skip("")
	max := maxEachNeighbour
	n := NewPot(nil, 0, nil)
	var m []*testAddr
	for j := 0; j < 2*max; {
		v := randomtestAddr(keylen, j)
		_, found := n.Add(v)
		if !found {
			m = append(m, v)
			j++
		}
	}
	k := make(map[string]*testAddr)
	for j := 0; j < max; {
		v := randomtestAddr(keylen, 1)
		_, found := k[Label(v)]
		if !found {
			k[Label(v)] = v
			j++
		}
	}
	for _, v := range k {
		m = append(m, v)
	}
	f := func(v Val) Val {
		tv := v.(*testAddr)
		if tv.i < max {
			return nil
		}
		tv.i = 0
		return v
	}
	for _, val := range m {
		n.Swap(val, func(v Val) Val {
			if v == nil {
				return val
			}
			return f(v)
		})
	}
	sum := 0
	n.Each(func(v Val, i int) bool {
		sum++
		tv := v.(*testAddr)
		if tv.i > 1 {
			t.Fatalf("item value incorrect, expected 0, got %v", tv.i)
		}
		return true
	})
	if sum != 2*max {
		t.Fatalf("incorrect number of elements. expected %v, got %v", 2*max, sum)
	}
	if sum != n.Size() {
		t.Fatalf("incorrect size. expected %v, got %v", sum, n.Size())
	}
}

func checkPo(val Val, pof Pof) func(Val, int) error {
	return func(v Val, po int) error {
		// check the po
		exp, _ := pof(val, v, 0)
		if po != exp {
			return fmt.Errorf("incorrect prox order for item %v in neighbour iteration for %v. Expected %v, got %v", v, val, exp, po)
		}
		return nil
	}
}

func checkOrder(val Val) func(Val, int) error {
	var po int = keylen
	return func(v Val, p int) error {
		if po < p {
			return fmt.Errorf("incorrect order for item %v in neighbour iteration for %v. PO %v > %v (previous max)", v, val, p, po)
		}
		po = p
		return nil
	}
}

func checkValues(m map[string]bool, val Val) func(Val, int) error {
	return func(v Val, po int) error {
		duplicate, ok := m[Label(v)]
		if !ok {
			return fmt.Errorf("alien value %v", v)
		}
		if duplicate {
			return fmt.Errorf("duplicate value returned: %v", v)
		}
		m[Label(v)] = true
		return nil
	}
}

var errNoCount = errors.New("not count")

func testPotEachNeighbour(n *Pot, val Val, expCount int, fs ...func(Val, int) error) error {
	var err error
	var count int
	n.EachNeighbour(val, func(v Val, po int) bool {
		for _, f := range fs {
			err = f(v, po)
			if err != nil {
				return err.Error() == errNoCount.Error()
			}
		}
		count++
		if count == expCount {
			return false
		}
		return true
	})
	if err == nil && count < expCount {
		return fmt.Errorf("not enough neighbours returned, expected %v, got %v", expCount, count)
	}
	return err
}

const (
	mergeTestCount  = 5
	mergeTestChoose = 5
)

func TestPotMergeOne(t *testing.T) {
	pot1 := NewPot(nil, 0, DefaultPof(2))
	pot1.Add(newTestAddr("10", 0))
	pot1.Add(newTestAddr("00", 0))
	pot2 := NewPot(nil, 0, DefaultPof(2))
	pot2.Add(newTestAddr("01", 0))
	pot1.Merge(pot2)
	count := 0
	pot1.Each(func(val Val, i int) bool {
		count++
		return true
	})
	if count != 3 {
		t.Fatalf("expected count to be 3, got %d\n%v\n%v", count, pot2, pot1)
	}
}

func TestPotMergeCommon(t *testing.T) {
	vs := make([]*testAddr, mergeTestCount)
	// vs := make([]*testAddr, mergeTestCount)
	for i := 0; i < maxEachNeighbourTests; i++ {

		for i := 0; i < len(vs); i++ {
			vs[i] = randomtestAddr(keylen, i)
		}
		max0 := rand.Intn(mergeTestChoose) + 1
		max1 := rand.Intn(mergeTestChoose) + 1
		n0 := NewPot(nil, 0, nil)
		n1 := NewPot(nil, 0, nil)
		log.Trace(fmt.Sprintf("round %v: %v - %v", i, max0, max1))
		m := make(map[string]bool)
		for j := 0; j < max0; {
			r := rand.Intn(max0)
			v := vs[r]
			_, found := n0.Add(v)
			if !found {
				m[Label(v)] = false
				j++
			}
		}
		expAdded := 0

		for j := 0; j < max1; {
			r := rand.Intn(max1)
			v := vs[r]
			_, found := n1.Add(v)
			if !found {
				j++
			}
			_, found = m[Label(v)]
			if !found {
				expAdded++
				m[Label(v)] = false
			}
		}
		if i < 6 {
			continue
		}
		expSize := len(m)
		log.Trace(fmt.Sprintf("%v-0: pin: %v, size: %v", i, n0.Pin(), max0))
		log.Trace(fmt.Sprintf("%v-1: pin: %v, size: %v", i, n1.Pin(), max1))
		log.Trace(fmt.Sprintf("%v: merged tree size: %v, newly added: %v", i, expSize, expAdded))
		n, common := Union(n0, n1)
		added := n1.Size() - common
		size := n.Size()

		if expSize != size {
			t.Fatalf("%v: incorrect number of elements in merged pot, expected %v, got %v\n%v", i, expSize, size, n)
		}
		if expAdded != added {
			t.Fatalf("%v: incorrect number of added elements in merged pot, expected %v, got %v", i, expAdded, added)
		}
		if !checkDuplicates(n.pot) {
			t.Fatalf("%v: merged pot contains duplicates: \n%v", i, n)
		}
		for k := range m {
			_, found := n.Add(newTestAddr(k, 0))
			if !found {
				t.Fatalf("%v: merged pot (size:%v, added: %v) missing element %v\n%v", i, size, added, k, n)
			}
		}
	}
}

func TestPotMergeScale(t *testing.T) {
	for i := 0; i < maxEachNeighbourTests; i++ {
		max0 := rand.Intn(maxEachNeighbour) + 1
		max1 := rand.Intn(maxEachNeighbour) + 1
		n0 := NewPot(nil, 0, nil)
		n1 := NewPot(nil, 0, nil)
		log.Trace(fmt.Sprintf("round %v: %v - %v", i, max0, max1))
		m := make(map[string]bool)
		for j := 0; j < max0; {
			v := randomtestAddr(keylen, j)
			// v := randomtestAddr(keylen, j)
			_, found := n0.Add(v)
			if !found {
				m[Label(v)] = false
				j++
			}
		}
		expAdded := 0

		for j := 0; j < max1; {
			v := randomtestAddr(keylen, j)
			// v := randomtestAddr(keylen, j)
			_, found := n1.Add(v)
			if !found {
				j++
			}
			_, found = m[Label(v)]
			if !found {
				expAdded++
				m[Label(v)] = false
			}
		}
		if i < 6 {
			continue
		}
		expSize := len(m)
		log.Trace(fmt.Sprintf("%v-0: pin: %v, size: %v", i, n0.Pin(), max0))
		log.Trace(fmt.Sprintf("%v-1: pin: %v, size: %v", i, n1.Pin(), max1))
		log.Trace(fmt.Sprintf("%v: merged tree size: %v, newly added: %v", i, expSize, expAdded))
		n, common := Union(n0, n1)
		added := n1.Size() - common
		size := n.Size()

		if expSize != size {
			t.Fatalf("%v: incorrect number of elements in merged pot, expected %v, got %v\n%v", i, expSize, size, n)
		}
		if expAdded != added {
			t.Fatalf("%v: incorrect number of added elements in merged pot, expected %v, got %v", i, expAdded, added)
		}
		if !checkDuplicates(n.pot) {
			t.Fatalf("%v: merged pot contains duplicates: \n%v", i, n)
		}
		for k := range m {
			_, found := n.Add(newTestAddr(k, 0))
			if !found {
				t.Fatalf("%v: merged pot (size:%v, added: %v) missing element %v\n%v", i, size, added, k, n)
			}
		}
	}
}

func checkDuplicates(t *pot) bool {
	po := -1
	for _, c := range t.bins {
		if c == nil {
			return false
		}
		if c.po <= po || !checkDuplicates(c) {
			return false
		}
		po = c.po
	}
	return true
}

func TestPotEachNeighbourSync(t *testing.T) {
	for i := 0; i < maxEachNeighbourTests; i++ {
		max := rand.Intn(maxEachNeighbour/2) + maxEachNeighbour/2
		pin := randomTestAddr(keylen, 0)
		n := NewPot(pin, 0, nil)
		m := make(map[string]bool)
		m[Label(pin)] = false
		for j := 1; j <= max; j++ {
			v := randomTestAddr(keylen, j)
			n.Add(v)
			m[Label(v)] = false
		}

		size := n.Size()
		if size < 2 {
			continue
		}
		count := rand.Intn(size/2) + size/2
		val := randomTestAddr(keylen, max+1)
		log.Trace(fmt.Sprintf("%v: pin: %v, size: %v, val: %v, count: %v", i, n.Pin(), size, val, count))
		err := testPotEachNeighbour(n, val, count, checkPo(val, n.pof), checkOrder(val), checkValues(m, val))
		if err != nil {
			t.Fatal(err)
		}
		minPoFound := keylen
		maxPoNotFound := 0
		for k, found := range m {
			po, _ := n.pof(val, newTestAddr(k, 0), 0)
			if found {
				if po < minPoFound {
					minPoFound = po
				}
			} else {
				if po > maxPoNotFound {
					maxPoNotFound = po
				}
			}
		}
		if minPoFound < maxPoNotFound {
			t.Fatalf("incorrect neighbours returned: found one with PO %v < there was one not found with PO %v", minPoFound, maxPoNotFound)
		}
	}
}

func TestPotEachNeighbourAsync(t *testing.T) {
	for i := 0; i < maxEachNeighbourTests; i++ {
		max := rand.Intn(maxEachNeighbour/2) + maxEachNeighbour/2
		n := NewPot(randomTestAddr(keylen, 0), 0, nil)
		size := 1
		for j := 1; j <= max; j++ {
			v := randomTestAddr(keylen, j)
			_, found := n.Add(v)
			if !found {
				size++
			}
		}
		if size != n.Size() {
			t.Fatal(n)
		}
		if size < 2 {
			continue
		}
		count := rand.Intn(size/2) + size/2
		val := randomTestAddr(keylen, max+1)

		mu := sync.Mutex{}
		m := make(map[string]bool)
		maxPos := rand.Intn(keylen)
		log.Trace(fmt.Sprintf("%v: pin: %v, size: %v, val: %v, count: %v, maxPos: %v", i, n.Pin(), size, val, count, maxPos))
		msize := 0
		remember := func(v Val, po int) error {
			if po > maxPos {
				return errNoCount
			}
			m[Label(v)] = true
			msize++
			return nil
		}
		if i == 0 {
			continue
		}
		testPotEachNeighbour(n, val, count, remember)
		d := 0
		forget := func(v Val, po int) {
			mu.Lock()
			defer mu.Unlock()
			d++
			delete(m, Label(v))
		}

		n.EachNeighbourAsync(val, count, maxPos, forget, true)
		if d != msize {
			t.Fatalf("incorrect number of neighbour calls in async iterator. expected %v, got %v", msize, d)
		}
		if len(m) != 0 {
			t.Fatalf("incorrect neighbour calls in async iterator. %v items missed:\n%v", len(m), n)
		}
	}
}

func benchmarkEachNeighbourSync(t *testing.B, max, count int, d time.Duration) {
	t.ReportAllocs()
	pin := randomTestAddr(keylen, 0)
	n := NewPot(pin, 0, nil)
	for j := 1; j <= max; {
		v := randomTestAddr(keylen, j)
		_, found := n.Add(v)
		if !found {
			j++
		}
	}
	t.ResetTimer()
	for i := 0; i < t.N; i++ {
		val := randomTestAddr(keylen, max+1)
		m := 0
		n.EachNeighbour(val, func(v Val, po int) bool {
			time.Sleep(d)
			m++
			if m == count {
				return false
			}
			return true
		})
	}
	t.StopTimer()
	stats := new(runtime.MemStats)
	runtime.ReadMemStats(stats)
	// fmt.Println(stats.Sys)
}

func benchmarkEachNeighbourAsync(t *testing.B, max, count int, d time.Duration) {
	t.ReportAllocs()
	pin := randomTestAddr(keylen, 0)
	n := NewPot(pin, 0, nil)
	for j := 1; j <= max; {
		v := randomTestAddr(keylen, j)
		_, found := n.Add(v)
		if !found {
			j++
		}
	}
	t.ResetTimer()
	for i := 0; i < t.N; i++ {
		val := randomTestAddr(keylen, max+1)
		n.EachNeighbourAsync(val, count, keylen, func(v Val, po int) {
			time.Sleep(d)
		}, true)
	}
	t.StopTimer()
	stats := new(runtime.MemStats)
	runtime.ReadMemStats(stats)
	// fmt.Println(stats.Sys)
}

func BenchmarkEachNeighbourSync_3_1_0(t *testing.B) {
	benchmarkEachNeighbourSync(t, 1000, 10, 1*time.Microsecond)
}
func BenchmarkEachNeighboursAsync_3_1_0(t *testing.B) {
	benchmarkEachNeighbourAsync(t, 1000, 10, 1*time.Microsecond)
}
func BenchmarkEachNeighbourSync_3_2_0(t *testing.B) {
	benchmarkEachNeighbourSync(t, 1000, 100, 1*time.Microsecond)
}
func BenchmarkEachNeighboursAsync_3_2_0(t *testing.B) {
	benchmarkEachNeighbourAsync(t, 1000, 100, 1*time.Microsecond)
}
func BenchmarkEachNeighbourSync_3_3_0(t *testing.B) {
	benchmarkEachNeighbourSync(t, 1000, 1000, 1*time.Microsecond)
}
func BenchmarkEachNeighboursAsync_3_3_0(t *testing.B) {
	benchmarkEachNeighbourAsync(t, 1000, 1000, 1*time.Microsecond)
}

func BenchmarkEachNeighbourSync_3_1_1(t *testing.B) {
	benchmarkEachNeighbourSync(t, 1000, 10, 2*time.Microsecond)
}
func BenchmarkEachNeighboursAsync_3_1_1(t *testing.B) {
	benchmarkEachNeighbourAsync(t, 1000, 10, 2*time.Microsecond)
}
func BenchmarkEachNeighbourSync_3_2_1(t *testing.B) {
	benchmarkEachNeighbourSync(t, 1000, 100, 2*time.Microsecond)
}
func BenchmarkEachNeighboursAsync_3_2_1(t *testing.B) {
	benchmarkEachNeighbourAsync(t, 1000, 100, 2*time.Microsecond)
}
func BenchmarkEachNeighbourSync_3_3_1(t *testing.B) {
	benchmarkEachNeighbourSync(t, 1000, 1000, 2*time.Microsecond)
}
func BenchmarkEachNeighboursAsync_3_3_1(t *testing.B) {
	benchmarkEachNeighbourAsync(t, 1000, 1000, 2*time.Microsecond)
}

func BenchmarkEachNeighbourSync_3_1_2(t *testing.B) {
	benchmarkEachNeighbourSync(t, 1000, 10, 4*time.Microsecond)
}
func BenchmarkEachNeighboursAsync_3_1_2(t *testing.B) {
	benchmarkEachNeighbourAsync(t, 1000, 10, 4*time.Microsecond)
}
func BenchmarkEachNeighbourSync_3_2_2(t *testing.B) {
	benchmarkEachNeighbourSync(t, 1000, 100, 4*time.Microsecond)
}
func BenchmarkEachNeighboursAsync_3_2_2(t *testing.B) {
	benchmarkEachNeighbourAsync(t, 1000, 100, 4*time.Microsecond)
}
func BenchmarkEachNeighbourSync_3_3_2(t *testing.B) {
	benchmarkEachNeighbourSync(t, 1000, 1000, 4*time.Microsecond)
}
func BenchmarkEachNeighboursAsync_3_3_2(t *testing.B) {
	benchmarkEachNeighbourAsync(t, 1000, 1000, 4*time.Microsecond)
}

func BenchmarkEachNeighbourSync_3_1_3(t *testing.B) {
	benchmarkEachNeighbourSync(t, 1000, 10, 8*time.Microsecond)
}
func BenchmarkEachNeighboursAsync_3_1_3(t *testing.B) {
	benchmarkEachNeighbourAsync(t, 1000, 10, 8*time.Microsecond)
}
func BenchmarkEachNeighbourSync_3_2_3(t *testing.B) {
	benchmarkEachNeighbourSync(t, 1000, 100, 8*time.Microsecond)
}
func BenchmarkEachNeighboursAsync_3_2_3(t *testing.B) {
	benchmarkEachNeighbourAsync(t, 1000, 100, 8*time.Microsecond)
}
func BenchmarkEachNeighbourSync_3_3_3(t *testing.B) {
	benchmarkEachNeighbourSync(t, 1000, 1000, 8*time.Microsecond)
}
func BenchmarkEachNeighboursAsync_3_3_3(t *testing.B) {
	benchmarkEachNeighbourAsync(t, 1000, 1000, 8*time.Microsecond)
}

func BenchmarkEachNeighbourSync_3_1_4(t *testing.B) {
	benchmarkEachNeighbourSync(t, 1000, 10, 16*time.Microsecond)
}
func BenchmarkEachNeighboursAsync_3_1_4(t *testing.B) {
	benchmarkEachNeighbourAsync(t, 1000, 10, 16*time.Microsecond)
}
func BenchmarkEachNeighbourSync_3_2_4(t *testing.B) {
	benchmarkEachNeighbourSync(t, 1000, 100, 16*time.Microsecond)
}
func BenchmarkEachNeighboursAsync_3_2_4(t *testing.B) {
	benchmarkEachNeighbourAsync(t, 1000, 100, 16*time.Microsecond)
}
func BenchmarkEachNeighbourSync_3_3_4(t *testing.B) {
	benchmarkEachNeighbourSync(t, 1000, 1000, 16*time.Microsecond)
}
func BenchmarkEachNeighboursAsync_3_3_4(t *testing.B) {
	benchmarkEachNeighbourAsync(t, 1000, 1000, 16*time.Microsecond)
}
