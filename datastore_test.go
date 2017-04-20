package spvwallet

import (
	"bytes"
	"encoding/hex"
	"errors"
	"github.com/btcsuite/btcd/btcec"
	"github.com/btcsuite/btcd/chaincfg/chainhash"
	"github.com/btcsuite/btcd/wire"
	"sort"
	"testing"
)

type keyStoreEntry struct {
	scriptPubKey []byte
	path         KeyPath
	used         bool
	key          *btcec.PrivateKey
}

type mockKeyStore struct {
	keys map[string]*keyStoreEntry
}

func (m *mockKeyStore) Put(scriptPubKey []byte, keyPath KeyPath) error {
	m.keys[hex.EncodeToString(scriptPubKey)] = &keyStoreEntry{scriptPubKey, keyPath, false, nil}
	return nil
}

func (m *mockKeyStore) ImportKey(scriptPubKey []byte, key *btcec.PrivateKey) error {
	kp := KeyPath{Purpose: EXTERNAL, Index: -1}
	m.keys[hex.EncodeToString(scriptPubKey)] = &keyStoreEntry{scriptPubKey, kp, false, key}
	return nil
}

func (m *mockKeyStore) MarkKeyAsUsed(scriptPubKey []byte) error {
	key, ok := m.keys[hex.EncodeToString(scriptPubKey)]
	if !ok {
		return errors.New("key does not exist")
	}
	key.used = true
	return nil
}

func (m *mockKeyStore) GetLastKeyIndex(purpose KeyPurpose) (int, bool, error) {
	i := -1
	used := false
	for _, key := range m.keys {
		if key.path.Purpose == purpose && key.path.Index > i {
			i = key.path.Index
			used = key.used
		}
	}
	if i == -1 {
		return i, used, errors.New("No saved keys")
	}
	return i, used, nil
}

func (m *mockKeyStore) GetPathForScript(scriptPubKey []byte) (KeyPath, error) {
	key, ok := m.keys[hex.EncodeToString(scriptPubKey)]
	if !ok || key.path.Index == -1 {
		return KeyPath{}, errors.New("key does not exist")
	}
	return key.path, nil
}

func (m *mockKeyStore) GetKeyForScript(scriptPubKey []byte) (*btcec.PrivateKey, error) {
	for _, k := range m.keys {
		if k.path.Index == -1 && bytes.Equal(scriptPubKey, k.scriptPubKey) {
			return k.key, nil
		}
	}
	return nil, errors.New("Not found")
}

func (m *mockKeyStore) GetUnused(purpose KeyPurpose) ([]int, error) {
	var i []int
	for _, key := range m.keys {
		if !key.used && key.path.Purpose == purpose {
			i = append(i, key.path.Index)
		}
	}
	sort.Ints(i)
	return i, nil
}

func (m *mockKeyStore) GetAll() ([]KeyPath, error) {
	var kp []KeyPath
	for _, key := range m.keys {
		kp = append(kp, key.path)
	}
	return kp, nil
}

func (m *mockKeyStore) GetLookaheadWindows() map[KeyPurpose]int {
	internalLastUsed := -1
	externalLastUsed := -1
	for _, key := range m.keys {
		if key.path.Purpose == INTERNAL && key.used && key.path.Index > internalLastUsed {
			internalLastUsed = key.path.Index
		}
		if key.path.Purpose == EXTERNAL && key.used && key.path.Index > externalLastUsed {
			externalLastUsed = key.path.Index
		}
	}
	internalUnused := 0
	externalUnused := 0
	for _, key := range m.keys {
		if key.path.Purpose == INTERNAL && !key.used && key.path.Index > internalLastUsed {
			internalUnused++
		}
		if key.path.Purpose == EXTERNAL && !key.used && key.path.Index > externalLastUsed {
			externalUnused++
		}
	}
	mp := make(map[KeyPurpose]int)
	mp[INTERNAL] = internalUnused
	mp[EXTERNAL] = externalUnused
	return mp
}

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
