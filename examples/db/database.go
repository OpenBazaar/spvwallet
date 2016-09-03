package db

import (
	"path"
	"sync"
	"database/sql"
	"github.com/OpenBazaar/spvwallet"
	_ "github.com/mattn/go-sqlite3"
)

// This database is mostly just an example implementation used for testing.
// End users are free to user their own database.
type SQLiteDatastore struct {
	keys 	        spvwallet.Keys
	utxos           spvwallet.Utxos
	stxos           spvwallet.Stxos
	txns            spvwallet.Txns
	state           spvwallet.State
	watchedScripts  spvwallet.WatchedScripts
	db              *sql.DB
	lock            *sync.Mutex
}

func Create(repoPath string) (*SQLiteDatastore, error) {
	dbPath := path.Join(repoPath, "state.db")
	conn, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		return nil, err
	}

	l := new(sync.Mutex)
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
		state: &StateDB{
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

func (db *SQLiteDatastore) Keys() spvwallet.Keys {
	return db.keys
}
func (db *SQLiteDatastore) Utxos() spvwallet.Utxos{
	return db.utxos
}
func (db *SQLiteDatastore) Stxos() spvwallet.Stxos{
	return db.stxos
}
func (db *SQLiteDatastore) Txns() spvwallet.Txns{
	return db.txns
}
func (db *SQLiteDatastore) State() spvwallet.State{
	return db.state
}
func (db *SQLiteDatastore) WatchedScripts() spvwallet.WatchedScripts{
	return db.watchedScripts
}

func initDatabaseTables(db *sql.DB) error {
	var sqlStmt string
	sqlStmt = sqlStmt + `
	create table if not exists keys (scriptPubKey text primary key not null, purpose integer, keyIndex integer, used integer);
	create table if not exists utxos (outpoint text primary key not null, value integer, height integer, scriptPubKey text, freeze integer);
	create table if not exists stxos (outpoint text primary key not null, value integer, height integer, scriptPubKey text, spendHeight integer, spendTxid text);
	create table if not exists txns (txid text primary key not null, tx blob);
	create table if not exists state (key text primary key not null, value text);
	create table if not exists watchedScripts (scriptPubKey text primary key not null);
	`
	_, err := db.Exec(sqlStmt)
	if err != nil {
		return err
	}
	return nil
}