package hasher

import (
	"context"
	"fmt"
	"math/rand"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethersphere/swarm/bmt"
	"github.com/ethersphere/swarm/file"
	"github.com/ethersphere/swarm/file/testutillocal"
	"github.com/ethersphere/swarm/log"
	"github.com/ethersphere/swarm/testutil"
	"golang.org/x/crypto/sha3"
)

const (
	zeroHex = "0000000000000000000000000000000000000000000000000000000000000000"
)

// TestTreeParams verifies that params are set correctly by the param constructor
func TestTreeParams(t *testing.T) {

	params := newTreeParams(dummyHashFunc)

	if params.SectionSize != 32 {
		t.Fatalf("section: expected %d, got %d", sectionSize, params.SectionSize)
	}

	if params.Branches != 128 {
		t.Fatalf("branches: expected %d, got %d", branches, params.SectionSize)
	}

	if params.Spans[2] != branches*branches {
		t.Fatalf("span %d: expected %d, got %d", 2, branches*branches, params.Spans[1])
	}

}

// TestTarget verifies that params are set correctly by the target constructor
func TestTarget(t *testing.T) {

	tgt := newTarget()
	tgt.Set(32, 1, 2)

	if tgt.size != 32 {
		t.Fatalf("target size expected %d, got %d", 32, tgt.size)
	}

	if tgt.sections != 1 {
		t.Fatalf("target sections expected %d, got %d", 1, tgt.sections)
	}

	if tgt.level != 2 {
		t.Fatalf("target level expected %d, got %d", 2, tgt.level)
	}
}

// TestJobTargetWithinJobDefault verifies the calculation of whether a final data section index
// falls within a particular job's span without regard to differing SectionSize
func TestJobTargetWithinDefault(t *testing.T) {
	params := newTreeParams(dummyHashFunc)
	index := newJobIndex(9)
	tgt := newTarget()

	jb := newJob(params, tgt, index, 1, branches*branches)
	defer jb.destroy()

	finalSize := chunkSize*branches + chunkSize*2
	finalCount := dataSizeToSectionCount(finalSize, sectionSize)
	log.Trace("within test", "size", finalSize, "count", finalCount)
	c, ok := jb.targetWithinJob(finalCount - 1)
	if !ok {
		t.Fatalf("target %d within %d: expected true", finalCount, jb.level)
	}
	if c != 1 {
		t.Fatalf("target %d within %d: expected %d, got %d", finalCount, jb.level, 2, c)
	}
}

// TestJobTargetWithinDifferentSections does the same as TestTargetWithinJobDefault but
// with SectionSize/Branches settings differeing between client target and underlying writer
func TestJobTargetWithinDifferentSections(t *testing.T) {
	dummyHashDoubleFunc := func(_ context.Context) file.SectionWriter {
		return newDummySectionWriter(chunkSize, sectionSize*2, sectionSize*2, branches/2)
	}
	params := newTreeParams(dummyHashDoubleFunc)
	index := newJobIndex(9)
	tgt := newTarget()

	//jb := newJob(params, tgt, index, 1, branches*branches)
	jb := newJob(params, tgt, index, 1, 0)
	defer jb.destroy()

	//finalSize := chunkSize*branches + chunkSize*2
	finalSize := chunkSize
	finalCount := dataSizeToSectionCount(finalSize, sectionSize)
	log.Trace("within test", "size", finalSize, "count", finalCount)
	c, ok := jb.targetWithinJob(finalCount - 1)
	if !ok {
		t.Fatalf("target %d within %d: expected true", finalCount, jb.level)
	}
	if c != 1 {
		t.Fatalf("target %d within %d: expected %d, got %d", finalCount, jb.level, 1, c)
	}
}

