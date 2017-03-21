package spvwallet

import (
	"bytes"
	"encoding/hex"
	"github.com/btcsuite/btcd/chaincfg"
	"github.com/btcsuite/btcd/wire"
	"os"
	"testing"
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
	for _, c := range chain {
		b, err := hex.DecodeString(c)
		if err != nil {
			t.Error(err)
		}
		var hdr wire.BlockHeader
		hdr.Deserialize(bytes.NewReader(b))
		bc.CommitHeader(hdr)
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

func TestBlockchain_GetReorgHeight(t *testing.T) {
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

	height, err := bc.GetReorgHeight(currentBest, prevBest)
	if err != nil {
		t.Error(err)
	}
	if height != 5 {
		t.Error("Incorrect reorg height")
	}
	os.RemoveAll("headers.bin")
}
