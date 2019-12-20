package db

import (
	"database/sql"
	"encoding/hex"
	"sync"
)

type WatchedScriptsDB struct {
	db   *sql.DB
	lock *sync.RWMutex
}

func (w *WatchedScriptsDB) Put(scriptPubKey []byte) error {
	w.lock.Lock()
	defer w.lock.Unlock()
	tx, _ := w.db.Begin()
	stmt, err := tx.Prepare("insert or replace into watchedScripts(scriptPubKey) values(?)")
	defer stmt.Close()
	if err != nil {
		tx.Rollback()
		return err
	}
	_, err = stmt.Exec(hex.EncodeToString(scriptPubKey))
	if err != nil {
		tx.Rollback()
		return err
	}
	tx.Commit()
	return nil
}

func (w *WatchedScriptsDB) PutAll(scripts [][]byte) error {
	for _, s := range scripts {
		if err := w.Put(s); err != nil {
			return err
		}
	}
	return nil
}

func (w *WatchedScriptsDB) GetAll() ([][]byte, error) {
	w.lock.RLock()
	defer w.lock.RUnlock()
	var ret [][]byte
	stm := "select scriptPubKey from watchedScripts"
	rows, err := w.db.Query(stm)
	defer rows.Close()
	if err != nil {
		return ret, err
	}
	for rows.Next() {
		var scriptHex string
		if err := rows.Scan(&scriptHex); err != nil {
			continue
		}
		scriptPubKey, err := hex.DecodeString(scriptHex)
		if err != nil {
			continue
		}
		ret = append(ret, scriptPubKey)
	}
	return ret, nil
}

func (w *WatchedScriptsDB) Delete(scriptPubKey []byte) error {
	w.lock.Lock()
	defer w.lock.Unlock()
	_, err := w.db.Exec("delete from watchedScripts where scriptPubKey=?", hex.EncodeToString(scriptPubKey))
	if err != nil {
		return err
	}
	return nil
}
