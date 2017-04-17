package spvwallet

import (
	"bytes"
	"crypto/rand"
	"encoding/hex"
	"github.com/btcsuite/btcd/blockchain"
	"github.com/btcsuite/btcd/chaincfg"
	"github.com/btcsuite/btcd/chaincfg/chainhash"
	"github.com/btcsuite/btcd/wire"
	"math/big"
	"os"
	"testing"
	"time"
)

// New chain starting from regtest genesis
var chain = []string{
	"0000002006226e46111a0b59caaf126043eb5bbf28c34f3a5e332a1fc7b2b73cf188910fc3ed4523bf94fc1fa184bee85af604c9ebeea6b39b498f62703fd3f03e7475534658d158ffff7f2001000000",
	"000000207c3d2d417ff34a46f4f11a972d8e32bc98b300112dd4d9a1dae9ff87468eae136b90f1757adfab2056d693160b417b8f87a65c2c0735a47e63768f26473905506059d158ffff7f2003000000",
	"000000200c6ea2eaf928b2d5d080c2f36dac1185865db1289c7339834b98e8034e4274073ed977491ebe6f9c0e01f5796e36ed66bf4e410bbbc2635129d6e0ecfc1897908459d158ffff7f2001000000",
	"000000202e1569563ff6463f65bb7669b35fb9dd95ba0b251e30251b9877d9578b8700680337ff38b71d9667190c99e8fae337ba8c9c40cbd2c4678ba71d81cf6d3a1aa2ac59d158ffff7f2001000000",
	"000000204525edcccf706e3769a54c8772934f291d6810315a26c177862c66feb9f3896e090c84be811cfdfed6da043cb337fccecff95fc73810ca82adb3d032b5d49140c759d158ffff7f2000000000",
	"00000020ada1a9efa81df10d7b430e2fd5f3b085180c91b0e9b0f6e9af2d9b733544015eab404ef503e538909a04a419499133af9bcee47fcfc84baaab5344f77ebd455dec59d158ffff7f2000000000",
	"000000204fdcb9ca4cc47ae7485bfc2f8adcbd515b1ee0cb724d343c91f02b6ec5a0ba507dddd2639fc1bd522489a2c2f2b681a60c6c7939490458dc1c008f3217cb47d6035ad158ffff7f2001000000",
	"0000002019dbc9a6cec93be207053e4dfbc63af20c3cedba68f890c5a90f27aeb2ecc73386692b64e16ea4b87fc877cb3762394d12b597a0ca8d5efb2ea2c6e163f9e4c8225ad158ffff7f2000000000",
	"000000203afc4a1c100fe3e21fa24ef92857613bb00890564e3529623780bc8d4a86d15cfd35aef39950dc53c348b5013f4ee3d94afc16745d6b3c8a9e6acfb8a2641c6f3e5ad158ffff7f2000000000",
	"000000200e1b58feab56f9fe5ed7484a8c7bfecdb270da528db7a805d18208891bde3726a5ccb0a073d0cc7402ac89f4bb4b64c39bc365bfee7ccd7ea3a24996ee684c775a5ad158ffff7f2000000000",
}

// Forks `chain` starting at block 6
var fork = []string{
	"00000020ada1a9efa81df10d7b430e2fd5f3b085180c91b0e9b0f6e9af2d9b733544015eead915a2f4521c58cb1c42a469aefede5a9d1dddfe8ccc408f8135fc2560f25a096dd158ffff7f20e9aace03",
	"0000002097e3603b40c0c7add951e3a7dba5088836d17e1123ef7cffdd60174e3dce0024cffe0c74189d854a778a3e57fee8510103e83d95b221b8bfe1159806b3bde27e236dd158ffff7f20794caff6",
	"0000002085a3bf0898ed1cad9e868120c8e044673425a13ecc7ab2daec204ca9190e643ca32434566054789e79214a7cb7c1b6e37084cbfce7564d4aabb10ef6fc1d655c3d6dd158ffff7f20c2e4cb6f",
	"000000209aa626e76fbcfc08bc1626a0a9bc7b82d8521de22a477e7b377d8f83be8d446a05aae352ffe9f09af1d79d24992dbee2785b3fe4eb4a0e21e7a3b26a90115dac536dd158ffff7f201d2f76eb",
	"000000208d6d636589b4056d1486fbcc0b46adefbb770b7e6a8d668fe65c3f58f5c2c70934008f98664ffec01f583870f843b617c869ec30f1b37723b3d0f0d4a3ba6a88686dd158ffff7f209d12ee06",
	"0000002067cf05afedc2b5956c10845006358fe480893e1199a0c0e2b70d5ecf2787af760385ca3d191d1800cd7b6a56d8b44853109f3e5983a94c7e10818541278ec6027b6dd158ffff7f2004e2c75c",
	"00000020b2227c6c858a36af167d9667dcf4f58df604ab7962a660d69d233a63e7269f06ecb669fff090b7f2f6952d52c96ca0c8abe1e266d9740f8548eeb10eea9e3536906dd158ffff7f20c0ac3d1e",
}