// TestNewJob verifies that a job is initialized with the correct values
func TestNewJob(t *testing.T) {

	params := newTreeParams(dummyHashFunc)
	params.Debug = true

	tgt := newTarget()
	jb := newJob(params, tgt, nil, 1, branches*branches+1)
	if jb.level != 1 {
		t.Fatalf("job level expected 1, got %d", jb.level)
	}
	if jb.dataSection != branches*branches+1 {
		t.Fatalf("datasectionindex: expected %d, got %d", branches+1, jb.dataSection)
	}
	tgt.Set(0, 0, 0)
	jb.destroy()
}

// TestJobSize verifies the data size calculation used for calculating the span of data
// under a particular level reference
// it tests both a balanced and an unbalanced tree
func TestJobSize(t *testing.T) {
	params := newTreeParams(dummyHashFunc)
	params.Debug = true
	index := newJobIndex(9)

	tgt := newTarget()
	jb := newJob(params, tgt, index, 3, 0)
	jb.cursorSection = 1
	jb.endCount = 1
	size := chunkSize*branches + chunkSize
	sections := dataSizeToSectionIndex(size, sectionSize) + 1
	tgt.Set(size, sections, 3)
	jobSize := jb.size()
	if jobSize != size {
		t.Fatalf("job size: expected %d, got %d", size, jobSize)
	}
	jb.destroy()

	tgt = newTarget()
	jb = newJob(params, tgt, index, 3, 0)
	jb.cursorSection = 1
	jb.endCount = 1
	size = chunkSize * branches * branches
	sections = dataSizeToSectionIndex(size, sectionSize) + 1
	tgt.Set(size, sections, 3)
	jobSize = jb.size()
	if jobSize != size {
		t.Fatalf("job size: expected %d, got %d", size, jobSize)
	}
	jb.destroy()

}

// TestJobTarget verifies that the underlying calculation for determining whether
// a data section index is within a level's span is correct
func TestJobTarget(t *testing.T) {
	tgt := newTarget()
	params := newTreeParams(dummyHashFunc)
	params.Debug = true
	index := newJobIndex(9)

	jb := newJob(params, tgt, index, 1, branches*branches)

	// this is less than chunksize * 128
	// it will not be in the job span
	finalSize := chunkSize + sectionSize + 1
	finalSection := dataSizeToSectionIndex(finalSize, sectionSize)
	c, ok := jb.targetWithinJob(finalSection)
	if ok {
		t.Fatalf("targetwithinjob: expected false")
	}
	jb.destroy()

	// chunkSize*128+chunkSize*2 (532480) is within chunksize*128 (524288) and chunksize*128*2 (1048576)
	// it will be within the job span
	finalSize = chunkSize*branches + chunkSize*2
	finalSection = dataSizeToSectionIndex(finalSize, sectionSize)
	c, ok = jb.targetWithinJob(finalSection)
	if !ok {
		t.Fatalf("targetwithinjob section %d: expected true", branches*branches)
	}
	if c != 1 {
		t.Fatalf("targetwithinjob section %d: expected %d, got %d", branches*branches, 1, c)
	}
	c = jb.targetCountToEndCount(finalSection + 1)
	if c != 2 {
		t.Fatalf("targetcounttoendcount section %d: expected %d, got %d", branches*branches, 2, c)
	}
	jb.destroy()
}

// TestJobIndex verifies that the job constructor adds the job to the job index
// and removes it on job destruction
func TestJobIndex(t *testing.T) {
	tgt := newTarget()
	params := newTreeParams(dummyHashFunc)

	jb := newJob(params, tgt, nil, 1, branches)
	jobIndex := jb.index
	jbGot := jobIndex.Get(1, branches)
	if jb != jbGot {
		t.Fatalf("jbIndex get: expect %p, got %p", jb, jbGot)
	}
	jbGot.destroy()
	if jobIndex.Get(1, branches) != nil {
		t.Fatalf("jbIndex delete: expected nil")
	}
}

