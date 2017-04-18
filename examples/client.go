package main

import (
	"fmt"
	"github.com/OpenBazaar/spvwallet"
	"github.com/OpenBazaar/spvwallet/db"
	"github.com/btcsuite/btcd/chaincfg"
	"github.com/op/go-logging"
	"os"
)

func main() {
	// Create a new config
	config := spvwallet.NewDefaultConfig()

	// Make the logging a little prettier
	backend := logging.NewLogBackend(os.Stdout, "", 0)
	formatter := logging.MustStringFormatter(`%{color:reset}%{color}%{time:15:04:05.000} [%{shortfunc}] [%{level}] %{message}`)
	stdoutFormatter := logging.NewBackendFormatter(backend, formatter)
	config.Logger = logging.MultiLogger(stdoutFormatter)

	// Use testnet
	config.Params = &chaincfg.TestNet3Params

	// Select wallet datastore
	sqliteDatastore, _ := db.Create(config.RepoPath)
	config.DB = sqliteDatastore

	// Create the wallet
	wallet, err := spvwallet.NewSPVWallet(config)
	if err != nil {
		fmt.Println(err)
		return
	}

	// Start it!
	wallet.Start()
}