func createBlockChain(bc *Blockchain) error {
	best, err := bc.db.GetBestHeader()
	if err != nil {
		return err
	}
	last := best.header
	for i := 0; i < 2015; i++ {
		hdr := wire.BlockHeader{}
		hdr.PrevBlock = last.BlockHash()
		hdr.Nonce = 0
		hdr.Timestamp = time.Now().Add(time.Minute * time.Duration(i))
		mr := make([]byte, 32)
		rand.Read(mr)
		ch, err := chainhash.NewHash(mr)
		if err != nil {
			return err
		}
		hdr.MerkleRoot = *ch
		hdr.Bits = best.header.Bits
		hdr.Version = 3
		sh := StoredHeader{
			header:    hdr,
			height:    uint32(i) + 1,
			totalWork: big.NewInt(0),
		}
		bc.db.Put(sh, true)
		last = hdr
	}
	return nil
}

func TestNewBlockchain(t *testing.T) {
	bc, err := NewBlockchain("", &chaincfg.MainNetParams)
	if err != nil {
		t.Error(err)
	}
	best, err := bc.db.GetBestHeader()
	if err != nil {
		t.Error(err)
	}
	bestHash := best.header.BlockHash()
	checkHash := mainnetCheckpoint.BlockHash()
	if !bestHash.IsEqual(&checkHash) {
		t.Error("Blockchain failed to initialize with correct mainnet checkpoint")
	}
	if best.height != MAINNET_CHECKPOINT_HEIGHT {
		t.Error("Blockchain failed to initialized with correct mainnet checkpoint height")
	}
	if best.totalWork.Uint64() != 0 {
		t.Error("Blockchain failed to initialized with correct mainnet total work")
	}
	os.RemoveAll("headers.bin")
	bc, err = NewBlockchain("", &chaincfg.TestNet3Params)
	if err != nil {
		t.Error(err)
	}
	best, err = bc.db.GetBestHeader()
	if err != nil {
		t.Error(err)
	}
	bestHash = best.header.BlockHash()
	checkHash = testnet3Checkpoint.BlockHash()
	if !bestHash.IsEqual(&checkHash) {
		t.Error("Blockchain failed to initialize with correct testnet checkpoint")
	}
	if best.height != TESTNET3_CHECKPOINT_HEIGHT {
		t.Error("Blockchain failed to initialized with correct testnet checkpoint height")
	}
	if best.totalWork.Uint64() != 0 {
		t.Error("Blockchain failed to initialized with correct testnet total work")
	}
	os.RemoveAll("headers.bin")
	bc, err = NewBlockchain("", &chaincfg.RegressionNetParams)
	if err != nil {
		t.Error(err)
	}
	best, err = bc.db.GetBestHeader()
	if err != nil {
		t.Error(err)
	}
	bestHash = best.header.BlockHash()
	checkHash = regtestCheckpoint.BlockHash()
	if !bestHash.IsEqual(&checkHash) {
		t.Error("Blockchain failed to initialize with correct regtest checkpoint")
	}
	if best.height != REGTEST_CHECKPOINT_HEIGHT {
		t.Error("Blockchain failed to initialized with correct regtest checkpoint height")
	}
	if best.totalWork.Uint64() != 0 {
		t.Error("Blockchain failed to initialized with correct regtest total work")
	}
	os.RemoveAll("headers.bin")
}

