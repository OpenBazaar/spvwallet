package spvwallet

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"github.com/btcsuite/btcd/chaincfg/chainhash"
	"github.com/btcsuite/btcd/wire"
	"testing"
)

func TestMakeMerkleParent(t *testing.T) {
	// Test same hash
	left, err := chainhash.NewHashFromStr("35be3035ce615f40af9b04124d05c64ecf14c96f37e7de02a57e4211972df04d")
	if err != nil {
		t.Error(err)
	}
	h, err := MakeMerkleParent(left, left)
	if err == nil {
		t.Error("Checking for duplicate hashes failed")
	}

	// Check left child nil
	h, err = MakeMerkleParent(nil, left)
	if err == nil {
		t.Error("Checking for nil left failed")
	}

	// Check right child nil
	h, err = MakeMerkleParent(left, nil)
	if err != nil {
		t.Error(err)
	}
	var sha [64]byte
	copy(sha[:32], left.CloneBytes()[:])
	copy(sha[32:], left.CloneBytes()[:])
	sgl := sha256.Sum256(sha[:])
	dbl := sha256.Sum256(sgl[:])
	if !bytes.Equal(dbl[:], h.CloneBytes()) {
		t.Error("Invalid hash returned when right is nil")
	}

	// Check valid hash return
	right, err := chainhash.NewHashFromStr("051b2338a496800ac09d130aee71096e13c73ccc28e83dc92d9439491d8be449")
	if err != nil {
		t.Error(err)
	}
	h, err = MakeMerkleParent(left, right)
	if err != nil {
		t.Error(err)
	}
	copy(sha[:32], left.CloneBytes()[:])
	copy(sha[32:], right.CloneBytes()[:])
	sgl = sha256.Sum256(sha[:])
	dbl = sha256.Sum256(sgl[:])
	if !bytes.Equal(dbl[:], h.CloneBytes()) {
		t.Error("Invalid hash returned")
	}
}

func TestMBolck_treeDepth(t *testing.T) {
	if treeDepth(8) != 3 {
		t.Error("treeDepth returned incorrect value")
	}
	if treeDepth(16) != 4 {
		t.Error("treeDepth returned incorrect value")
	}
	if treeDepth(64) != 6 {
		t.Error("treeDepth returned incorrect value")
	}
}

func TestMBolck_nextPowerOfTwo(t *testing.T) {
	if nextPowerOfTwo(5) != 8 {
		t.Error("treeDepth returned incorrect value")
	}
	if nextPowerOfTwo(15) != 16 {
		t.Error("treeDepth returned incorrect value")
	}
	if nextPowerOfTwo(57) != 64 {
		t.Error("treeDepth returned incorrect value")
	}
}

func TestMBlock_inDeadZone(t *testing.T) {
	// Test greater than root
	if !inDeadZone(127, 57) {
		t.Error("Failed to detect position greater than root")
	}
	// Test not in dead zone
	if inDeadZone(126, 57) {
		t.Error("Incorrectly returned in dead zone")
	}
}

func TestMBlockCheckMBlock(t *testing.T) {
	rawBlock, err := hex.DecodeString("0100000082bb869cf3a793432a66e826e05a6fc37469f8efb7421dc880670100000000007f16c5962e8bd963659c793ce370d95f093bc7e367117b3c30c1f8fdd0d9728776381b4d4c86041b554b852907000000043612262624047ee87660be1a707519a443b1c1ce3d248cbfc6c15870f6c5daa2019f5b01d4195ecbc9398fbf3c3b1fa9bb3183301d7a1fb3bd174fcfa40a2b6541ed70551dd7e841883ab8f0b16bf04176b7d1480e4f0af9f3d4c3595768d06820d2a7bc994987302e5b1ac80fc425fe25f8b63169ea78e68fbaaefa59379bbf011d")
	if err != nil {
		t.Error(err)
	}
	merkleBlock := &wire.MsgMerkleBlock{}
	r := bytes.NewReader(rawBlock)
	merkleBlock.BtcDecode(r, 70002)
	hashes, err := checkMBlock(merkleBlock)
	if err != nil {
		t.Error(err)
	}
	if len(hashes) != 1 {
		t.Error("Returned incorrect number of hashes")
	}
	if hashes[0].String() != "652b0aa4cf4f17bdb31f7a1d308331bba91f3b3cbf8f39c9cb5e19d4015b9f01" {
		t.Error("Returned incorrect hash")
	}
}
