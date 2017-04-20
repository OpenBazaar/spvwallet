package spvwallet

import (
	"github.com/btcsuite/btcd/chaincfg/chainhash"
	"github.com/btcsuite/btcd/wire"
	"testing"
)

func TestUtxo_IsEqual(t *testing.T) {
	h, err := chainhash.NewHashFromStr("16bed6368b8b1542cd6eb87f5bc20dc830b41a2258dde40438a75fa701d24e9a")
	if err != nil {
		t.Error(err)
	}
	u := &Utxo{
		Op:           *wire.NewOutPoint(h, 0),
		ScriptPubkey: make([]byte, 32),
		AtHeight:     400000,
		Value:        1000000,
	}
	if !u.IsEqual(u) {
		t.Error("Failed to return utxos as equal")
	}
	testUtxo := *u
	testUtxo.Op.Index = 3
	if u.IsEqual(&testUtxo) {
		t.Error("Failed to return utxos as not equal")
	}
	testUtxo = *u
	testUtxo.AtHeight = 1
	if u.IsEqual(&testUtxo) {
		t.Error("Failed to return utxos as not equal")
	}
	testUtxo = *u
	testUtxo.Value = 4
	if u.IsEqual(&testUtxo) {
		t.Error("Failed to return utxos as not equal")
	}
	testUtxo = *u
	ch2, err := chainhash.NewHashFromStr("1f64249abbf2fcc83fc060a64f69a91391e9f5d98c5d3135fe9716838283aa4c")
	if err != nil {
		t.Error(err)
	}
	testUtxo.Op.Hash = *ch2
	if u.IsEqual(&testUtxo) {
		t.Error("Failed to return utxos as not equal")
	}
	testUtxo = *u
	testUtxo.ScriptPubkey = make([]byte, 4)
	if u.IsEqual(&testUtxo) {
		t.Error("Failed to return utxos as not equal")
	}
	if u.IsEqual(nil) {
		t.Error("Failed to return utxos as not equal")
	}
}

func TestStxo_IsEqual(t *testing.T) {
	h, err := chainhash.NewHashFromStr("16bed6368b8b1542cd6eb87f5bc20dc830b41a2258dde40438a75fa701d24e9a")
	if err != nil {
		t.Error(err)
	}
	u := &Utxo{
		Op:           *wire.NewOutPoint(h, 0),
		ScriptPubkey: make([]byte, 32),
		AtHeight:     400000,
		Value:        1000000,
	}
	h2, err := chainhash.NewHashFromStr("1f64249abbf2fcc83fc060a64f69a91391e9f5d98c5d3135fe9716838283aa4c")
	s := &Stxo{
		Utxo:        *u,
		SpendHeight: 400001,
		SpendTxid:   *h2,
	}
	if !s.IsEqual(s) {
		t.Error("Failed to return stxos as equal")
	}

	testStxo := *s
	testStxo.SpendHeight = 5
	if s.IsEqual(&testStxo) {
		t.Error("Failed to return stxos as not equal")
	}
	h3, err := chainhash.NewHashFromStr("3c5cea030a432ba9c8cf138a93f7b2e5b28263ea416894ee0bdf91bc31bb04f2")
	testStxo = *s
	testStxo.SpendTxid = *h3
	if s.IsEqual(&testStxo) {
		t.Error("Failed to return stxos as not equal")
	}
	if s.IsEqual(nil) {
		t.Error("Failed to return stxos as not equal")
	}
	testStxo = *s
	testStxo.Utxo.AtHeight = 7
	if s.IsEqual(&testStxo) {
		t.Error("Failed to return stxos as not equal")
	}
}