// TestJobGetNext verifies that the new job constructed through the job.Next() method
// has the correct level and data section index
func TestJobGetNext(t *testing.T) {
	tgt := newTarget()
	params := newTreeParams(dummyHashFunc)
	params.Debug = true

	jb := newJob(params, tgt, nil, 1, branches*branches)
	jbn := jb.Next()
	if jbn == nil {
		t.Fatalf("parent: nil")
	}
	if jbn.level != 1 {
		t.Fatalf("nextjob level: expected %d, got %d", 2, jbn.level)
	}
	if jbn.dataSection != jb.dataSection+branches*branches {
		t.Fatalf("nextjob section: expected %d, got %d", jb.dataSection+branches*branches, jbn.dataSection)
	}
}

// TestJobWriteTwoAndFinish writes two references to a job and sets the job target to two chunks
// it verifies that the job count after the writes is two, and the hash is correct
func TestJobWriteTwoAndFinish(t *testing.T) {

	tgt := newTarget()
	params := newTreeParams(dummyHashFunc)

	jb := newJob(params, tgt, nil, 1, 0)
	jb.start()
	_, data := testutil.SerialData(sectionSize*2, 255, 0)
	jb.write(0, data[:sectionSize])
	jb.write(1, data[sectionSize:])

	finalSize := chunkSize * 2
	finalSection := dataSizeToSectionIndex(finalSize, sectionSize)
	tgt.Set(finalSize, finalSection-1, 2)

	ctx, cancel := context.WithTimeout(context.Background(), time.Millisecond*199)
	defer cancel()
	select {
	case ref := <-tgt.Done():
		correctRefHex := "0xe1553e1a3a6b73f96e6fc48318895e401e7db2972962ee934633fa8b3eaaf78b"
		refHex := hexutil.Encode(ref)
		if refHex != correctRefHex {
			t.Fatalf("job write two and finish: expected %s, got %s", correctRefHex, refHex)
		}
	case <-ctx.Done():
		t.Fatalf("timeout: %v", ctx.Err())
	}

	if jb.count() != 2 {
		t.Fatalf("jobcount: expected %d, got %d", 2, jb.count())
	}
}

// TestJobGetParent verifies that the parent returned from two jobs' parent() calls
// that are within the same span as the parent chunk of references is the same
// BUG: not guaranteed to return same parent when run with eg -count 100
func TestJobGetParent(t *testing.T) {
	tgt := newTarget()
	params := newTreeParams(dummyHashFunc)

	jb := newJob(params, tgt, nil, 1, branches*branches)
	jb.start()
	jbp := jb.parent()
	if jbp == nil {
		t.Fatalf("parent: nil")
	}
	if jbp.level != 2 {
		t.Fatalf("parent level: expected %d, got %d", 2, jbp.level)
	}
	if jbp.dataSection != 0 {
		t.Fatalf("parent data section: expected %d, got %d", 0, jbp.dataSection)
	}
	jbGot := jb.index.Get(2, 0)
	if jbGot == nil {
		t.Fatalf("index get: nil")
	}

	jbNext := jb.Next()
	jbpNext := jbNext.parent()
	if jbpNext != jbp {
		t.Fatalf("next parent: expected %p, got %p", jbp, jbpNext)
	}
}

// TestJobWriteParentSection verifies that a data write translates to a write
// in the correct section of its parent
func TestJobWriteParentSection(t *testing.T) {
	tgt := newTarget()
	params := newTreeParams(dummyHashFunc)
	index := newJobIndex(9)

	jb := newJob(params, tgt, index, 1, 0)
	jbn := jb.Next()
	_, data := testutil.SerialData(sectionSize*2, 255, 0)
	jbn.write(0, data[:sectionSize])
	jbn.write(1, data[sectionSize:])

	finalSize := chunkSize*branches + chunkSize*2
	finalSection := dataSizeToSectionIndex(finalSize, sectionSize)
	tgt.Set(finalSize, finalSection, 3)

	ctx, cancel := context.WithTimeout(context.Background(), time.Millisecond*10)
	defer cancel()
	select {
	case <-tgt.Done():
		t.Fatalf("unexpected done")
	case <-ctx.Done():
	}
	jbnp := jbn.parent()
	if jbnp.count() != 1 {
		t.Fatalf("parent count: expected %d, got %d", 1, jbnp.count())
	}
	correctRefHex := "0xe1553e1a3a6b73f96e6fc48318895e401e7db2972962ee934633fa8b3eaaf78b"

	// extract data in section 2 from the writer
	// TODO: overload writer to provide a get method to extract data to improve clarity
	w := jbnp.writer.(*dummySectionWriter)
	w.mu.Lock()
	parentRef := w.data[32:64]
	w.mu.Unlock()
	parentRefHex := hexutil.Encode(parentRef)
	if parentRefHex != correctRefHex {
		t.Fatalf("parent data: expected %s, got %s", correctRefHex, parentRefHex)
	}
}

