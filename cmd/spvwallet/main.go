package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/OpenBazaar/openbazaar-go/bitcoin/exchange"
	"github.com/OpenBazaar/spvwallet"
	"github.com/OpenBazaar/spvwallet/api"
	"github.com/OpenBazaar/spvwallet/cli"
	"github.com/OpenBazaar/spvwallet/db"
	"github.com/OpenBazaar/spvwallet/gui"
	"github.com/OpenBazaar/spvwallet/gui/bootstrap"
	"github.com/asticode/go-astilectron"
	"github.com/asticode/go-astilog"
	"github.com/btcsuite/btcd/chaincfg"
	"github.com/fatih/color"
	"github.com/jessevdk/go-flags"
	"github.com/natefinch/lumberjack"
	"github.com/op/go-logging"
	"github.com/yawning/bulb"
	"net"
	"net/url"
	"os"
	"os/signal"
	"path"
	"time"
)

var parser = flags.NewParser(nil, flags.Default)

type Start struct {
	DataDir            string `short:"d" long:"datadir" description:"specify the data directory to be used"`
	Testnet            bool   `short:"t" long:"testnet" description:"use the test network"`
	Regtest            bool   `short:"r" long:"regtest" description:"run in regression test mode"`
	Mnemonic           string `short:"m" long:"mnemonic" description:"specify a mnemonic seed to use to derive the keychain"`
	WalletCreationDate string `short:"w" long:"walletcreationdate" description:"specify the date the seed was created. if omitted the wallet will sync from the oldest checkpoint."`
	TrustedPeer        string `short:"i" long:"trustedpeer" description:"specify a single trusted peer to connect to"`
	Tor                bool   `long:"tor" description:"connect via a running Tor daemon"`
	FeeAPI             string `short:"f" long:"feeapi" description:"fee API to use to fetch current fee rates. set as empty string to disable API lookups." default:"https://bitcoinfees.21.co/api/v1/fees/recommended"`
	MaxFee             uint64 `short:"x" long:"maxfee" description:"the fee-per-byte ceiling beyond which fees cannot go" default:"2000"`
	LowDefaultFee      uint64 `short:"e" long:"economicfee" description:"the default low fee-per-byte" default:"140"`
	MediumDefaultFee   uint64 `short:"n" long:"normalfee" description:"the default medium fee-per-byte" default:"160"`
	HighDefaultFee     uint64 `short:"p" long:"priorityfee" description:"the default high fee-per-byte" default:"180"`
	Gui                bool   `long:"gui" description:"launch an experimental GUI"`
}
type Version struct{}

var start Start
var version Version
var wallet *spvwallet.SPVWallet

func main() {
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt)
	go func() {
		for range c {
			fmt.Println("SPVWallet shutting down...")
			wallet.Close()
			os.Exit(1)
		}
	}()
	parser.AddCommand("start",
		"start the wallet",
		"The start command starts the wallet daemon",
		&start)
	parser.AddCommand("version",
		"print the version number",
		"Print the version number and exit",
		&version)
	cli.SetupCli(parser)
	if _, err := parser.Parse(); err != nil {
		os.Exit(1)
	}
}

func (x *Version) Execute(args []string) error {
	fmt.Println(spvwallet.WALLET_VERSION)
	return nil
}

