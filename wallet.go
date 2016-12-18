package spvwallet

import (
	"github.com/btcsuite/btcd/chaincfg"
	"github.com/btcsuite/btcd/chaincfg/chainhash"
	hd "github.com/btcsuite/btcutil/hdkeychain"
	"github.com/op/go-logging"
	b39 "github.com/tyler-smith/go-bip39"
	"sync"
)

type SPVWallet struct {
	params *chaincfg.Params

	masterPrivateKey *hd.ExtendedKey
	masterPublicKey  *hd.ExtendedKey

	maxFee      uint64
	priorityFee uint64
	normalFee   uint64
	economicFee uint64
	feeAPI      string

	repoPath string

	blockchain  *Blockchain
	txstore     *TxStore
	PeerManager *PeerManager

	config *Config
}

var log = logging.MustGetLogger("bitcoin")

const WALLET_VERSION = "0.1.0"

func NewSPVWallet(mnemonic string, params *chaincfg.Params, maxFee uint64, lowFee uint64, mediumFee uint64, highFee uint64, feeApi,
	repoPath string, db Datastore, userAgent string, trustedPeer string, logger logging.LeveledBackend) (*SPVWallet, error) {

	log.SetBackend(logger)

	seed := b39.NewSeed(mnemonic, "")

	mPrivKey, err := hd.NewMaster(seed, params)
	if err != nil {
		return nil, err
	}
	mPubKey, err := mPrivKey.Neuter()
	if err != nil {
		return nil, err
	}

	w := &SPVWallet{
		masterPrivateKey: mPrivKey,
		masterPublicKey:  mPubKey,
		params:           params,
		maxFee:           maxFee,
		priorityFee:      highFee,
		normalFee:        mediumFee,
		economicFee:      lowFee,
		feeAPI:           feeApi,
	}

	w.txstore, err = NewTxStore(w.params, db, w.masterPrivateKey)
	if err != nil {
		return nil, err
	}
	w.blockchain, err = NewBlockchain(w.repoPath, w.params)
	if err != nil {
		return nil, err
	}
	e := &Eight333{
		Blockchain: w.blockchain,
		TxStore:    w.txstore,
		blockQueue: make(chan HashAndHeight, 32),
		toDownload: make(map[*chainhash.Hash]int32),
		mutex:      new(sync.Mutex),
	}
	w.config = &Config{
		UserAgentName:    userAgent,
		UserAgentVersion: WALLET_VERSION,
		Params:           w.params,
		AddressCacheDir:  repoPath,
		GetFilter:        w.txstore.GimmeFilter,
		Eight333:         e,
	}
	w.PeerManager, err = NewPeerManager(w.config)
	if err != nil {
		return nil, err
	}

	return w, nil
}

func (w *SPVWallet) Start() {
	// If this is a new wallet or restoring from seed. Set the db height to the
	// height of the checkpoint block.
	tipHeight, _ := w.txstore.GetDBSyncHeight()
	if tipHeight == 0 {
		if w.params.Name == chaincfg.MainNetParams.Name {
			w.txstore.SetDBSyncHeight(MAINNET_CHECKPOINT_HEIGHT)
		} else if w.params.Name == chaincfg.TestNet3Params.Name {
			w.txstore.SetDBSyncHeight(TESTNET3_CHECKPOINT_HEIGHT)
		}
	}
	w.PeerManager.Start()
}

//////////////////////////////////////////////////////////////////////////////////////////////////////////////////
//
// API
//
//////////////

/*
func (w *SPVWallet) CurrencyCode() string {
	return "btc"
}

func (w *SPVWallet) MasterPrivateKey() *hd.ExtendedKey {
	return w.masterPrivateKey
}

func (w *SPVWallet) MasterPublicKey() *hd.ExtendedKey {
	return w.masterPublicKey
}

func (w *SPVWallet) CurrentAddress(purpose KeyPurpose) btc.Address {
	key := w.txstore.GetCurrentKey(purpose)
	addr, _ := key.Address(w.params)
	return btc.Address(addr)
}

func (w *SPVWallet) HasKey(addr btc.Address) bool {
	script, err := txscript.PayToAddrScript(addr)
	if err != nil {
		return false
	}
	_, err = w.txstore.GetKeyForScript(script)
	if err != nil {
		return false
	}
	return true
}

func (w *SPVWallet) Balance() (confirmed, unconfirmed int64) {
	utxos, _ := w.txstore.Utxos().GetAll()
	stxos, _ := w.txstore.Stxos().GetAll()
	for _, utxo := range utxos {
		if !utxo.Freeze {
			if utxo.AtHeight > 0 {
				confirmed += utxo.Value
			} else {
				if w.checkIfStxoIsConfirmed(utxo, stxos) {
					confirmed += utxo.Value
				} else {
					unconfirmed += utxo.Value
				}
			}
		}
	}
	return confirmed, unconfirmed
}

func (w *SPVWallet) checkIfStxoIsConfirmed(utxo Utxo, stxos []Stxo) bool {
	for _, stxo := range stxos {
		if stxo.SpendTxid.IsEqual(&utxo.Op.Hash) {
			if stxo.Utxo.AtHeight > 0 {
				return true
			} else {
				return w.checkIfStxoIsConfirmed(stxo.Utxo, stxos)
			}
		}
	}
	return false
}

func (w *SPVWallet) Params() *chaincfg.Params {
	return w.params
}

func (w *SPVWallet) AddTransactionListener(callback func(TransactionCallback)) {
	w.txstore.listeners = append(w.txstore.listeners, callback)
}

func (w *SPVWallet) ChainTip() uint32 {
	height, _ := w.txstore.GetDBSyncHeight()
	return uint32(height)
}

func (w *SPVWallet) AddWatchedScript(script []byte) error {
	err := w.txstore.WatchedScripts().Put(script)
	w.txstore.PopulateAdrs()
	// TODO: update filter and resend to all peers
	return err
}

func (w *SPVWallet) GenerateMultisigScript(keys []hd.ExtendedKey, threshold int) (addr btc.Address, redeemScript []byte, err error) {
	var addrPubKeys []*btc.AddressPubKey
	for _, key := range keys {
		ecKey, err := key.ECPubKey()
		if err != nil {
			return nil, nil, err
		}
		k, err := btc.NewAddressPubKey(ecKey.SerializeCompressed(), w.params)
		if err != nil {
			return nil, nil, err
		}
		addrPubKeys = append(addrPubKeys, k)
	}
	redeemScript, err = txscript.MultiSigScript(addrPubKeys, threshold)
	if err != nil {
		return nil, nil, err
	}
	addr, err = btc.NewAddressScriptHash(redeemScript, w.params)
	if err != nil {
		return nil, nil, err
	}
	return addr, redeemScript, nil
}

func (w *SPVWallet) Close() {
	log.Info("Disconnecting from peers and shutting down")
	w.PeerManager.Stop()
	w.blockchain.Close()
}

func (w *SPVWallet) ReSyncBlockchain(fromHeight int32) {
	w.Close()
	if w.params.Name == chaincfg.MainNetParams.Name && fromHeight < MAINNET_CHECKPOINT_HEIGHT {
		fromHeight = MAINNET_CHECKPOINT_HEIGHT
	} else if w.params.Name == chaincfg.TestNet3Params.Name && fromHeight < TESTNET3_CHECKPOINT_HEIGHT {
		fromHeight = TESTNET3_CHECKPOINT_HEIGHT
	}
	w.txstore.SetDBSyncHeight(fromHeight)
	go w.Start()
}
*/
