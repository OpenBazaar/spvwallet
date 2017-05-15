package spvwallet

import (
	"bytes"
	"crypto/rand"
	"encoding/hex"
	"github.com/btcsuite/btcd/chaincfg"
	"github.com/btcsuite/btcd/chaincfg/chainhash"
	"github.com/btcsuite/btcd/txscript"
	"github.com/btcsuite/btcd/wire"
	"github.com/btcsuite/btcutil/hdkeychain"
	"testing"
	"time"
)

func createTxStore() (*TxStore, error) {
	mockDb := MockDatastore{
		&mockKeyStore{make(map[string]*keyStoreEntry)},
		&mockUtxoStore{make(map[string]*Utxo)},
		&mockStxoStore{make(map[string]*Stxo)},
		&mockTxnStore{make(map[string]*txnStoreEntry)},
		&mockWatchedScriptsStore{make(map[string][]byte)},
	}
	seed := make([]byte, 32)
	rand.Read(seed)
	key, _ := hdkeychain.NewMaster(seed, &chaincfg.TestNet3Params)
	km, _ := NewKeyManager(mockDb.Keys(), &chaincfg.TestNet3Params, key)
	return NewTxStore(&chaincfg.TestNet3Params, &mockDb, km)
}

func TestNewTxStore(t *testing.T) {
	ts, err := createTxStore()
	if err != nil {
		t.Error(err)
	}
	if len(ts.adrs) != LOOKAHEADWINDOW*2 {
		t.Error("Failed to populate addresses for new TxStore")
	}
}

func TestTxStore_PopulateAdrs(t *testing.T) {
	txStore, err := createTxStore()
	if err != nil {
		t.Error(err)
	}
	err = txStore.PopulateAdrs()
	if err != nil {
		t.Error(err)
	}
	if len(txStore.adrs) != LOOKAHEADWINDOW*2 {
		t.Error("Failed to load addresses into memory")
	}
	b, err := hex.DecodeString("a91446cc55cee35873e0ebe0a90f66f942919b84d63e87")
	if err != nil {
		t.Error(err)
	}
	err = txStore.WatchedScripts().Put(b)
	if err != nil {
		t.Error(err)
	}
	err = txStore.PopulateAdrs()
	if err != nil {
		t.Error(err)
	}
	if len(txStore.watchedScripts) != 1 {
		t.Error("Failed to load watched scripts into memory")
	}
	tx1Hex := "0100000001f0c1a0d39f0f1357fcead5897f1eed424d9835d30d2543f3d804138ba825939b010000006b483045022100ed5c193377e4fb7d8df067c18e4982f55f2443cd9b41548347f646448cc5ad9f02202ad6ad5041246a23868bc52675c4c1a4018e1cfd180dcd63897fb9040df14d85012102e2606d87535c7b15855a854c09225ba025230f8b79332a6d1d06b39cd711f821ffffffff0264f3cc03000000001976a9148f83a59ebdf80b8cc965a28da3a825c126a4cefb88ac204e0000000000001976a9140706d0505002aa3ef07a822b9c143b0047b07bdf88ac00000000"
	tx1Bytes, err := hex.DecodeString(tx1Hex)
	r := bytes.NewReader(tx1Bytes)
	tx := wire.NewMsgTx(1)
	tx.BtcDecode(r, 1)

	err = txStore.Txns().Put(tx, 100000, 0, time.Now(), false)
	err = txStore.PopulateAdrs()
	if err != nil {
		t.Error(err)
	}
	if len(txStore.txids) != 1 {
		t.Error("Failed to load txids into memory")
	}

}

func TestTxStore_GimmeFilter(t *testing.T) {
	txStore, err := createTxStore()
	if err != nil {
		t.Error(err)
	}
	b, err := hex.DecodeString("a91446cc55cee35873e0ebe0a90f66f942919b84d63e87")
	if err != nil {
		t.Error(err)
	}
	err = txStore.WatchedScripts().Put(b)
	if err != nil {
		t.Error(err)
	}
	op := wire.NewOutPoint(maxHash, 0)
	err = txStore.Utxos().Put(Utxo{Op: *op})
	if err != nil {
		t.Error(err)
	}
	op2 := wire.NewOutPoint(maxHash, 1)
	err = txStore.Stxos().Put(Stxo{Utxo: Utxo{Op: *op2}})
	if err != nil {
		t.Error(err)
	}
	err = txStore.PopulateAdrs()
	if err != nil {
		t.Error(err)
	}
	filter, err := txStore.GimmeFilter()
	if err != nil {
		t.Error(err)
	}
	for _, addr := range txStore.adrs {
		if !filter.Matches(addr.ScriptAddress()) {
			t.Error("Filter does not match address")
		}
	}
	for _, script := range txStore.watchedScripts {
		_, addrs, _, err := txscript.ExtractPkScriptAddrs(script, txStore.params)
		if err != nil {
			t.Error(err)
		}
		if !filter.Matches(addrs[0].ScriptAddress()) {
			t.Error("Filter does not match watched script")
		}
	}
	if !filter.MatchesOutPoint(op) {
		t.Error("Failed to match utxo")
	}
	if !filter.MatchesOutPoint(op2) {
		t.Error("Failed to match stxo")
	}
}

