package db

import (
	"database/sql"
	"encoding/hex"
	"github.com/OpenBazaar/wallet-interface"
	"github.com/btcsuite/btcd/chaincfg/chainhash"
	"github.com/btcsuite/btcd/wire"
	"strconv"
	"strings"
	"sync"
)

type StxoDB struct {
	db   *sql.DB
	lock *sync.RWMutex
}

func (s *StxoDB) Put(stxo wallet.Stxo) error {
	s.lock.Lock()
	defer s.lock.Unlock()
	tx, _ := s.db.Begin()
	stmt, err := tx.Prepare("insert or replace into stxos(outpoint, value, height, scriptPubKey, watchOnly, spendHeight, spendTxid) values(?,?,?,?,?,?,?)")
	defer stmt.Close()
	if err != nil {
		tx.Rollback()
		return err
	}
	watchOnly := 0
	if stxo.Utxo.WatchOnly {
		watchOnly = 1
	}
	outpoint := stxo.Utxo.Op.Hash.String() + ":" + strconv.Itoa(int(stxo.Utxo.Op.Index))
	_, err = stmt.Exec(outpoint, stxo.Utxo.Value, int(stxo.Utxo.AtHeight), hex.EncodeToString(stxo.Utxo.ScriptPubkey), watchOnly, int(stxo.SpendHeight), stxo.SpendTxid.String())
	if err != nil {
		tx.Rollback()
		return err
	}
	tx.Commit()
	return nil
}

func (s *StxoDB) GetAll() ([]wallet.Stxo, error) {
	s.lock.RLock()
	defer s.lock.RUnlock()
	var ret []wallet.Stxo
	stm := "select outpoint, value, height, scriptPubKey, watchOnly, spendHeight, spendTxid from stxos"
	rows, err := s.db.Query(stm)
	defer rows.Close()
	if err != nil {
		return ret, err
	}
	for rows.Next() {
		var outpoint string
		var value string
		var height int
		var scriptPubKey string
		var watchOnlyInt int
		var spendHeight int
		var spendTxid string
		if err := rows.Scan(&outpoint, &value, &height, &scriptPubKey, &watchOnlyInt, &spendHeight, &spendTxid); err != nil {
			continue
		}
		s := strings.Split(outpoint, ":")
		if err != nil {
			continue
		}
		shaHash, err := chainhash.NewHashFromStr(s[0])
		if err != nil {
			continue
		}
		index, err := strconv.Atoi(s[1])
		if err != nil {
			continue
		}
		watchOnly := false
		if watchOnlyInt > 0 {
			watchOnly = true
		}
		scriptBytes, err := hex.DecodeString(scriptPubKey)
		if err != nil {
			continue
		}
		spentHash, err := chainhash.NewHashFromStr(spendTxid)
		if err != nil {
			continue
		}
		utxo := wallet.Utxo{
			Op:           *wire.NewOutPoint(shaHash, uint32(index)),
			AtHeight:     int32(height),
			Value:        value,
			ScriptPubkey: scriptBytes,
			WatchOnly:    watchOnly,
		}
		ret = append(ret, wallet.Stxo{
			Utxo:        utxo,
			SpendHeight: int32(spendHeight),
			SpendTxid:   *spentHash,
		})
	}
	return ret, nil
}

func (s *StxoDB) Delete(stxo wallet.Stxo) error {
	s.lock.Lock()
	defer s.lock.Unlock()
	outpoint := stxo.Utxo.Op.Hash.String() + ":" + strconv.Itoa(int(stxo.Utxo.Op.Index))
	_, err := s.db.Exec("delete from stxos where outpoint=?", outpoint)
	if err != nil {
		return err
	}
	return nil
}