func TestBlockchain_CommitHeader(t *testing.T) {
	bc, err := NewBlockchain("", &chaincfg.RegressionNetParams)
	if err != nil {
		t.Error(err)
	}
	var headers = []wire.BlockHeader{regtestCheckpoint}
	for i, c := range chain {
		b, err := hex.DecodeString(c)
		if err != nil {
			t.Error(err)
		}
		var hdr wire.BlockHeader
		hdr.Deserialize(bytes.NewReader(b))
		newTip, reorg, height, err := bc.CommitHeader(hdr)
		if err != nil {
			t.Error()
		}
		if !newTip {
			t.Error("Failed to set new tip when inserting header")
		}
		if reorg != nil {
			t.Error("Incorrectly set reorg when inserting header")
		}
		if height != uint32(i+1) {
			t.Error("Returned incorrect height when inserting header")
		}
		headers = append(headers, hdr)
	}
	best, err := bc.db.GetBestHeader()
	if err != nil {
		t.Error(err)
	}
	for i := len(headers) - 1; i >= 0; i-- {
		putHash := headers[i].BlockHash()
		retHash := best.header.BlockHash()
		if !putHash.IsEqual(&retHash) {
			t.Error("Header put failed")
		}
		best, err = bc.db.GetPreviousHeader(best.header)
	}
	os.RemoveAll("headers.bin")
}

func Test_Reorg(t *testing.T) {
	bc, err := NewBlockchain("", &chaincfg.RegressionNetParams)
	if err != nil {
		t.Error(err)
	}
	var headers = []wire.BlockHeader{regtestCheckpoint}
	for i, c := range chain {
		b, err := hex.DecodeString(c)
		if err != nil {
			t.Error(err)
		}
		var hdr wire.BlockHeader
		hdr.Deserialize(bytes.NewReader(b))
		newTip, reorg, height, err := bc.CommitHeader(hdr)
		if err != nil {
			t.Error()
		}
		if !newTip {
			t.Error("Failed to set new tip when inserting header")
		}
		if reorg != nil {
			t.Error("Incorrectly set reorg when inserting header")
		}
		if height != uint32(i+1) {
			t.Error("Returned incorrect height when inserting header")
		}
		if i < 5 {
			headers = append(headers, hdr)
		}
	}
	for i, c := range fork {
		b, err := hex.DecodeString(c)
		if err != nil {
			t.Error(err)
		}
		var hdr wire.BlockHeader
		hdr.Deserialize(bytes.NewReader(b))
		newTip, reorg, height, err := bc.CommitHeader(hdr)
		if err != nil {
			t.Error()
		}
		if newTip && i+6 < 11 {
			t.Error("Incorrectly set new tip when inserting header")
		}
		if !newTip && i+6 >= 11 {
			t.Error("Failed to set new tip when inserting header")
		}
		if reorg != nil && i+6 != 11 {
			t.Error("Incorrectly set reorg when inserting header")
		}
		if reorg == nil && i+6 == 11 {
			t.Error("Failed to return reorg when inserting a header that caused a reorg")
		}
		if height != uint32(i+6) {
			t.Error("Returned incorrect height when inserting header")
		}
		headers = append(headers, hdr)
	}
	best, err := bc.db.GetBestHeader()
	if err != nil {
		t.Error(err)
	}
	for i := len(headers) - 1; i >= 0; i-- {
		putHash := headers[i].BlockHash()
		retHash := best.header.BlockHash()
		if !putHash.IsEqual(&retHash) {
			t.Error("Header put failed")
		}
		best, err = bc.db.GetPreviousHeader(best.header)
	}
	os.RemoveAll("headers.bin")
}

func TestBlockchain_GetLastGoodHeader(t *testing.T) {
	bc, err := NewBlockchain("", &chaincfg.RegressionNetParams)
	if err != nil {
		t.Error(err)
	}
	var hdr wire.BlockHeader
	for _, c := range chain {
		b, err := hex.DecodeString(c)
		if err != nil {
			t.Error(err)
		}
		hdr.Deserialize(bytes.NewReader(b))
		bc.CommitHeader(hdr)
	}
	prevBest := StoredHeader{header: hdr, height: 10}
	for i := 0; i < len(fork)-1; i++ {
		b, err := hex.DecodeString(fork[i])
		if err != nil {
			t.Error(err)
		}
		hdr.Deserialize(bytes.NewReader(b))
		bc.CommitHeader(hdr)
	}
	currentBest := StoredHeader{header: hdr, height: 11}

	last, err := bc.GetLastGoodHeader(currentBest, prevBest)
	if err != nil {
		t.Error(err)
	}
	if last.height != 5 {
		t.Error("Incorrect reorg height")
	}
	os.RemoveAll("headers.bin")
}

