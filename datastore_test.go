package spvwallet

import (
	"bytes"
	"encoding/hex"
	"errors"
	"github.com/btcsuite/btcd/btcec"
	"github.com/btcsuite/btcd/chaincfg/chainhash"
	"github.com/btcsuite/btcd/wire"
	"sort"
	"strconv"
	"testing"
	"time"
)

type MockDatastore struct {
	keys           Keys
	utxos          Utxos
	stxos          Stxos
	txns           Txns
	watchedScripts WatchedScripts
}

func (m *MockDatastore) Keys() Keys {
	return m.keys
}

func (m *MockDatastore) Utxos() Utxos {
	return m.utxos
}

func (m *MockDatastore) Stxos() Stxos {
	return m.stxos
}

func (m *MockDatastore) Txns() Txns {
	return m.txns
}

func (m *MockDatastore) WatchedScripts() WatchedScripts {
	return m.watchedScripts
}

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

type mockUtxoStore struct {
	utxos map[string]*Utxo
}

func (m *mockUtxoStore) Put(utxo Utxo) error {
	key := utxo.Op.Hash.String() + ":" + strconv.Itoa(int(utxo.Op.Index))
	m.utxos[key] = &utxo
	return nil
}

func (m *mockUtxoStore) GetAll() ([]Utxo, error) {
	var utxos []Utxo
	for _, v := range m.utxos {
		utxos = append(utxos, *v)
	}
	return utxos, nil
}

func (m *mockUtxoStore) SetWatchOnly(utxo Utxo) error {
	key := utxo.Op.Hash.String() + ":" + strconv.Itoa(int(utxo.Op.Index))
	u, ok := m.utxos[key]
	if !ok {
		return errors.New("Not found")
	}
	u.WatchOnly = true
	return nil
}

func (m *mockUtxoStore) Delete(utxo Utxo) error {
	key := utxo.Op.Hash.String() + ":" + strconv.Itoa(int(utxo.Op.Index))
	_, ok := m.utxos[key]
	if !ok {
		return errors.New("Not found")
	}
	delete(m.utxos, key)
	return nil
}

type mockStxoStore struct {
	stxos map[string]*Stxo
}

func (m *mockStxoStore) Put(stxo Stxo) error {
	m.stxos[stxo.SpendTxid.String()] = &stxo
	return nil
}

func (m *mockStxoStore) GetAll() ([]Stxo, error) {
	var stxos []Stxo
	for _, v := range m.stxos {
		stxos = append(stxos, *v)
	}
	return stxos, nil
}

func (m *mockStxoStore) Delete(stxo Stxo) error {
	_, ok := m.stxos[stxo.SpendTxid.String()]
	if !ok {
		return errors.New("Not found")
	}
	delete(m.stxos, stxo.SpendTxid.String())
	return nil
}

type txnStoreEntry struct {
	txn       *wire.MsgTx
	value     int
	height    int
	timestamp time.Time
	watchOnly bool
}

type mockTxnStore struct {
	txns map[string]*txnStoreEntry
}

func (m *mockTxnStore) Put(txn *wire.MsgTx, value, height int, timestamp time.Time, watchOnly bool) error {
	m.txns[txn.TxHash().String()] = &txnStoreEntry{
		txn:       txn,
		value:     value,
		height:    height,
		timestamp: timestamp,
		watchOnly: watchOnly,
	}
	return nil
}

func (m *mockTxnStore) Get(txid chainhash.Hash) (*wire.MsgTx, Txn, error) {
	t, ok := m.txns[txid.String()]
	if !ok {
		return nil, Txn{}, errors.New("Not found")
	}
	var buf bytes.Buffer
	t.txn.Serialize(&buf)
	return t.txn, Txn{txid.String(), int64(t.value), int32(t.height), t.timestamp, t.watchOnly, buf.Bytes()}, nil
}

func (m *mockTxnStore) GetAll(includeWatchOnly bool) ([]Txn, error) {
	var txns []Txn
	for _, t := range m.txns {
		var buf bytes.Buffer
		t.txn.Serialize(&buf)
		txn := Txn{t.txn.TxHash().String(), int64(t.value), int32(t.height), t.timestamp, t.watchOnly, buf.Bytes()}
		txns = append(txns, txn)
	}
	return txns, nil
}

func (m *mockTxnStore) UpdateHeight(txid chainhash.Hash, height int) error {
	txn, ok := m.txns[txid.String()]
	if !ok {
		return errors.New("Not found")
	}
	txn.height = height
	return nil
}

func (m *mockTxnStore) Delete(txid *chainhash.Hash) error {
	_, ok := m.txns[txid.String()]
	if !ok {
		return errors.New("Not found")
	}
	delete(m.txns, txid.String())
	return nil
}

type mockWatchedScriptsStore struct {
	scripts map[string][]byte
}

func (m *mockWatchedScriptsStore) Put(scriptPubKey []byte) error {
	m.scripts[hex.EncodeToString(scriptPubKey)] = scriptPubKey
	return nil
}

func (m *mockWatchedScriptsStore) GetAll() ([][]byte, error) {
	var ret [][]byte
	for _, b := range m.scripts {
		ret = append(ret, b)
	}
	return ret, nil
}

func (m *mockWatchedScriptsStore) Delete(scriptPubKey []byte) error {
	enc := hex.EncodeToString(scriptPubKey)
	_, ok := m.scripts[enc]
	if !ok {
		return errors.New("Not found")
	}
	delete(m.scripts, enc)
	return nil
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