// TestJobWriteFull verifies the hashing result of the write of a balanced tree
// where the simulated tree is chunkSize*branches worth of data
func TestJobWriteFull(t *testing.T) {

	tgt := newTarget()
	params := newTreeParams(dummyHashFunc)

	jb := newJob(params, tgt, nil, 1, 0)
	jb.start()
	_, data := testutil.SerialData(chunkSize, 255, 0)
	for i := 0; i < chunkSize; i += sectionSize {
		jb.write(i/sectionSize, data[i:i+sectionSize])
	}

	tgt.Set(chunkSize, branches, 2)
	ctx, cancel := context.WithTimeout(context.Background(), time.Millisecond*10)
	defer cancel()
	select {
	case ref := <-tgt.Done():
		correctRefHex := "0x1ed31ae32fe570b69b01f800fbdec1f17b7ea6f0348216d6d79df91ddf28344e"
		refHex := hexutil.Encode(ref)
		if refHex != correctRefHex {
			t.Fatalf("job write full: expected %s, got %s", correctRefHex, refHex)
		}
	case <-ctx.Done():
		t.Fatalf("timeout: %v", ctx.Err())
	}
	if jb.count() != branches {
		t.Fatalf("jobcount: expected %d, got %d", 32, jb.count())
	}
}

// TestJobWriteSpan uses the bmt asynchronous hasher
// it verifies that a result can be attained at chunkSize+sectionSize*2 references
// which translates to chunkSize*branches+chunkSize*2 bytes worth of data
func TestJobWriteSpan(t *testing.T) {

	tgt := newTarget()
	hashFunc := testutillocal.NewBMTHasherFunc(0)
	params := newTreeParams(hashFunc)

	jb := newJob(params, tgt, nil, 1, 0)
	jb.start()
	_, data := testutil.SerialData(chunkSize+sectionSize*2, 255, 0)

	for i := 0; i < chunkSize; i += sectionSize {
		jb.write(i/sectionSize, data[i:i+sectionSize])
	}
	jbn := jb.Next()
	jbn.write(0, data[chunkSize:chunkSize+sectionSize])
	jbn.write(1, data[chunkSize+sectionSize:])
	finalSize := chunkSize*branches + chunkSize*2
	finalSection := dataSizeToSectionIndex(finalSize, sectionSize)
	tgt.Set(finalSize, finalSection, 3)

	ctx, cancel := context.WithTimeout(context.Background(), time.Millisecond*1000)
	defer cancel()
	select {
	case ref := <-tgt.Done():
		// TODO: double check that this hash if correct!!
		refCorrectHex := "0xee56134cab34a5a612648dcc22d88b7cb543081bd144906dfc4fa93802c9addf"
		refHex := hexutil.Encode(ref)
		if refHex != refCorrectHex {
			t.Fatalf("writespan sequential: expected %s, got %s", refCorrectHex, refHex)
		}
	case <-ctx.Done():
		t.Fatalf("timeout: %v", ctx.Err())
	}

	sz := jb.size()
	if sz != chunkSize*branches {
		t.Fatalf("job 1 size: expected %d, got %d", chunkSize, sz)
	}

	sz = jbn.size()
	if sz != chunkSize*2 {
		t.Fatalf("job 2 size: expected %d, got %d", sectionSize, sz)
	}
}

