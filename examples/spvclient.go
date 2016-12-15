package main

import (
	"fmt"
	"github.com/OpenBazaar/spvwallet"
	"github.com/OpenBazaar/spvwallet/examples/db"
	"github.com/btcsuite/btcd/chaincfg"
	"github.com/op/go-logging"
	"os"
	"sync"
)

var stdoutLogFormat = logging.MustStringFormatter(
	`%{color:reset}%{color}%{time:15:04:05.000} [%{shortfunc}] [%{level}] %{message}`,
)

func main() {
	var wg sync.WaitGroup
	wg.Add(1)

	// logging
	backendStdout := logging.NewLogBackend(os.Stdout, "", 0)
	backendStdoutFormatter := logging.NewBackendFormatter(backendStdout, stdoutLogFormat)
	ml := logging.MultiLogger(backendStdoutFormatter)

	dirPath := "spvwallet"
	os.Mkdir(dirPath, os.ModePerm)
	// example database
	database, err := db.Create(dirPath)
	if err != nil {
		fmt.Println(err)
	}

	mnemonic := "salon around sketch ivory analyst vital erosion shift organ hub assault notice"

	// Start up a new node
	wallet := spvwallet.NewSPVWallet(mnemonic, &chaincfg.TestNet3Params, 1000, 60, 40, 20, "", dirPath, database, "OpenBazaar", "", ml)
	go wallet.Start()
	wg.Wait()
}