func (x *Start) Execute(args []string) error {
	var err error
	// Create a new config
	config := spvwallet.NewDefaultConfig()
	if x.DataDir != "" {
		config.RepoPath = x.DataDir
	}
	if x.Testnet && x.Regtest {
		return errors.New("Invalid combination of testnet and regtest modes")
	}
	basepath := config.RepoPath
	if x.Testnet {
		config.Params = &chaincfg.TestNet3Params
		config.RepoPath = path.Join(config.RepoPath, "testnet")
	}
	if x.Regtest {
		config.Params = &chaincfg.RegressionNetParams
		config.RepoPath = path.Join(config.RepoPath, "regtest")
	}
	creationDate := time.Now()
	if x.WalletCreationDate != "" {
		creationDate, err = time.Parse(time.RFC3339, x.WalletCreationDate)
		if err != nil {
			return errors.New("Wallet creation date timestamp must be in RFC3339 format")
		}
	}
	config.CreationDate = creationDate

	_, ferr := os.Stat(config.RepoPath)
	if os.IsNotExist(ferr) {
		os.Mkdir(config.RepoPath, os.ModePerm)
	}
	if x.Mnemonic != "" {
		config.Mnemonic = x.Mnemonic
	}
	if x.TrustedPeer != "" {
		addr, err := net.ResolveTCPAddr("tcp", x.TrustedPeer)
		if err != nil {
			return err
		}
		config.TrustedPeer = addr
	}
	if x.Tor {
		var conn *bulb.Conn
		conn, err = bulb.Dial("tcp4", "127.0.0.1:9151")
		if err != nil {
			conn, err = bulb.Dial("tcp4", "127.0.0.1:9151")
			if err != nil {
				return errors.New("Tor daemon not found")
			}
		}
		dialer, err := conn.Dialer(nil)
		if err != nil {
			return err
		}
		config.Proxy = dialer
	}
	if x.FeeAPI != "" {
		u, err := url.Parse(x.FeeAPI)
		if err != nil {
			return err
		}
		config.FeeAPI = *u
	}
	config.MaxFee = x.MaxFee
	config.LowFee = x.LowDefaultFee
	config.MediumFee = x.MediumDefaultFee
	config.HighFee = x.HighDefaultFee

	// Make the logging a little prettier
	var fileLogFormat = logging.MustStringFormatter(`%{time:15:04:05.000} [%{shortfunc}] [%{level}] %{message}`)
	w := &lumberjack.Logger{
		Filename:   path.Join(config.RepoPath, "logs", "bitcoin.log"),
		MaxSize:    10, // Megabytes
		MaxBackups: 3,
		MaxAge:     30, // Days
	}
	bitcoinFile := logging.NewLogBackend(w, "", 0)
	bitcoinFileFormatter := logging.NewBackendFormatter(bitcoinFile, fileLogFormat)
	config.Logger = logging.MultiLogger(logging.MultiLogger(bitcoinFileFormatter))

	// Select wallet datastore
	sqliteDatastore, _ := db.Create(config.RepoPath)
	config.DB = sqliteDatastore

	mn, _ := sqliteDatastore.GetMnemonic()
	if mn != "" {
		config.Mnemonic = mn
	}

	// Create the wallet
	wallet, err = spvwallet.NewSPVWallet(config)
	if err != nil {
		return err
	}

	if err := sqliteDatastore.SetMnemonic(config.Mnemonic); err != nil {
		return err
	}

	go api.ServeAPI(wallet)

	// Start it!
	printSplashScreen()

	if x.Gui {
		exchangeRates := exchange.NewBitcoinPriceFetcher(nil)

		type Stats struct {
			Confirmed    int64  `json:"confirmed"`
			Fiat         string `json:"fiat"`
			Transactions int    `json:"transactions"`
			Height       uint32 `json:"height"`
		}

		//go wallet.Start()
		os.RemoveAll(path.Join(basepath, "resources"))
		iconPath := path.Join(basepath, "icon.png")
		_, err := os.Stat(iconPath)
		if os.IsNotExist(err) {
			f, err := os.Create(iconPath)
			if err != nil {
				return err
			}
			icon, err := gui.AppIconPngBytes()
			if err != nil {
				return err
			}
			f.Write(icon)
			defer f.Close()
		}

		// Run bootstrap
		if err := bootstrap.Run(bootstrap.Options{
			AstilectronOptions: astilectron.Options{
				AppName:            "spvwallet",
				AppIconDefaultPath: iconPath,
				//AppIconDarwinPath:  p + "/gopher.icns",
				BaseDirectoryPath: basepath,
			},
			Homepage: "index.html",
			MessageHandler: func(w *astilectron.Window, m bootstrap.MessageIn) {
				switch m.Name {
				case "getStats":
					type P struct {
						CurrencyCode string `json:"currencyCode"`
					}
					var p P
					if err := json.Unmarshal(m.Payload, &p); err != nil {
						astilog.Errorf("Unmarshaling %s failed", m.Payload)
						return
					}
					confirmed, _ := wallet.Balance()
					txs, err := wallet.Transactions()
					if err != nil {
						astilog.Errorf(err.Error())
						return
					}
					rate, err := exchangeRates.GetExchangeRate(p.CurrencyCode)
					if err != nil {
						astilog.Errorf("Failed to get exchange rate")
						return
					}
					btcVal := confirmed / 100000000
					fiatVal := float64(btcVal) * rate
					height := wallet.ChainTip()

					st := Stats{
						Confirmed:    confirmed,
						Fiat:         fmt.Sprintf("%.2f", fiatVal),
						Transactions: len(txs),
						Height:       height,
					}
					w.Send(bootstrap.MessageOut{Name: "statsUpdate", Payload: st})
				}
			},
			RestoreAssets: gui.RestoreAssets,
			WindowOptions: &astilectron.WindowOptions{
				Center:         astilectron.PtrBool(true),
				Height:         astilectron.PtrInt(370),
				Width:          astilectron.PtrInt(757),
				Maximizable:    astilectron.PtrBool(false),
				Fullscreenable: astilectron.PtrBool(false),
				Resizable:      astilectron.PtrBool(false),
			},
			TrayOptions: &astilectron.TrayOptions{
				Image: astilectron.PtrStr(iconPath),
			},
			BaseDirectoryPath: basepath,
		}); err != nil {
			astilog.Fatal(err)
		}
	} else {
		wallet.Start()
	}
	return nil
}

func printSplashScreen() {
	blue := color.New(color.FgBlue)
	white := color.New(color.FgWhite)
	white.Printf("  _______________________   ______")
	blue.Println("      __        .__  .__          __")
	white.Printf(` /   _____/\______   \   \ /   /`)
	blue.Println(`  \    /  \_____  |  | |  |   _____/  |_`)
	white.Printf(` \_____  \  |     ___/\   Y   /`)
	blue.Println(`\   \/\/   /\__  \ |  | |  | _/ __ \   __\`)
	white.Printf(` /        \ |    |     \     / `)
	blue.Println(` \        /  / __ \|  |_|  |_\  ___/|  |`)
	white.Printf(`/_______  / |____|      \___/ `)
	blue.Println(`   \__/\  /  (____  /____/____/\___  >__|`)
	white.Printf(`	\/ `)
	blue.Println(`                           \/        \/               \/`)
	blue.DisableColor()
	white.DisableColor()
	fmt.Println("")
	fmt.Println("SPVWallet v" + spvwallet.WALLET_VERSION + " starting...")
	fmt.Println("[Press Ctrl+C to exit]")
}
