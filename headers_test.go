package spvwallet

import (
	"bytes"
	"crypto/rand"
	"github.com/boltdb/bolt"
	"github.com/btcsuite/btcd/chaincfg/chainhash"
	"github.com/btcsuite/btcd/wire"
	"math/big"
	"os"
	"strings"
	"testing"
)

var (
	header1Bytes []byte = []byte{0x00, 0x00, 0x00, 0x20, 0x02, 0x59, 0x46, 0xb0, 0x8b, 0x8d, 0x8a, 0xf9,
		0xb8, 0x2c, 0x16, 0xb6, 0xf0, 0x6e, 0x50, 0x12, 0xaa, 0x48, 0xdd, 0x18, 0x6a, 0x35, 0xcf, 0x1c,
		0x47, 0x08, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x6f, 0x81, 0x10, 0x30, 0x0d, 0x39, 0x6b, 0xb0,
		0x5b, 0x53, 0x8f, 0x9b, 0x0c, 0xac, 0x81, 0x4e, 0xa6, 0x55, 0x3c, 0x59, 0xdf, 0x00, 0xbc, 0xa7,
		0xb1, 0x8e, 0xc6, 0x9a, 0x5d, 0x97, 0xbd, 0xa2, 0x4a, 0x96, 0x4d, 0x58, 0x8b, 0xf9, 0x09, 0x1a,
		0xd0, 0xcb, 0x4e, 0xb6}
	header2Bytes []byte = []byte{0x00, 0x00, 0x00, 0x20, 0x41, 0x81, 0xef, 0x59, 0x0c, 0x87, 0x8a, 0x97,
		0x7b, 0xba, 0x99, 0xad, 0x50, 0x98, 0x17, 0xd1, 0xf3, 0x09, 0x3b, 0x1d, 0xcd, 0xbc, 0xb8, 0x8a,
		0x2a, 0x01, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x84, 0x15, 0x97, 0x0b, 0xdc, 0xc8, 0x35, 0x29,
		0x3a, 0x11, 0x0e, 0xe2, 0x38, 0x79, 0x74, 0x4b, 0x3e, 0x15, 0x38, 0xf5, 0x19, 0xa3, 0xf6, 0xf9,
		0x09, 0x8d, 0xa2, 0xda, 0x02, 0xa9, 0xd4, 0x33, 0xaa, 0x09, 0x4d, 0x58, 0x85, 0x8b, 0x03, 0x18,
		0xc6, 0xdd, 0xfe, 0x0e}
	header3Bytes []byte = []byte{0x00, 0x00, 0x00, 0x20, 0x44, 0x88, 0xea, 0x55, 0x00, 0x77, 0xaa, 0x90,
		0xbb, 0xba, 0x96, 0xa5, 0x52, 0x91, 0x27, 0xd4, 0xff, 0x99, 0x31, 0x1a, 0xad, 0xcc, 0x88, 0x8c,
		0x22, 0x11, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x84, 0x15, 0x97, 0x0b, 0xdc, 0xc8, 0x35, 0x29,
		0x3a, 0x11, 0x0e, 0xe2, 0x38, 0x79, 0x74, 0x4b, 0x3e, 0x15, 0x38, 0xf5, 0x19, 0xa3, 0xf6, 0xf9,
		0x09, 0x8d, 0xa2, 0xda, 0x02, 0xa9, 0xd4, 0x33, 0xaa, 0x09, 0x4d, 0x58, 0x85, 0x8b, 0x03, 0x18,
		0xc6, 0xdd, 0xfe, 0x00}
	testHdr1 wire.BlockHeader = wire.BlockHeader{}
	testHdr2 wire.BlockHeader = wire.BlockHeader{}
	testHdr3 wire.BlockHeader = wire.BlockHeader{}
	testSh1  StoredHeader
	testSh2  StoredHeader
	testSh3  StoredHeader
)

func init() {
	var buf bytes.Buffer
	buf.Write(header1Bytes)
	testHdr1.Deserialize(&buf)
	buf.Write(header2Bytes)
	testHdr2.Deserialize(&buf)
	buf.Write(header3Bytes)
	testHdr3.Deserialize(&buf)
	testSh1 = StoredHeader{
		header:    testHdr1,
		height:    100,
		totalWork: big.NewInt(500),
	}
	testSh2 = StoredHeader{
		header:    testHdr2,
		height:    200,
		totalWork: big.NewInt(1000),
	}
	testSh3 = StoredHeader{
		header:    testHdr3,
		height:    200,
		totalWork: big.NewInt(1000),
	}
}