func TestBlockchain_CheckHeader(t *testing.T) {
	params := &chaincfg.RegressionNetParams
	bc, err := NewBlockchain("", params)
	if err != nil {
		t.Error(err)
	}

	// Test valid header
	header0, err := hex.DecodeString(chain[0])
	if err != nil {
		t.Error(err)
	}
	var buf bytes.Buffer
	buf.Write(header0)
	hdr0 := wire.BlockHeader{}
	hdr0.Deserialize(&buf)

	header1, err := hex.DecodeString(chain[1])
	if err != nil {
		t.Error(err)
	}
	buf.Write(header1)
	hdr1 := wire.BlockHeader{}
	hdr1.Deserialize(&buf)
	sh := StoredHeader{
		header:    hdr0,
		height:    0,
		totalWork: big.NewInt(0),
	}
	if !bc.CheckHeader(hdr1, sh) {
		t.Error("Check header incorrectly returned false")
	}

	// Test header doesn't link
	header2, err := hex.DecodeString(chain[2])
	if err != nil {
		t.Error(err)
	}
	buf.Write(header2)
	hdr2 := wire.BlockHeader{}
	hdr2.Deserialize(&buf)
	if bc.CheckHeader(hdr2, sh) {
		t.Error("Check header missed headers that don't link")
	}
	// Test invalid difficulty
	params.ReduceMinDifficulty = false
	invalidDiffHdr := hdr1
	invalidDiffHdr.Bits = 0
	if bc.CheckHeader(invalidDiffHdr, sh) {
		t.Error("Check header did not detect invalid PoW")
	}

	// Test invalid proof of work
	params.ReduceMinDifficulty = true
	invalidPoWHdr := hdr1
	invalidPoWHdr.Nonce = 0
	if bc.CheckHeader(invalidPoWHdr, sh) {
		t.Error("Check header did not detect invalid PoW")
	}

	os.RemoveAll("headers.bin")
}

func TestBlockchain_GetNPrevBlockHashes(t *testing.T) {
	bc, err := NewBlockchain("", &chaincfg.RegressionNetParams)
	if err != nil {
		t.Error(err)
	}
	var headers = []wire.BlockHeader{regtestCheckpoint}
	for i, c := range chain {
		b, err := hex.DecodeString(c)
		if err != nil {
			t.Error(err)
		}
		var hdr wire.BlockHeader
		hdr.Deserialize(bytes.NewReader(b))
		newTip, reorg, height, err := bc.CommitHeader(hdr)
		if err != nil {
			t.Error()
		}
		if !newTip {
			t.Error("Failed to set new tip when inserting header")
		}
		if reorg != nil {
			t.Error("Incorrectly set reorg when inserting header")
		}
		if height != uint32(i+1) {
			t.Error("Returned incorrect height when inserting header")
		}
		headers = append(headers, hdr)
	}

	nHashes := bc.GetNPrevBlockHashes(5)
	for i := 0; i < 5; i++ {
		h := headers[(len(headers)-1)-i].BlockHash()
		if !nHashes[i].IsEqual(&h) {
			t.Error("GetNPrevBlockHashes returned invalid hashes")
		}
	}
	os.RemoveAll("headers.bin")
}

func TestBlockchain_checkProofOfWork(t *testing.T) {
	// Test valid
	header0, err := hex.DecodeString(chain[0])
	if err != nil {
		t.Error(err)
	}
	var buf bytes.Buffer
	buf.Write(header0)
	hdr0 := wire.BlockHeader{}
	hdr0.Deserialize(&buf)
	if !checkProofOfWork(hdr0, &chaincfg.RegressionNetParams) {
		t.Error("checkProofOfWork failed")
	}

	// Test negative target
	neg := hdr0
	neg.Bits = 1000000000
	if checkProofOfWork(neg, &chaincfg.RegressionNetParams) {
		t.Error("checkProofOfWork failed to negative target")
	}

	// Test too high diff
	params := chaincfg.RegressionNetParams
	params.PowLimit = big.NewInt(0)
	if checkProofOfWork(hdr0, &params) {
		t.Error("checkProofOfWork failed to detect above max PoW")
	}

	// Test to low work
	badHeader := "1" + chain[0][1:]
	header0, err = hex.DecodeString(badHeader)
	if err != nil {
		t.Error(err)
	}
	badHdr := wire.BlockHeader{}
	buf.Write(header0)
	badHdr.Deserialize(&buf)
	if checkProofOfWork(badHdr, &chaincfg.RegressionNetParams) {
		t.Error("checkProofOfWork failed to detect insuffient work")
	}
}

