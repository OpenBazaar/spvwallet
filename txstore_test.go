package spvwallet

import (
	"bytes"
	"crypto/rand"
	"encoding/hex"
	"github.com/OpenBazaar/wallet-interface"
	"github.com/btcsuite/btcd/chaincfg"
	"github.com/btcsuite/btcd/chaincfg/chainhash"
	"github.com/btcsuite/btcd/txscript"
	"github.com/btcsuite/btcd/wire"
	"github.com/btcsuite/btcutil/hdkeychain"
	"testing"
	"time"
)

var zeroHash chainhash.Hash

func createTxStore() (*TxStore, error) {
	mockDb := MockDatastore{
		&mockKeyStore{make(map[string]*keyStoreEntry)},
		&mockUtxoStore{make(map[string]*wallet.Utxo)},
		&mockStxoStore{make(map[string]*wallet.Stxo)},
		&mockTxnStore{make(map[string]*wallet.Txn)},
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
	tx.BtcDecode(r, 1, wire.WitnessEncoding)

	err = txStore.Txns().Put(tx1Bytes, tx.TxHash().String(), "100000", 0, time.Now(), false)
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
	op := wire.NewOutPoint(&zeroHash, 0)
	err = txStore.Utxos().Put(wallet.Utxo{Op: *op})
	if err != nil {
		t.Error(err)
	}
	op2 := wire.NewOutPoint(&zeroHash, 1)
	err = txStore.Stxos().Put(wallet.Stxo{Utxo: wallet.Utxo{Op: *op2}})
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
	tx1.BtcDecode(r, 1, wire.WitnessEncoding)
	txStore.Txns().Put(tx1Bytes, tx1.TxHash().String(), "100", 400000, time.Now(), false)
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
	err = txStore.Utxos().Put(wallet.Utxo{Op: *op})
	if err != nil {
		t.Error(err)
	}
	h2, err := chainhash.NewHashFromStr("a0d4cbcd8d0694e1132400b5e114b31bc3e0d8a2ac26e054f78727c95485b528")
	err = txStore.Stxos().Put(wallet.Stxo{SpendTxid: *h2})
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
	txStore, err := createTxStore()
	if err != nil {
		t.Error(err)
	}
	tx1Hex := "0100000001f0c1a0d39f0f1357fcead5897f1eed424d9835d30d2543f3d804138ba825939b010000006b483045022100ed5c193377e4fb7d8df067c18e4982f55f2443cd9b41548347f646448cc5ad9f02202ad6ad5041246a23868bc52675c4c1a4018e1cfd180dcd63897fb9040df14d85012102e2606d87535c7b15855a854c09225ba025230f8b79332a6d1d06b39cd711f821ffffffff0264f3cc03000000001976a9148f83a59ebdf80b8cc965a28da3a825c126a4cefb88ac204e0000000000001976a9140706d0505002aa3ef07a822b9c143b0047b07bdf88ac00000000"
	tx1Bytes, err := hex.DecodeString(tx1Hex)
	r := bytes.NewReader(tx1Bytes)
	tx1 := wire.NewMsgTx(1)
	tx1.BtcDecode(r, 1, wire.WitnessEncoding)
	txStore.Txns().Put(tx1Bytes, tx1.TxHash().String(), "100", 0, time.Now(), false)

	h1 := tx1.TxHash()
	op := wire.NewOutPoint(&h1, 0)
	err = txStore.Utxos().Put(wallet.Utxo{Op: *op})
	if err != nil {
		t.Error(err)
	}

	err = txStore.markAsDead(tx1.TxHash())
	if err != nil {
		t.Error(err)
	}
	checkTx, err := txStore.Txns().Get(tx1.TxHash())
	if err != nil {
		t.Error(err)
	}
	if checkTx.Height >= 0 {
		t.Error("Failed to mark tx as dead")
	}

	utxos, err := txStore.Utxos().GetAll()
	if err != nil {
		t.Error(err)
	}
	if len(utxos) > 0 {
		t.Error("Failed to delete dead utxo")
	}

	// Test marking STXO as dead
	txStore, err = createTxStore()
	if err != nil {
		t.Error(err)
	}

	txStore.Txns().Put(tx1Bytes, tx1.TxHash().String(), "100", 400000, time.Now(), false)

	tx2Hex := "01000000018dce6a1748a0b35475903ae654bb0c000fa004a8a83f16a18464de473da42b1c010000006a473044022001c7c890110c94a22bbb004b75364b03b157cb0f71a97c419a4ed80f0155649b0220257f54fbda579e0c4063f980ddd2ea9bfa591c42e759cc0cd78370bd1d24afba01210245a1619fc1feb837ed54a9dfa71d7abea445ef193fd1f9fa0d5b4141980bff11ffffffff0280a4bf070000000017a9143cb6156a7f8b5c8e72b764e00fbdfe31e77fe86187084cd600010000001976a914a4cf57fd8d825995d5fd5104675ccedd39cf924988ac00000000"
	tx2Bytes, err := hex.DecodeString(tx2Hex)
	r = bytes.NewReader(tx2Bytes)
	tx2 := wire.NewMsgTx(1)
	tx2.BtcDecode(r, 1, wire.WitnessEncoding)
	txStore.Txns().Put(tx2Bytes, tx2.TxHash().String(), "100", 0, time.Now(), false)

	op = wire.NewOutPoint(&h1, 0)
	st := wallet.Stxo{
		Utxo:        wallet.Utxo{Op: *op},
		SpendHeight: 0,
		SpendTxid:   tx2.TxHash(),
	}
	err = txStore.Stxos().Put(st)

	err = txStore.markAsDead(tx2.TxHash())
	if err != nil {
		t.Error(err)
	}

	checkTx, err = txStore.Txns().Get(tx2.TxHash())
	if err != nil {
		t.Error(err)
	}
	if checkTx.Height >= 0 {
		t.Error("Failed to mark tx as dead")
	}

	utxos, err = txStore.Utxos().GetAll()
	if err != nil {
		t.Error(err)
	}
	if len(utxos) > 1 {
		t.Error("Failed to move stxo back into utxo db")
	}

	stxos, err := txStore.Stxos().GetAll()
	if err != nil {
		t.Error(err)
	}
	if len(stxos) > 0 {
		t.Error("Failed to delete dead stxo")
	}

	// Test marking STXO dependency as dead
	txStore, err = createTxStore()
	if err != nil {
		t.Error(err)
	}

	txStore.Txns().Put(tx1Bytes, tx1.TxHash().String(), "100", 0, time.Now(), false)
	txStore.Txns().Put(tx2Bytes, tx2.TxHash().String(), "100", 0, time.Now(), false)

	op = wire.NewOutPoint(&h1, 0)
	st = wallet.Stxo{
		Utxo:        wallet.Utxo{Op: *op},
		SpendHeight: 0,
		SpendTxid:   tx2.TxHash(),
	}
	err = txStore.Stxos().Put(st)

	err = txStore.markAsDead(tx1.TxHash())
	if err != nil {
		t.Error(err)
	}

	checkTx, err = txStore.Txns().Get(tx1.TxHash())
	if err != nil {
		t.Error(err)
	}
	if checkTx.Height >= 0 {
		t.Error("Failed to mark tx as dead")
	}

	checkTx, err = txStore.Txns().Get(tx2.TxHash())
	if err != nil {
		t.Error(err)
	}
	if checkTx.Height >= 0 {
		t.Error("Failed to mark tx as dead")
	}

	utxos, err = txStore.Utxos().GetAll()
	if err != nil {
		t.Error(err)
	}
	if len(utxos) > 0 {
		t.Error("Failed to move stxo back into utxo db")
	}

	stxos, err = txStore.Stxos().GetAll()
	if err != nil {
		t.Error(err)
	}
	if len(stxos) > 0 {
		t.Error("Failed to delete dead stxo")
	}

	// Test marking STXO and UXTO change as dead
	txStore, err = createTxStore()
	if err != nil {
		t.Error(err)
	}

	txStore.Txns().Put(tx1Bytes, tx1.TxHash().String(), "100", 0, time.Now(), false)
	txStore.Txns().Put(tx2Bytes, tx2.TxHash().String(), "100", 0, time.Now(), false)

	op = wire.NewOutPoint(&h1, 0)
	st = wallet.Stxo{
		Utxo:        wallet.Utxo{Op: *op},
		SpendHeight: 0,
		SpendTxid:   tx2.TxHash(),
	}
	err = txStore.Stxos().Put(st)

	h2 := tx2.TxHash()
	op2 := wire.NewOutPoint(&h2, 0)
	err = txStore.Utxos().Put(wallet.Utxo{Op: *op2})
	if err != nil {
		t.Error(err)
	}

	tx3Hex := "0100000001dc8910ef79c4bc690cdf3e335c0f88757ba176e00057cb63ccbae6a13205d4cf010000006a47304402203c5203c53b463ac459c93954513ffb32c7056c5f2a6c825362afba21f5d1c88202207a121f13fa0f2cfe1392d2b2a4139485cc4251058a6cccc1e3c25970104df5cd012102530f811d7da235aad895cba33e2d42d1092140d1c6e6b4d965db861f5988d64affffffff02fc2f0600000000001976a9145d9e4978b7998369cda5ce3ae79c6db25957e91d88ac29dc6a16000000001976a9145cd1285c75daa5adc4c5b979b0f96c01dd08dfec88ac00000000"
	tx3Bytes, err := hex.DecodeString(tx3Hex)
	r = bytes.NewReader(tx3Bytes)
	tx3 := wire.NewMsgTx(1)
	tx3.BtcDecode(r, 1, wire.WitnessEncoding)
	txStore.Txns().Put(tx3Bytes, tx3.TxHash().String(), "100", 0, time.Now(), false)

	op = wire.NewOutPoint(&h2, 0)
	st = wallet.Stxo{
		Utxo:        wallet.Utxo{Op: *op},
		SpendHeight: 0,
		SpendTxid:   tx3.TxHash(),
	}
	err = txStore.Stxos().Put(st)

	err = txStore.markAsDead(tx1.TxHash())
	if err != nil {
		t.Error(err)
	}

	checkTx, err = txStore.Txns().Get(tx1.TxHash())
	if err != nil {
		t.Error(err)
	}
	if checkTx.Height >= 0 {
		t.Error("Failed to mark tx as dead")
	}

	checkTx, err = txStore.Txns().Get(tx2.TxHash())
	if err != nil {
		t.Error(err)
	}
	if checkTx.Height >= 0 {
		t.Error("Failed to mark tx as dead")
	}

	checkTx, err = txStore.Txns().Get(tx3.TxHash())
	if err != nil {
		t.Error(err)
	}
	if checkTx.Height >= 0 {
		t.Error("Failed to mark tx as dead")
	}

	utxos, err = txStore.Utxos().GetAll()
	if err != nil {
		t.Error(err)
	}
	if len(utxos) > 0 {
		t.Error("Failed to move stxo back into utxo db")
	}

	stxos, err = txStore.Stxos().GetAll()
	if err != nil {
		t.Error(err)
	}
	if len(stxos) > 0 {
		t.Error("Failed to delete dead stxo")
	}
}

func TestTxStore_processReorg(t *testing.T) {
	// Receive utxo1 (tx1)
	// Spend utxo1 and create change utxo2 (tx2)
	// Spend utxo2 (tx3)
	// tx2 and tx3 get reorged away
	// Should only have utxo1 left and two dead txs

	txStore, err := createTxStore()
	if err != nil {
		t.Error(err)
	}

	tx1Hex := "0100000001f0c1a0d39f0f1357fcead5897f1eed424d9835d30d2543f3d804138ba825939b010000006b483045022100ed5c193377e4fb7d8df067c18e4982f55f2443cd9b41548347f646448cc5ad9f02202ad6ad5041246a23868bc52675c4c1a4018e1cfd180dcd63897fb9040df14d85012102e2606d87535c7b15855a854c09225ba025230f8b79332a6d1d06b39cd711f821ffffffff0264f3cc03000000001976a9148f83a59ebdf80b8cc965a28da3a825c126a4cefb88ac204e0000000000001976a9140706d0505002aa3ef07a822b9c143b0047b07bdf88ac00000000"
	tx1Bytes, err := hex.DecodeString(tx1Hex)
	r := bytes.NewReader(tx1Bytes)
	tx1 := wire.NewMsgTx(1)
	tx1.BtcDecode(r, 1, wire.WitnessEncoding)
	txStore.Txns().Put(tx1Bytes, tx1.TxHash().String(), "100", 400000, time.Now(), false)

	tx2Hex := "01000000018dce6a1748a0b35475903ae654bb0c000fa004a8a83f16a18464de473da42b1c010000006a473044022001c7c890110c94a22bbb004b75364b03b157cb0f71a97c419a4ed80f0155649b0220257f54fbda579e0c4063f980ddd2ea9bfa591c42e759cc0cd78370bd1d24afba01210245a1619fc1feb837ed54a9dfa71d7abea445ef193fd1f9fa0d5b4141980bff11ffffffff0280a4bf070000000017a9143cb6156a7f8b5c8e72b764e00fbdfe31e77fe86187084cd600010000001976a914a4cf57fd8d825995d5fd5104675ccedd39cf924988ac00000000"
	tx2Bytes, err := hex.DecodeString(tx2Hex)
	r = bytes.NewReader(tx2Bytes)
	tx2 := wire.NewMsgTx(1)
	tx2.BtcDecode(r, 1, wire.WitnessEncoding)
	txStore.Txns().Put(tx2Bytes, tx2.TxHash().String(), "100", 400001, time.Now(), false)

	tx3Hex := "0100000001dc8910ef79c4bc690cdf3e335c0f88757ba176e00057cb63ccbae6a13205d4cf010000006a47304402203c5203c53b463ac459c93954513ffb32c7056c5f2a6c825362afba21f5d1c88202207a121f13fa0f2cfe1392d2b2a4139485cc4251058a6cccc1e3c25970104df5cd012102530f811d7da235aad895cba33e2d42d1092140d1c6e6b4d965db861f5988d64affffffff02fc2f0600000000001976a9145d9e4978b7998369cda5ce3ae79c6db25957e91d88ac29dc6a16000000001976a9145cd1285c75daa5adc4c5b979b0f96c01dd08dfec88ac00000000"
	tx3Bytes, err := hex.DecodeString(tx3Hex)
	r = bytes.NewReader(tx3Bytes)
	tx3 := wire.NewMsgTx(1)
	tx3.BtcDecode(r, 1, wire.WitnessEncoding)
	txStore.Txns().Put(tx3Bytes, tx3.TxHash().String(), "100", 400002, time.Now(), false)

	h1 := tx1.TxHash()
	op := wire.NewOutPoint(&h1, 0)
	st := wallet.Stxo{
		Utxo:        wallet.Utxo{Op: *op},
		SpendHeight: 0,
		SpendTxid:   tx2.TxHash(),
	}
	err = txStore.Stxos().Put(st)

	h2 := tx2.TxHash()
	op2 := wire.NewOutPoint(&h2, 0)
	txStore.Utxos().Put(wallet.Utxo{Op: *op2})

	op = wire.NewOutPoint(&h2, 0)
	st = wallet.Stxo{
		Utxo:        wallet.Utxo{Op: *op},
		SpendHeight: 0,
		SpendTxid:   tx3.TxHash(),
	}
	err = txStore.Stxos().Put(st)

	err = txStore.processReorg(400000)
	if err != nil {
		t.Error(err)
	}

	utxos, err := txStore.Utxos().GetAll()
	if err != nil {
		t.Error(err)
	}
	if len(utxos) != 1 {
		t.Error("Failed to move stxo back into utxo db")
	}

	stxos, err := txStore.Stxos().GetAll()
	if err != nil {
		t.Error(err)
	}
	if len(stxos) > 0 {
		t.Error("Failed to delete dead stxo")
	}

	checkTx, err := txStore.Txns().Get(tx1.TxHash())
	if err != nil {
		t.Error(err)
	}
	if checkTx.Height != 400000 {
		t.Error("Failed to mark tx as dead")
	}

	checkTx, err = txStore.Txns().Get(tx2.TxHash())
	if err != nil {
		t.Error(err)
	}
	if checkTx.Height >= 0 {
		t.Error("Failed to mark tx as dead")
	}

	checkTx, err = txStore.Txns().Get(tx3.TxHash())
	if err != nil {
		t.Error(err)
	}
	if checkTx.Height >= 0 {
		t.Error("Failed to mark tx as dead")
	}

}

func TestTxStore_Ingest(t *testing.T) {
	txStore, err := createTxStore()
	if err != nil {
		t.Error(err)
	}

	tx1Hex := "0100000001f0c1a0d39f0f1357fcead5897f1eed424d9835d30d2543f3d804138ba825939b010000006b483045022100ed5c193377e4fb7d8df067c18e4982f55f2443cd9b41548347f646448cc5ad9f02202ad6ad5041246a23868bc52675c4c1a4018e1cfd180dcd63897fb9040df14d85012102e2606d87535c7b15855a854c09225ba025230f8b79332a6d1d06b39cd711f821ffffffff0264f3cc03000000001976a9148f83a59ebdf80b8cc965a28da3a825c126a4cefb88ac204e0000000000001976a9140706d0505002aa3ef07a822b9c143b0047b07bdf88ac00000000"
	tx1Bytes, err := hex.DecodeString(tx1Hex)
	r := bytes.NewReader(tx1Bytes)
	tx1 := wire.NewMsgTx(1)
	tx1.BtcDecode(r, 1, wire.WitnessEncoding)

	// Ingest no hits
	hits, err := txStore.Ingest(tx1, 0, time.Now())
	if err != nil {
		t.Error(err)
	}
	if hits != 0 {
		t.Error("Failed to correctly ingest tx")
	}

	// Ingest output hit
	key, err := txStore.keyManager.GetCurrentKey(wallet.EXTERNAL)
	if err != nil {
		t.Error(err)
	}
	addr, err := key.Address(&chaincfg.TestNet3Params)
	if err != nil {
		t.Error(err)
	}
	script, err := txscript.PayToAddrScript(addr)
	if err != nil {
		t.Error(err)
	}
	out := wire.NewTxOut(100000, script)
	tx1.AddTxOut(out)

	hits, err = txStore.Ingest(tx1, 0, time.Now())
	if err != nil {
		t.Error(err)
	}
	if hits < 1 {
		t.Error("Failed to correctly ingest tx")
	}
	txns, err := txStore.Txns().GetAll(true)
	if err != nil {
		t.Error(err)
	}
	if len(txns) != 1 {
		t.Error("Failed to record tx in database")
	}

	// Ingest duplicate
	hits, err = txStore.Ingest(tx1, 0, time.Now())
	if err != nil {
		t.Error(err)
	}
	if hits != 1 {
		t.Error("Failed to correctly ingest tx")
	}

	// Ingest double spend with height zero
	tx2Hex := "01000000017a00467fc0a1ef040bbc544a66a5d4c7badd35efe18c343cf403f63937dfd9b1000000006b483045022100a08ea162b0591d3438bdab3ef8a80c6a7ce62dd593e01b96165ea7a6d72cb5ca02202e9db6dfd216a40cf0c0a466218decaf0f5c52c00f389be3e96a32d35559e150012102257118cc606883162804ce7ee371b97a9f58ee759ed819120b9c640e0d3ca8e4ffffffff01e4ab7c000000000017a914ac66e5ca929ded3d146c77ae988886050b1a8e528700000000"
	tx2Bytes, err := hex.DecodeString(tx2Hex)
	r = bytes.NewReader(tx2Bytes)
	tx2 := wire.NewMsgTx(1)
	tx2.BtcDecode(r, 1, wire.WitnessEncoding)
	tx2.AddTxIn(tx1.TxIn[0])
	hits, err = txStore.Ingest(tx2, 0, time.Now())
	if err != nil {
		t.Error(err)
	}
	if hits != 0 {
		t.Error("Failed to correctly ingest tx")
	}

	// Ingest double spend that supersedes a previous committed tx
	hits, err = txStore.Ingest(tx2, 50, time.Now())
	if err != nil {
		t.Error(err)
	}
	checkTx, err := txStore.Txns().Get(tx1.TxHash())
	if err != nil {
		t.Error(err)
	}
	if checkTx.Height >= 0 {
		t.Error("Failed to mark tx as dead")
	}
	utxos, err := txStore.Utxos().GetAll()
	if err != nil {
		t.Error(err)
	}
	if len(utxos) > 0 {
		t.Error("Failed to move stxo back into utxo db")
	}

	// Ingest watched script hit
	tx3Hex := "010000000140f831600eac0c1741c89f61134cb65142a4d95e0d53deb313872b2c5c675a82010000006a47304402203e002a46d94e917c99ecbea7dc5744f65d9f5c78c97802c85aa424f5521f024002206c315f5ae183bb4f007190f1f9c61dbfa3c6127ac45a381956f8de3894196afd012102fe6d4e37bb5956b51b62e87e3163530f20a33a8aba13ff973e84d7061b53ca5effffffff02bc733400000000001976a914fcd6edaae418f8ba77112965d7a1e997a660893a88ac41fe1c14010000001976a9145c069b3af330230523d378824e366ab9a4a1731188ac00000000"
	tx3Bytes, err := hex.DecodeString(tx3Hex)
	r = bytes.NewReader(tx3Bytes)
	tx3 := wire.NewMsgTx(1)
	tx3.BtcDecode(r, 1, wire.WitnessEncoding)
	script, err = hex.DecodeString("a914ac66e5ca929ded3d146c77ae988886050b1a8e5287")
	if err != nil {
		t.Error(err)
	}
	txStore.WatchedScripts().Put(script)
	txStore.PopulateAdrs()
	tx3.AddTxOut(wire.NewTxOut(400000, script))

	_, err = txStore.Ingest(tx3, 0, time.Now())
	if err != nil {
		t.Error(err)
	}
	checkTx, err = txStore.Txns().Get(tx3.TxHash())
	if err != nil {
		t.Error(err)
	}
	if !checkTx.WatchOnly {
		t.Error("Incorrectly set watch only transaction")
	}

	// Test input match
	tx4Hex := "0100000001473cc2e9e542f64361d9faba24316a8d4de19a00c0c4763d944dac6d00f5f641000000006b483045022100d3cf0dcec644d7964e350c3d5e25bf63be6e04b1cb8004b84cd880836fb668cf02207919d75ee5ba66a49456430f90eebda05f6284ec8831ff9f82f101893e77f7a7012103238c6886300c26ac580f37e4df1e710e74854bd9b183dfce48c5c1ab204d99b1feffffff02de6b0300000000001976a9147acf80f880c32c742b3331dbcb1a6ed51d76fc6188ace50f0f01000000001976a914827b6495bfac61449ab852d94e0abad0eb90bca388acf22e1100"
	tx4Bytes, err := hex.DecodeString(tx4Hex)
	r = bytes.NewReader(tx4Bytes)
	tx4 := wire.NewMsgTx(1)
	tx4.BtcDecode(r, 1, wire.WitnessEncoding)
	txStore.Ingest(tx1, 0, time.Now())

	h := tx1.TxHash()
	op := wire.NewOutPoint(&h, 2)
	tx4.AddTxIn(wire.NewTxIn(op, []byte{}, [][]byte{}))

	hits, err = txStore.Ingest(tx4, 0, time.Now())
	if err != nil {
		t.Error(err)
	}
	if hits < 1 {
		t.Error("Failed to ingest matching input")
	}

	utxos, err = txStore.Utxos().GetAll()
	if err != nil {
		t.Error(err)
	}
	for _, u := range utxos {
		if outPointsEqual(*op, u.Op) {
			t.Error("Failed to move utxo to stxo")
		}
	}

	// Spend the watch only utxo
	tx5Hex := "01000000010bfec1c7b63fb4d9165a4ea763858ea651e1d71d2bd75f03dd4dadda2198cd8c01000000da00473044022022360ce1b5619dd13e2d0bf81f0913faf395779c170d399a3e8af9ac89f7bb8202203ef78b426f90bc05018593730a0a842b621556ddc20d38af5375cd35ce84387e01483045022100b14db9ef3357c07b2b09ce10d95b7da0d64644e2178d1aa01fc679ddf3d1ae6c0220109d732f4609cb37b06fa42888931080789ee37a314a370cef01f17aaac0851b0147522102632178d046673c9729d828cfee388e121f497707f810c131e0d3fc0fe0bd66d62103a0951ec7d3a9da9de171617026442fcd30f34d66100fab539853b43f508787d452aeffffffff0240420f000000000017a9142384d7645fc56018b44e98afd4cb689e5d5e0b35877d12297a0000000017a9148ce5408cfeaddb7ccb2545ded41ef478109454848700000000"
	tx5Bytes, err := hex.DecodeString(tx5Hex)
	r = bytes.NewReader(tx5Bytes)
	tx5 := wire.NewMsgTx(1)
	tx5.BtcDecode(r, 1, wire.WitnessEncoding)

	h3 := tx3.TxHash()
	op3 := wire.NewOutPoint(&h3, 2)
	tx5.AddTxIn(wire.NewTxIn(op3, []byte{}, [][]byte{}))

	hits, err = txStore.Ingest(tx5, 0, time.Now())
	if err != nil {
		t.Error(err)
	}

	utxos, err = txStore.Utxos().GetAll()
	if err != nil {
		t.Error(err)
	}
	for _, u := range utxos {
		if outPointsEqual(*op3, u.Op) {
			t.Error("Failed to move utxo to stxo")
		}
	}

	// Update stxo height
	_, err = txStore.Ingest(tx5, 1000, time.Now())
	if err != nil {
		t.Error(err)
	}
	stxos, err := txStore.Stxos().GetAll()
	if err != nil {
		t.Error(err)
	}
	found := false
	for _, s := range stxos {
		if outPointsEqual(*op3, s.Utxo.Op) {
			found = true
			if s.SpendHeight != 1000 {
				t.Error("Failed to set stxo height correctly")
			}
			break
		}
	}
	if !found {
		t.Error("Utxo failed to move to the stxo table")
	}
}