// TestJobWriteSpanShuffle does the same as TestJobWriteSpan but
// shuffles the indices of the first chunk write
// verifying that sequential use of the underlying hasher is not required
func TestJobWriteSpanShuffle(t *testing.T) {

	tgt := newTarget()
	hashFunc := testutillocal.NewBMTHasherFunc(0)
	params := newTreeParams(hashFunc)

	jb := newJob(params, tgt, nil, 1, 0)
	jb.start()
	_, data := testutil.SerialData(chunkSize+sectionSize*2, 255, 0)

	var idxs []int
	for i := 0; i < branches; i++ {
		idxs = append(idxs, i)
	}
	rand.Shuffle(branches, func(i int, j int) {
		idxs[i], idxs[j] = idxs[j], idxs[i]
	})
	for _, idx := range idxs {
		jb.write(idx, data[idx*sectionSize:idx*sectionSize+sectionSize])
	}

	jbn := jb.Next()
	jbn.write(0, data[chunkSize:chunkSize+sectionSize])
	jbn.write(1, data[chunkSize+sectionSize:])
	finalSize := chunkSize*branches + chunkSize*2
	finalSection := dataSizeToSectionIndex(finalSize, sectionSize)
	tgt.Set(finalSize, finalSection, 3)

	ctx, cancel := context.WithTimeout(context.Background(), time.Millisecond*100)
	defer cancel()
	select {
	case ref := <-tgt.Done():
		refCorrectHex := "0xee56134cab34a5a612648dcc22d88b7cb543081bd144906dfc4fa93802c9addf"
		refHex := hexutil.Encode(ref)
		jbparent := jb.parent()
		jbnparent := jbn.parent()
		log.Info("succeeding", "jb count", jb.count(), "jbn count", jbn.count(), "jb parent count", jbparent.count(), "jbn parent count", jbnparent.count())
		if refHex != refCorrectHex {
			t.Fatalf("writespan sequential: expected %s, got %s", refCorrectHex, refHex)
		}
	case <-ctx.Done():

		jbparent := jb.parent()
		jbnparent := jbn.parent()
		log.Error("failing", "jb count", jb.count(), "jbn count", jbn.count(), "jb parent count", jbparent.count(), "jbn parent count", jbnparent.count(), "jb parent p", fmt.Sprintf("%p", jbparent), "jbn parent p", fmt.Sprintf("%p", jbnparent))
		t.Fatalf("timeout: %v", ctx.Err())
	}

	sz := jb.size()
	if sz != chunkSize*branches {
		t.Fatalf("job size: expected %d, got %d", chunkSize*branches, sz)
	}

	sz = jbn.size()
	if sz != chunkSize*2 {
		t.Fatalf("job size: expected %d, got %d", chunkSize*branches, sz)
	}
}