func TestBlockchain_SetChainState(t *testing.T) {
	bc, err := NewBlockchain("", &chaincfg.RegressionNetParams)
	if err != nil {
		t.Error(err)
	}
	bc.SetChainState(WAITING)
	if bc.ChainState() != WAITING {
		t.Error("Failed to set chainstate correctly")
	}
	os.RemoveAll("headers.bin")
}

func TestBlockchain_calcDiffAdjust(t *testing.T) {

	// Test calculation of next difficulty target with no constraints applying
	start := wire.BlockHeader{}
	end := wire.BlockHeader{}
	start.Timestamp = time.Unix(1261130161, 0) // Block #30240
	end.Timestamp = time.Unix(1262152739, 0)   // Block #32255
	end.Bits = 0x1d00ffff
	if calcDiffAdjust(start, end, &chaincfg.RegressionNetParams) != 0x1d00d86a {
		t.Error("callDiffAdjust returned incorrect difficulty")
	}

	// Test the constraint on the upper bound for next work
	start = wire.BlockHeader{}
	end = wire.BlockHeader{}
	start.Timestamp = time.Unix(1279008237, 0) // Block #0
	end.Timestamp = time.Unix(1279297671, 0)   // Block #2015
	end.Bits = 0x1c05a3f4
	if calcDiffAdjust(start, end, &chaincfg.RegressionNetParams) != 0x1c0168fd {
		t.Error("callDiffAdjust returned incorrect difficulty")
	}

	// Test the constraint on the lower bound for actual time taken
	start = wire.BlockHeader{}
	end = wire.BlockHeader{}
	start.Timestamp = time.Unix(1279008237, 0) // Block #66528
	end.Timestamp = time.Unix(1279297671, 0)   // Block #68543
	end.Bits = 0x1c05a3f4
	if calcDiffAdjust(start, end, &chaincfg.RegressionNetParams) != 0x1c0168fd {
		t.Error("callDiffAdjust returned incorrect difficulty")
	}

	// Test the constraint on the upper bound for actual time taken
	start = wire.BlockHeader{}
	end = wire.BlockHeader{}
	start.Timestamp = time.Unix(1263163443, 0) // NOTE: Not an actual block time
	end.Timestamp = time.Unix(1269211443, 0)   // Block #46367
	end.Bits = 0x1c387f6f
	if calcDiffAdjust(start, end, &chaincfg.RegressionNetParams) != 0x1d00e1fd {
		t.Error("callDiffAdjust returned incorrect difficulty")
	}
}

func TestBlockchain_GetBlockLocatorHashes(t *testing.T) {
	bc, err := NewBlockchain("", &chaincfg.RegressionNetParams)
	if err != nil {
		t.Error(err)
	}
	var headers = []wire.BlockHeader{regtestCheckpoint}
	for i, c := range chain {
		b, err := hex.DecodeString(c)
		if err != nil {
			t.Error(err)
		}
		var hdr wire.BlockHeader
		hdr.Deserialize(bytes.NewReader(b))
		bc.CommitHeader(hdr)
		if i < 5 {
			headers = append(headers, hdr)
		}
	}

	for _, c := range fork {
		b, err := hex.DecodeString(c)
		if err != nil {
			t.Error(err)
		}
		var hdr wire.BlockHeader
		hdr.Deserialize(bytes.NewReader(b))
		bc.CommitHeader(hdr)
		headers = append(headers, hdr)
	}

	nHashes := bc.GetBlockLocatorHashes()
	for i := 0; i < 10; i++ {
		h := headers[(len(headers)-1)-i].BlockHash()
		if !nHashes[i].IsEqual(&h) {
			t.Error("GetBlockLocatorHashes returned invalid hashes")
		}
	}
	if nHashes[10].String() != "13ae8e4687ffe9daa1d9d42d1100b398bc328e2d971af1f4464af37f412d3d7c" {
		t.Error("Error calculating locator hashes after step increase")
	}
	os.RemoveAll("headers.bin")
}

