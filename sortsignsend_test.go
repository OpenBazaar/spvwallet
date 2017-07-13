package spvwallet

import (
	"bytes"
	"github.com/btcsuite/btcd/chaincfg"
	"github.com/btcsuite/btcd/chaincfg/chainhash"
	"github.com/btcsuite/btcd/wire"
	"os"
	"testing"
)

func MockWallet() *SPVWallet {
	txstore, _ := createTxStore()

	peerCfg := &PeerManagerConfig{
		UserAgentVersion: WALLET_VERSION,
		Params:           &chaincfg.TestNet3Params,
		GetFilter:        txstore.GimmeFilter,
	}

	bc, _ := NewBlockchain("", MockCreationTime, &chaincfg.TestNet3Params)
	createBlockChain(bc)

	peerManager, _ := NewPeerManager(peerCfg)
	return &SPVWallet{txstore: txstore, peerManager: peerManager, blockchain: bc, keyManager: txstore.keyManager, params: &chaincfg.TestNet3Params}
}

func Test_gatherCoins(t *testing.T) {
	wallet := MockWallet()
	h1, err := chainhash.NewHashFromStr("6f7a58ad92702601fcbaac0e039943a384f5274a205c16bb8bbab54f9ea2fbad")
	if err != nil {
		t.Error(err)
	}
	key1, err := wallet.keyManager.GetFreshKey(EXTERNAL)
	if err != nil {
		t.Error(err)
	}
	addr1, err := key1.Address(&chaincfg.TestNet3Params)
	if err != nil {
		t.Error(err)
	}
	script1, err := wallet.AddressToScript(addr1)
	if err != nil {
		t.Error(err)
	}
	op := wire.NewOutPoint(h1, 0)
	err = wallet.txstore.Utxos().Put(Utxo{Op: *op, ScriptPubkey: script1, AtHeight: 5, Value: 10000})
	if err != nil {
		t.Error(err)
	}
	coinmap := wallet.gatherCoins()
	for coin, key := range coinmap {
		if !bytes.Equal(coin.PkScript(), script1) {
			t.Error("Pubkey script in coin is incorrect")
		}
		if coin.Index() != 0 {
			t.Error("Returned incorrect index")
		}
		if !coin.Hash().IsEqual(h1) {
			t.Error("Returned incorrect hash")
		}
		height, _ := wallet.blockchain.db.Height()
		if coin.NumConfs() != int64(height-5) {
			t.Error("Returned incorrect number of confirmations")
		}
		if coin.Value() != 10000 {
			t.Error("Returned incorrect coin value")
		}
		addr2, err := key.Address(&chaincfg.TestNet3Params)
		if err != nil {
			t.Error(err)
		}
		if addr2.EncodeAddress() != addr1.EncodeAddress() {
			t.Error("Returned incorrect key")
		}
	}
	os.Remove("headers.bin")
}
