package main

import (
	"os"
	"sync"
	"fmt"
	"github.com/OpenBazaar/spvwallet"
	"github.com/btcsuite/btcd/chaincfg"
	"github.com/op/go-logging"
	"github.com/OpenBazaar/spvwallet/examples/db"
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

	// example database
	database, err := db.Create("/home/chris/.openbazaar2")
	if err != nil {
		fmt.Println(err)
	}

	mnemonic := "salon around sketch ivory analyst vital erosion shift organ hub assault notice"

	// Start up a new node
	spvwallet.NewSPVWallet(mnemonic, &chaincfg.TestNet3Params, 1000, 60, 40, 20, "", "/home/chris/.openbazaar2", database, "OpenBazaar", backendStdoutFormatter)
	wg.Wait()
}