func TestBlockchain_GetEpoch(t *testing.T) {
	bc, err := NewBlockchain("", &chaincfg.RegressionNetParams)
	if err != nil {
		t.Error(err)
	}
	err = createBlockChain(bc)
	if err != nil {
		t.Error(err)
	}
	epoch, err := bc.GetEpoch()
	if err != nil {
		t.Error(err)
	}
	if epoch.BlockHash().String() != "0f9188f13cb7b2c71f2a335e3a4fc328bf5beb436012afca590b1a11466e2206" {
		t.Error("Returned incorrect epoch")
	}
	os.RemoveAll("headers.bin")
}

func TestBlockchain_calcRequiredWork(t *testing.T) {
	params := &chaincfg.TestNet3Params
	bc, err := NewBlockchain("", params)
	if err != nil {
		t.Error(err)
	}
	err = createBlockChain(bc)
	if err != nil {
		t.Error(err)
	}
	best, err := bc.db.GetBestHeader()
	if err != nil {
		t.Error(err)
	}

	// Test during difficulty adjust period
	newHdr := wire.BlockHeader{}
	newHdr.PrevBlock = best.header.BlockHash()
	work, err := bc.calcRequiredWork(newHdr, 2016, best)
	if err != nil {
		t.Error(err)
	}
	if work <= best.header.Bits {
		t.Error("Returned in correct bits")
	}
	newHdr.Bits = work
	sh := StoredHeader{
		header:    newHdr,
		height:    2016,
		totalWork: blockchain.CompactToBig(work),
	}
	bc.db.Put(sh, true)

	// Test during normal adjustment
	params.ReduceMinDifficulty = false
	newHdr1 := wire.BlockHeader{}
	newHdr1.PrevBlock = newHdr.BlockHash()
	work1, err := bc.calcRequiredWork(newHdr1, 2017, sh)
	if err != nil {
		t.Error(err)
	}
	if work1 != work {
		t.Error("Returned in correct bits")
	}
	newHdr1.Bits = work1
	sh = StoredHeader{
		header:    newHdr1,
		height:    2017,
		totalWork: blockchain.CompactToBig(work1),
	}
	bc.db.Put(sh, true)

	// Test with reduced difficult flag
	params.ReduceMinDifficulty = true
	newHdr2 := wire.BlockHeader{}
	newHdr2.PrevBlock = newHdr1.BlockHash()
	work2, err := bc.calcRequiredWork(newHdr2, 2018, sh)
	if err != nil {
		t.Error(err)
	}
	if work2 != work1 {
		t.Error("Returned in correct bits")
	}
	newHdr2.Bits = work2
	sh = StoredHeader{
		header:    newHdr2,
		height:    2018,
		totalWork: blockchain.CompactToBig(work2),
	}
	bc.db.Put(sh, true)

	// Test testnet exemption
	newHdr3 := wire.BlockHeader{}
	newHdr3.PrevBlock = newHdr2.BlockHash()
	newHdr3.Timestamp = newHdr2.Timestamp.Add(time.Minute * 21)
	work3, err := bc.calcRequiredWork(newHdr3, 2019, sh)
	if err != nil {
		t.Error(err)
	}
	if work3 != params.PowLimitBits {
		t.Error("Returned in correct bits")
	}
	newHdr3.Bits = work3
	sh = StoredHeader{
		header:    newHdr3,
		height:    2019,
		totalWork: blockchain.CompactToBig(work3),
	}
	bc.db.Put(sh, true)

	// Test multiple special difficulty blocks in a row
	params.ReduceMinDifficulty = true
	newHdr4 := wire.BlockHeader{}
	newHdr4.PrevBlock = newHdr3.BlockHash()
	work4, err := bc.calcRequiredWork(newHdr4, 2020, sh)
	if err != nil {
		t.Error(err)
	}
	if work4 != work2 {
		t.Error("Returned in correct bits")
	}
	os.RemoveAll("headers.bin")
}
