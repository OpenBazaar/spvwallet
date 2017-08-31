package db

import (
	"database/sql"
	"github.com/OpenBazaar/wallet-interface"
	_ "github.com/mattn/go-sqlite3"
	"path"
	"sync"
	"time"
)

// This database is mostly just an example implementation used for testing.
// End users are free to user their own database.
type SQLiteDatastore struct {
	keys           wallet.Keys
	utxos          wallet.Utxos
	stxos          wallet.Stxos
	txns           wallet.Txns
	watchedScripts wallet.WatchedScripts
	db             *sql.DB
	lock           *sync.RWMutex
}

func Create(repoPath string) (*SQLiteDatastore, error) {
	dbPath := path.Join(repoPath, "wallet.db")
	conn, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		return nil, err
	}

	l := new(sync.RWMutex)
	sqliteDB := &SQLiteDatastore{
		keys: &KeysDB{
			db:   conn,
			lock: l,
		},
		utxos: &UtxoDB{
			db:   conn,
			lock: l,
		},
		stxos: &StxoDB{
			db:   conn,
			lock: l,
		},
		txns: &TxnsDB{
			db:   conn,
			lock: l,
		},
		watchedScripts: &WatchedScriptsDB{
			db:   conn,
			lock: l,
		},
		db:   conn,
		lock: l,
	}
	initDatabaseTables(conn)
	return sqliteDB, nil
}

func (db *SQLiteDatastore) Keys() wallet.Keys {
	return db.keys
}
func (db *SQLiteDatastore) Utxos() wallet.Utxos {
	return db.utxos
}
func (db *SQLiteDatastore) Stxos() wallet.Stxos {
	return db.stxos
}
func (db *SQLiteDatastore) Txns() wallet.Txns {
	return db.txns
}
func (db *SQLiteDatastore) WatchedScripts() wallet.WatchedScripts {
	return db.watchedScripts
}

func initDatabaseTables(db *sql.DB) error {
	var sqlStmt string
	sqlStmt = sqlStmt + `
	create table if not exists keys (scriptAddress text primary key not null, purpose integer, keyIndex integer, used integer, key text);
	create table if not exists utxos (outpoint text primary key not null, value integer, height integer, scriptPubKey text, watchOnly integer);
	create table if not exists stxos (outpoint text primary key not null, value integer, height integer, scriptPubKey text, watchOnly integer, spendHeight integer, spendTxid text);
	create table if not exists txns (txid text primary key not null, value integer, height integer, timestamp integer, watchOnly integer, tx blob);
	create table if not exists watchedScripts (scriptPubKey text primary key not null);
	create table if not exists config(key text primary key not null, value blob);
	`
	_, err := db.Exec(sqlStmt)
	if err != nil {
		return err
	}
	return nil
}

func (s *SQLiteDatastore) GetMnemonic() (string, error) {
	s.lock.RLock()
	defer s.lock.RUnlock()
	stmt, err := s.db.Prepare("select value from config where key=?")
	defer stmt.Close()
	var mnemonic string
	err = stmt.QueryRow("mnemonic").Scan(&mnemonic)
	if err != nil {
		return "", err
	}
	return mnemonic, nil
}

func (s *SQLiteDatastore) SetMnemonic(mnemonic string) error {
	s.lock.RLock()
	defer s.lock.RUnlock()
	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	stmt, err := tx.Prepare("insert or replace into config(key, value) values(?,?)")
	if err != nil {
		return err
	}
	defer stmt.Close()
	_, err = stmt.Exec("mnemonic", mnemonic)
	if err != nil {
		tx.Rollback()
		return err
	}
	tx.Commit()
	return nil
}

func (s *SQLiteDatastore) GetCreationDate() (time.Time, error) {
	s.lock.RLock()
	defer s.lock.RUnlock()
	var t time.Time
	stmt, err := s.db.Prepare("select value from config where key=?")
	if err != nil {
		return t, err
	}
	defer stmt.Close()
	var creationDate []byte
	err = stmt.QueryRow("creationDate").Scan(&creationDate)
	if err != nil {
		return t, err
	}
	return time.Parse(time.RFC3339, string(creationDate))
}

func (s *SQLiteDatastore) SetCreationDate(creationDate time.Time) error {
	s.lock.RLock()
	defer s.lock.RUnlock()
	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	stmt, err := tx.Prepare("insert or replace into config(key, value) values(?,?)")
	if err != nil {
		return err
	}
	defer stmt.Close()
	_, err = stmt.Exec("creationDate", creationDate.Format(time.RFC3339))
	if err != nil {
		tx.Rollback()
		return err
	}
	tx.Commit()
	return nil
}