// TestVectors executes the barebones functionality of the hasher
// and verifies against source of truth results generated from the reference hasher
// for the same data
// TODO: vet dynamically against the referencefilehasher instead of expect vector
func TestJobVector(t *testing.T) {
	poolSync := bmt.NewTreePool(sha3.NewLegacyKeccak256, branches, bmt.PoolSize)
	dataHash := bmt.New(poolSync)
	hashFunc := testutillocal.NewBMTHasherFunc(0)
	params := newTreeParams(hashFunc)
	var mismatch int

	for i := start; i < end; i++ {
		tgt := newTarget()
		dataLength := dataLengths[i]
		_, data := testutil.SerialData(dataLength, 255, 0)
		jb := newJob(params, tgt, nil, 1, 0)
		jb.start()
		count := 0
		log.Info("test vector", "length", dataLength)
		for i := 0; i < dataLength; i += chunkSize {
			ie := i + chunkSize
			if ie > dataLength {
				ie = dataLength
			}
			writeSize := ie - i
			dataHash.Reset()
			dataHash.SetSpan(writeSize)
			c, err := dataHash.Write(data[i:ie])
			if err != nil {
				jb.destroy()
				t.Fatalf("data ref fail: %v", err)
			}
			if c != ie-i {
				jb.destroy()
				t.Fatalf("data ref short write: expect %d, got %d", ie-i, c)
			}
			ref := dataHash.Sum(nil) //, writeSize)
			log.Debug("data ref", "i", i, "ie", ie, "data", hexutil.Encode(ref))
			jb.write(count, ref)
			count += 1
			if ie%(chunkSize*branches) == 0 {
				jb = jb.Next()
				count = 0
			}
		}
		dataSections := dataSizeToSectionIndex(dataLength, params.SectionSize)
		tgt.Set(dataLength, dataSections, getLevelsFromLength(dataLength, params.SectionSize, params.Branches))
		eq := true
		ctx, cancel := context.WithTimeout(context.Background(), time.Millisecond*1000)
		defer cancel()
		select {
		case ref := <-tgt.Done():
			refCorrectHex := "0x" + expected[i]
			refHex := hexutil.Encode(ref)
			if refHex != refCorrectHex {
				mismatch++
				eq = false
			}
			t.Logf("[%7d+%4d]\t%v\tref: %x\texpect: %s", dataLength/chunkSize, dataLength%chunkSize, eq, ref, expected[i])
		case <-ctx.Done():
			t.Fatalf("timeout: %v", ctx.Err())
		}

	}
	if mismatch > 0 {
		t.Fatalf("mismatches: %d/%d", mismatch, end-start)
	}
}

// BenchmarkVector generates benchmarks that are comparable to the pyramid hasher
func BenchmarkJob(b *testing.B) {
	for i := start; i < end; i++ {
		b.Run(fmt.Sprintf("%d/%d", i, dataLengths[i]), benchmarkJob)
	}
}

func benchmarkJob(b *testing.B) {
	params := strings.Split(b.Name(), "/")
	dataLengthParam, err := strconv.ParseInt(params[2], 10, 64)
	if err != nil {
		b.Fatal(err)
	}
	dataLength := int(dataLengthParam)

	poolSync := bmt.NewTreePool(sha3.NewLegacyKeccak256, branches, bmt.PoolSize)
	dataHash := bmt.New(poolSync)
	hashFunc := testutillocal.NewBMTHasherFunc(0)
	treeParams := newTreeParams(hashFunc)
	_, data := testutil.SerialData(dataLength, 255, 0)

	for j := 0; j < b.N; j++ {
		tgt := newTarget()
		jb := newJob(treeParams, tgt, nil, 1, 0)
		jb.start()
		count := 0
		//log.Info("test vector", "length", dataLength)
		for i := 0; i < dataLength; i += chunkSize {
			ie := i + chunkSize
			if ie > dataLength {
				ie = dataLength
			}
			//writeSize := ie - i
			dataHash.Reset()
			//dataHash.SetLength(writeSize)
			c, err := dataHash.Write(data[i:ie])
			if err != nil {
				jb.destroy()
				b.Fatalf("data ref fail: %v", err)
			}
			if c != ie-i {
				jb.destroy()
				b.Fatalf("data ref short write: expect %d, got %d", ie-i, c)
			}
			ref := dataHash.Sum(nil)
			jb.write(count, ref)
			count += 1
			if ie%(chunkSize*branches) == 0 {
				jb = jb.Next()
				count = 0
			}
		}
		dataSections := dataSizeToSectionIndex(dataLength, treeParams.SectionSize)
		tgt.Set(dataLength, dataSections, getLevelsFromLength(dataLength, treeParams.SectionSize, treeParams.Branches))
		ctx, cancel := context.WithTimeout(context.Background(), time.Millisecond*1000)
		defer cancel()
		select {
		case <-tgt.Done():
		case <-ctx.Done():
			b.Fatalf("timeout: %v", ctx.Err())
		}
	}
}