func TestSerializeHeader(t *testing.T) {
	b, err := serializeHeader(testSh1)
	if err != nil {
		t.Error(err)
	}
	var buf bytes.Buffer
	testHdr1.Serialize(&buf)
	if !bytes.Equal(buf.Bytes(), b[:80]) {
		t.Error("Failed to serialize header")
	}
	if !bytes.Equal([]byte{0x00, 0x00, 0x00, 0x64}, b[80:84]) {
		t.Error("Failed to serialize height")
	}
	if !bytes.Equal([]byte{0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
		0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
		0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x01, 0xF4}, b[84:116]) {
		t.Error("Failed to serialize big int")
	}
}

func TestDeserializeHeader(t *testing.T) {
	height := []byte{0x00, 0x00, 0x00, 0x64}
	work := []byte{0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x01, 0xf4}
	sh, err := deserializeHeader(append(header1Bytes, append(height, work...)...))
	if err != nil {
		t.Error(err)
	}
	shHash := sh.header.BlockHash()
	testHdrHash := testHdr1.BlockHash()
	if !testHdrHash.IsEqual(&shHash) {
		t.Error("Failed to serialize block header")
	}
	if sh.height != 100 {
		t.Error("Failed to serialize height")
	}
	if sh.totalWork.Cmp(big.NewInt(500)) != 0 {
		t.Error("Failed to serialize total work")
	}
}

func TestHeaderDB_put(t *testing.T) {
	headers, err := NewHeaderDB("")
	if err != nil {
		t.Error(err)
	}
	// Test put with new tip
	err = headers.put(testSh1, true)
	if err != nil {
		t.Error(err)
	}
	err = headers.db.View(func(btx *bolt.Tx) error {
		hdrs := btx.Bucket(BKTHeaders)
		testHash := testSh1.header.BlockHash()
		b := hdrs.Get(testHash.CloneBytes())
		if b == nil {
			t.Error("Header doesn't exist in db")
		}
		ser, err := serializeHeader(testSh1)
		if err != nil {
			return err
		}
		if !bytes.Equal(ser, b) {
			t.Error("Failed to PUT header correctly")
		}
		tip := btx.Bucket(BKTChainTip)
		if err != nil {
			return err
		}
		b = tip.Get(KEYChainTip)
		if b == nil {
			t.Error("Best tip doesn't exist in db")
		}
		ser, err = serializeHeader(testSh1)
		if err != nil {
			return err
		}
		if !bytes.Equal(ser, b) {
			t.Error("Best tip failed to PUT header correctly")
		}
		return nil
	})
	if err != nil {
		t.Error(err)
	}

	// Test header without new tip
	err = headers.put(testSh2, false)
	if err != nil {
		t.Error(err)
	}
	err = headers.db.View(func(btx *bolt.Tx) error {
		hdrs := btx.Bucket(BKTHeaders)
		testHash := testSh2.header.BlockHash()
		b := hdrs.Get(testHash.CloneBytes())
		if b == nil {
			t.Error("Header doesn't exist in db")
		}
		ser, err := serializeHeader(testSh2)
		if err != nil {
			return err
		}
		if !bytes.Equal(ser, b) {
			t.Error("Failed to PUT header correctly")
		}
		tip := btx.Bucket(BKTChainTip)
		if err != nil {
			return err
		}
		b = tip.Get(KEYChainTip)
		if b == nil {
			t.Error("Best tip doesn't exist in db")
		}
		ser, err = serializeHeader(testSh1)
		if err != nil {
			return err
		}
		if !bytes.Equal(ser, b) {
			t.Error("Best tip failed to PUT header correctly")
		}
		return nil
	})
	if err != nil {
		t.Error(err)
	}
	// Test put duplicate
	err = headers.put(testSh2, true)
	if err != nil {
		t.Error(err)
	}
	os.RemoveAll("headers.bin")
}

func TestHeaderDB_GetPreviousHeader(t *testing.T) {
	headers, err := NewHeaderDB("")
	if err != nil {
		t.Error(err)
	}
	err = headers.put(testSh1, false)
	if err != nil {
		t.Error(err)
	}
	hdr, err := headers.GetPreviousHeader(testSh2.header)
	if err != nil {
		t.Error(err)
	}
	shHash := testSh1.header.BlockHash()
	testHdrHash := hdr.header.BlockHash()
	if !testHdrHash.IsEqual(&shHash) {
		t.Error("Get previous header returned incorrect header")
	}
	os.RemoveAll("headers.bin")
}