func TestTxStore_CheckDoubleSpends(t *testing.T) {
	txStore, err := createTxStore()
	if err != nil {
		t.Error(err)
	}
	tx1Hex := "0100000001f0c1a0d39f0f1357fcead5897f1eed424d9835d30d2543f3d804138ba825939b010000006b483045022100ed5c193377e4fb7d8df067c18e4982f55f2443cd9b41548347f646448cc5ad9f02202ad6ad5041246a23868bc52675c4c1a4018e1cfd180dcd63897fb9040df14d85012102e2606d87535c7b15855a854c09225ba025230f8b79332a6d1d06b39cd711f821ffffffff0264f3cc03000000001976a9148f83a59ebdf80b8cc965a28da3a825c126a4cefb88ac204e0000000000001976a9140706d0505002aa3ef07a822b9c143b0047b07bdf88ac00000000"
	tx1Bytes, err := hex.DecodeString(tx1Hex)
	r := bytes.NewReader(tx1Bytes)
	tx1 := wire.NewMsgTx(1)
	tx1.BtcDecode(r, 1)
	txStore.Txns().Put(tx1, 100, 400000, time.Now(), false)
	doubles, err := txStore.CheckDoubleSpends(tx1)
	if err != nil {
		t.Error(err)
	}
	if len(doubles) > 0 {
		t.Error("Incorrect returned double spend")
	}
	tx2 := tx1.Copy()
	b, err := hex.DecodeString("a91446cc55cee35873e0ebe0a90f66f942919b84d63e87")
	if err != nil {
		t.Error(err)
	}
	tx2.TxOut[0].PkScript = b

	doubles, err = txStore.CheckDoubleSpends(tx2)
	if err != nil {
		t.Error(err)
	}

	if len(doubles) < 1 {
		t.Error("Failed to detect double spends")
	}
}

func TestTxStore_GetPendingInv(t *testing.T) {
	txStore, err := createTxStore()
	if err != nil {
		t.Error(err)
	}
	h1, err := chainhash.NewHashFromStr("6f7a58ad92702601fcbaac0e039943a384f5274a205c16bb8bbab54f9ea2fbad")
	op := wire.NewOutPoint(h1, 0)
	err = txStore.Utxos().Put(Utxo{Op: *op})
	if err != nil {
		t.Error(err)
	}
	h2, err := chainhash.NewHashFromStr("a0d4cbcd8d0694e1132400b5e114b31bc3e0d8a2ac26e054f78727c95485b528")
	err = txStore.Stxos().Put(Stxo{SpendTxid: *h2})
	if err != nil {
		t.Error(err)
	}
	inv, err := txStore.GetPendingInv()
	if err != nil {
		t.Error(err)
	}
	invMap := make(map[string]struct{})
	for _, i := range inv.InvList {
		invMap[i.Hash.String()] = struct{}{}
		if i.Type != wire.InvTypeTx {
			t.Error("Got invalid inventory type")
		}
	}
	if _, ok := invMap[h1.String()]; !ok {
		t.Error("Failed to return correct inventory packet")
	}
	if _, ok := invMap[h2.String()]; !ok {
		t.Error("Failed to return correct inventory packet")
	}
}

func TestTxStore_outpointsEqual(t *testing.T) {
	h1, err := chainhash.NewHashFromStr("6f7a58ad92702601fcbaac0e039943a384f5274a205c16bb8bbab54f9ea2fbad")
	if err != nil {
		t.Error(err)
	}
	op := wire.NewOutPoint(h1, 0)
	h2, err := chainhash.NewHashFromStr("a0d4cbcd8d0694e1132400b5e114b31bc3e0d8a2ac26e054f78727c95485b528")
	op2 := wire.NewOutPoint(h2, 0)
	if err != nil {
		t.Error(err)
	}
	if !outPointsEqual(*op, *op) {
		t.Error("Failed to detect equal outpoints")
	}
	if outPointsEqual(*op, *op2) {
		t.Error("Incorrectly returned equal outpoints")
	}
}

func TestTxStore_markAsDead(t *testing.T) {
	// Test marking single UTXO as dead

}