func TestHeaderDB_GetBestHeader(t *testing.T) {
	headers, err := NewHeaderDB("")
	if err != nil {
		t.Error(err)
	}
	err = headers.put(testSh1, false)
	if err != nil {
		t.Error(err)
	}
	hdr, err := headers.GetBestHeader()
	if err == nil {
		t.Error("Didn't receive error when fetching best header without one set")
	}

	err = headers.put(testSh1, true)
	if err != nil {
		t.Error(err)
	}
	err = headers.put(testSh2, false)
	if err != nil {
		t.Error(err)
	}
	hdr, err = headers.GetBestHeader()
	if err != nil {
		t.Error(err)
	}
	testSer, err := serializeHeader(testSh1)
	if err != nil {
		t.Error(err)
	}
	ser, err := serializeHeader(hdr)
	if err != nil {
		t.Error(err)
	}
	if !bytes.Equal(testSer, ser) {
		t.Error("Failed to fetch best header from the db")
	}
	os.RemoveAll("headers.bin")
}

func TestHeaderDB_Height(t *testing.T) {
	headers, err := NewHeaderDB("")
	if err != nil {
		t.Error(err)
	}
	err = headers.put(testSh1, true)
	if err != nil {
		t.Error(err)
	}
	height, err := headers.Height()
	if err != nil {
		t.Error(err)
	}
	if height != testSh1.height {
		t.Error("Returned incorrect height")
	}
	os.RemoveAll("headers.bin")
}

func TestHeaderDB_Prune(t *testing.T) {
	headers, err := NewHeaderDB("")
	if err != nil {
		t.Error(err)
	}
	var toDelete []chainhash.Hash
	var toStay []chainhash.Hash
	for i := 0; i < 2500; i++ {
		hdr := testSh1
		hdr.height = uint32(i)
		prev := make([]byte, 32)
		rand.Read(prev)
		prevHash, err := chainhash.NewHash(prev)
		if err != nil {
			t.Error(err)
		}
		hdr.header.PrevBlock = *prevHash
		err = headers.put(hdr, true)
		if err != nil {
			t.Error(err)
		}
		if i < 500 {
			toDelete = append(toDelete, hdr.header.BlockHash())
		} else {
			toStay = append(toStay, hdr.header.BlockHash())
		}
	}

	err = headers.Prune()
	if err != nil {
		t.Error(err)
	}
	err = headers.db.View(func(btx *bolt.Tx) error {
		hdrs := btx.Bucket(BKTHeaders)
		for _, hash := range toStay {
			b := hdrs.Get(hash.CloneBytes())
			if b == nil {
				t.Error("Pruned a header that should have stayed")
			}
		}
		for _, hash := range toDelete {
			b := hdrs.Get(hash.CloneBytes())
			if b != nil {
				t.Error("Failed to prune a header")
			}
		}

		return nil
	})
	if err != nil {
		t.Error(err)
	}
	os.RemoveAll("headers.bin")
}

func TestHeaderDB_Print(t *testing.T) {
	headers, err := NewHeaderDB("")
	if err != nil {
		t.Error(err)
	}
	// Test put with new tip
	err = headers.put(testSh1, true)
	if err != nil {
		t.Error(err)
	}
	err = headers.put(testSh2, true)
	if err != nil {
		t.Error(err)
	}
	err = headers.put(testSh3, true)
	if err != nil {
		t.Error(err)
	}
	var b bytes.Buffer
	headers.Print(&b)
	out := strings.Split(b.String(), "\n")
	if out[0] != `Height: 100.0, Hash: 000000000000012a8ab8bccd1d3b09f3d1179850ad99ba7b978a870c59ef8141, Parent: 00000000000008471ccf356a18dd48aa12506ef0b6162cb8f98a8d8bb0465902` {
		t.Error("Print function had incorrect return")
	}
	if out[1] != `Height: 200.0, Hash: 5f9e2cdf4dee12120f50f1b0e3086441a637fd09ff395dcf4e46735599633c4b, Parent: 00000000000011228c88ccad1a3199ffd4279152a596babb90aa770055ea8844` {
		t.Error("Print function had incorrect return")
	}
	if out[2] != `Height: 200.1, Hash: abe7bc6da630b6e0be7167ce23a4cf42d614543206e3725d21ebe829618a94af, Parent: 000000000000012a8ab8bccd1d3b09f3d1179850ad99ba7b978a870c59ef8141` {
		t.Error("Print function had incorrect return")
	}
	os.RemoveAll("headers.bin")
}
