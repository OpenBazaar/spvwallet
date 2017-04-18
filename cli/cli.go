package cli

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/OpenBazaar/spvwallet/api"
	"github.com/OpenBazaar/spvwallet/api/pb"
	"github.com/jessevdk/go-flags"
	"golang.org/x/net/context"
	"google.golang.org/grpc"
	"strconv"
	"strings"
	"time"
)

func SetupCli(parser *flags.Parser) {
	// Add commands to parser
	parser.AddCommand("stop",
		"stop the wallet",
		"The stop command disconnects from peers and shuts down the wallet",
		&stop)
	parser.AddCommand("currentaddress",
		"get the current bitcoin address",
		"Returns the first unused address in the keychain\n\n"+
			"Args:\n"+
			"1. purpose       (string default=external) The purpose for the address. Can be external for receiving from outside parties or internal for example, for change.\n\n"+
			"Examples:\n"+
			"> spvwallet currentaddress\n"+
			"1DxGWC22a46VPEjq8YKoeVXSLzB7BA8sJS\n"+
			"> spvwallet currentaddress internal\n"+
			"18zAxgfKx4NuTUGUEuB8p7FKgCYPM15DfS\n",
		&currentAddress)
	parser.AddCommand("newaddress",
		"get a new bitcoin address",
		"Returns a new unused address in the keychain. Use caution when using this function as generating too many new addresses may cause the keychain to extend further than the wallet's lookahead window, meaning it might fail to recover all transactions when restoring from seed. CurrentAddress is safer as it never extends past the lookahead window.\n\n"+
			"Args:\n"+
			"1. purpose       (string default=external) The purpose for the address. Can be external for receiving from outside parties or internal for example, for change.\n\n"+
			"Examples:\n"+
			"> spvwallet newaddress\n"+
			"1DxGWC22a46VPEjq8YKoeVXSLzB7BA8sJS\n"+
			"> spvwallet newaddress internal\n"+
			"18zAxgfKx4NuTUGUEuB8p7FKgCYPM15DfS\n",
		&newAddress)
	parser.AddCommand("chaintip",
		"return the height of the chain",
		"Returns the height of the best chain of headers",
		&chainTip)
	parser.AddCommand("balance",
		"get the wallet balance",
		"Returns both the confirmed and unconfirmed balances",
		&balance)
	parser.AddCommand("masterprivatekey",
		"get the wallet's master private key",
		"Returns the bip32 master private key",
		&masterPrivateKey)
	parser.AddCommand("masterpublickey",
		"get the wallet's master public key",
		"Returns the bip32 master public key",
		&masterPublicKey)
	parser.AddCommand("haskey",
		"does key exist",
		"Returns whether a key for the given address exists in the wallet\n\n"+
			"Args:\n"+
			"1. address       (string) The address to find a key for.\n\n"+
			"Examples:\n"+
			"> spvwallet haskey 1DxGWC22a46VPEjq8YKoeVXSLzB7BA8sJS\n"+
			"true\n",
		&hasKey)
	parser.AddCommand("transactions",
		"get a list of transactions",
		"Returns a json list of the wallet's transactions",
		&transactions)
	parser.AddCommand("gettransaction",
		"get a specific transaction",
		"Returns json data of a specific transaction\n\n"+
			"Args:\n"+
			"1. txid       (string) A transaction ID to seach for.\n\n"+
			"Examples:\n"+
			"> spvwallet gettransaction 190bd83935740b88ebdfe724485f36ca4aa40125a21b93c410e0e191d4e9e0b5\n",
		&getTransaction)
	parser.AddCommand("getfeeperbyte",
		"get the current bitcoin fee",
		"Returns the current network fee per byte for the given fee level.\n\n"+
			"Args:\n"+
			"1. feelevel       (string default=normal) The fee level: economic, normal, priority\n\n"+
			"Examples:\n"+
			"> spvwallet getfeeperbyte\n"+
			"140\n"+
			"> spvwallet getfeeperbyte priority\n"+
			"160\n",
		&getFeePerByte)
	parser.AddCommand("spend",
		"send bitcoins",
		"Send bitcoins to the given address\n\n"+
			"Args:\n"+
			"1. address       (string) The recipient's bitcoin address\n"+
			"2. amount        (integer) The amount to send in satoshi"+
			"3. feelevel      (string default=normal) The fee level: economic, normal, priority\n\n"+
			"Examples:\n"+
			"> spvwallet spend 1DxGWC22a46VPEjq8YKoeVXSLzB7BA8sJS 1000000\n"+
			"82bfd45f3564e0b5166ab9ca072200a237f78499576e9658b20b0ccd10ff325c"+
			"> spvwallet spend 1DxGWC22a46VPEjq8YKoeVXSLzB7BA8sJS 3000000000 priority\n"+
			"82bfd45f3564e0b5166ab9ca072200a237f78499576e9658b20b0ccd10ff325c",
		&spend)
	parser.AddCommand("bumpfee",
		"bump the tx fee",
		"Bumps the fee on an unconfirmed transaction\n\n"+
			"Args:\n"+
			"1. txid       (string) The transaction ID of the transaction to bump.\n\n"+
			"Examples:\n"+
			"> spvwallet bumpfee 190bd83935740b88ebdfe724485f36ca4aa40125a21b93c410e0e191d4e9e0b5\n"+
			"82bfd45f3564e0b5166ab9ca072200a237f78499576e9658b20b0ccd10ff325c",
		&bumpFee)
	parser.AddCommand("peers",
		"get info about peers",
		"Returns a list of json data on each connected peer",
		&peers)
}

func newGRPCClient() (pb.APIClient, *grpc.ClientConn, error) {
	// Set up a connection to the server.
	conn, err := grpc.Dial(api.Addr, grpc.WithInsecure())
	if err != nil {
		return nil, nil, err
	}
	client := pb.NewAPIClient(conn)
	return client, conn, nil
}

type Stop struct{}

var stop Stop

func (x *Stop) Execute(args []string) error {
	client, conn, err := newGRPCClient()
	if err != nil {
		return err
	}
	defer conn.Close()
	client.Stop(context.Background(), &pb.Empty{})
	return nil
}

type CurrentAddress struct{}

var currentAddress CurrentAddress

func (x *CurrentAddress) Execute(args []string) error {
	client, conn, err := newGRPCClient()
	if err != nil {
		return err
	}
	defer conn.Close()
	var purpose pb.KeyPurpose
	userSelection := ""
	if len(args) > 0 {
		userSelection = args[0]
	}
	switch strings.ToLower(userSelection) {
	case "internal":
		purpose = pb.KeyPurpose_INTERNAL
	case "external":
		purpose = pb.KeyPurpose_EXTERNAL
	default:
		purpose = pb.KeyPurpose_EXTERNAL
	}
	resp, err := client.CurrentAddress(context.Background(), &pb.KeySelection{purpose})
	if err != nil {
		return err
	}
	fmt.Println(resp.Addr)
	return nil
}

type NewAddress struct{}

var newAddress NewAddress

func (x *NewAddress) Execute(args []string) error {
	client, conn, err := newGRPCClient()
	if err != nil {
		return err
	}
	defer conn.Close()
	var purpose pb.KeyPurpose
	userSelection := ""
	if len(args) > 0 {
		userSelection = args[0]
	}
	switch strings.ToLower(userSelection) {
	case "internal":
		purpose = pb.KeyPurpose_INTERNAL
	case "external":
		purpose = pb.KeyPurpose_EXTERNAL
	default:
		purpose = pb.KeyPurpose_EXTERNAL
	}
	resp, err := client.NewAddress(context.Background(), &pb.KeySelection{purpose})
	if err != nil {
		return err
	}
	fmt.Println(resp.Addr)
	return nil
}

type ChainTip struct{}

var chainTip ChainTip

func (x *ChainTip) Execute(args []string) error {
	client, conn, err := newGRPCClient()
	if err != nil {
		return err
	}
	defer conn.Close()
	resp, err := client.ChainTip(context.Background(), &pb.Empty{})
	if err != nil {
		return err
	}
	fmt.Println(resp.Height)
	return nil
}

type Balance struct{}

var balance Balance

func (x *Balance) Execute(args []string) error {
	client, conn, err := newGRPCClient()
	if err != nil {
		return err
	}
	defer conn.Close()
	resp, err := client.Balance(context.Background(), &pb.Empty{})
	if err != nil {
		return err
	}
	type ret struct {
		Confirmed   uint64 `json:"confirmed"`
		Unconfirmed uint64 `json:"unconfirmed"`
	}
	out, err := json.MarshalIndent(&ret{resp.Confirmed, resp.Unconfirmed}, "", "    ")
	if err != nil {
		return err
	}
	fmt.Println(string(out))
	return nil
}

type MasterPrivateKey struct{}

var masterPrivateKey MasterPrivateKey

func (x *MasterPrivateKey) Execute(args []string) error {
	client, conn, err := newGRPCClient()
	if err != nil {
		return err
	}
	defer conn.Close()
	resp, err := client.MasterPrivateKey(context.Background(), &pb.Empty{})
	if err != nil {
		return err
	}
	fmt.Println(resp.Key)
	return nil
}

type MasterPublicKey struct{}

var masterPublicKey MasterPublicKey

func (x *MasterPublicKey) Execute(args []string) error {
	client, conn, err := newGRPCClient()
	if err != nil {
		return err
	}
	defer conn.Close()
	resp, err := client.MasterPublicKey(context.Background(), &pb.Empty{})
	if err != nil {
		return err
	}
	fmt.Println(resp.Key)
	return nil
}

type HasKey struct{}

var hasKey HasKey

func (x *HasKey) Execute(args []string) error {
	client, conn, err := newGRPCClient()
	if err != nil {
		return err
	}
	defer conn.Close()
	if len(args) <= 0 {
		return errors.New("Bitcoin address is required")
	}
	resp, err := client.HasKey(context.Background(), &pb.Address{args[0]})
	if err != nil {
		return err
	}
	fmt.Println(resp.Bool)
	return nil
}

type Transactions struct{}

var transactions Transactions

func (x *Transactions) Execute(args []string) error {
	client, conn, err := newGRPCClient()
	if err != nil {
		return err
	}
	defer conn.Close()
	resp, err := client.Transactions(context.Background(), &pb.Empty{})
	if err != nil {
		return err
	}
	chainTip, err := client.ChainTip(context.Background(), &pb.Empty{})
	if err != nil {
		return err
	}
	type Tx struct {
		Txid          string    `json:"txid"`
		Value         int64     `json:"value"`
		Status        string    `json:"status"`
		Timestamp     time.Time `json:"timestamp"`
		Confirmations int32     `json:"confirmations"`
		Height        int32     `json:"height"`
		WatchOnly     bool      `json:"watchOnly"`
	}
	var txns []Tx
	for _, tx := range resp.Transactions {
		var confirmations int32
		var status string
		confs := int32(chainTip.Height) - tx.Height + 1
		if tx.Height <= 0 {
			confs = tx.Height
		}
		ts := time.Unix(int64(tx.Timestamp.Seconds), int64(tx.Timestamp.Nanos))
		switch {
		case confs < 0:
			status = "DEAD"
		case confs == 0 && time.Since(ts) <= time.Hour*6:
			status = "UNCONFIRMED"
		case confs == 0 && time.Since(ts) > time.Hour*6:
			status = "STUCK"
		case confs > 0 && confs < 7:
			status = "PENDING"
			confirmations = confs
		case confs > 6:
			status = "CONFIRMED"
			confirmations = confs
		}
		t := Tx{
			Txid:          tx.Txid,
			Value:         tx.Value,
			Height:        tx.Height,
			WatchOnly:     tx.WatchOnly,
			Timestamp:     ts,
			Status:        status,
			Confirmations: confirmations,
		}
		txns = append(txns, t)
	}
	formatted, err := json.MarshalIndent(txns, "", "    ")
	if err != nil {
		return err
	}
	fmt.Println(string(formatted))
	return nil
}

type GetTransaction struct{}

var getTransaction GetTransaction

func (x *GetTransaction) Execute(args []string) error {
	client, conn, err := newGRPCClient()
	if err != nil {
		return err
	}
	defer conn.Close()
	if len(args) <= 0 {
		return errors.New("Txid is required")
	}
	resp, err := client.GetTransaction(context.Background(), &pb.Txid{args[0]})
	if err != nil {
		return err
	}
	chainTip, err := client.ChainTip(context.Background(), &pb.Empty{})
	if err != nil {
		return err
	}
	type Tx struct {
		Txid          string    `json:"txid"`
		Value         int64     `json:"value"`
		Status        string    `json:"status"`
		Timestamp     time.Time `json:"timestamp"`
		Confirmations int32     `json:"confirmations"`
		Height        int32     `json:"height"`
		WatchOnly     bool      `json:"watchOnly"`
	}
	var confirmations int32
	var status string
	confs := int32(chainTip.Height) - resp.Height + 1
	if resp.Height <= 0 {
		confs = resp.Height
	}
	ts := time.Unix(int64(resp.Timestamp.Seconds), int64(resp.Timestamp.Nanos))
	switch {
	case confs < 0:
		status = "DEAD"
	case confs == 0 && time.Since(ts) <= time.Hour*6:
		status = "UNCONFIRMED"
	case confs == 0 && time.Since(ts) > time.Hour*6:
		status = "STUCK"
	case confs > 0 && confs < 7:
		status = "PENDING"
		confirmations = confs
	case confs > 6:
		status = "CONFIRMED"
		confirmations = confs
	}
	t := Tx{
		Txid:          resp.Txid,
		Value:         resp.Value,
		Height:        resp.Height,
		WatchOnly:     resp.WatchOnly,
		Timestamp:     ts,
		Status:        status,
		Confirmations: confirmations,
	}
	formatted, err := json.MarshalIndent(t, "", "    ")
	if err != nil {
		return err
	}
	fmt.Println(string(formatted))
	return nil
}

type GetFeePerByte struct{}

var getFeePerByte GetFeePerByte

func (x *GetFeePerByte) Execute(args []string) error {
	client, conn, err := newGRPCClient()
	if err != nil {
		return err
	}
	defer conn.Close()
	var feeLevel pb.FeeLevel
	userSelection := ""
	if len(args) > 0 {
		userSelection = args[0]
	}
	switch strings.ToLower(userSelection) {
	case "economic":
		feeLevel = pb.FeeLevel_ECONOMIC
	case "normal":
		feeLevel = pb.FeeLevel_NORMAL
	case "priority":
		feeLevel = pb.FeeLevel_PRIORITY
	default:
		feeLevel = pb.FeeLevel_NORMAL
	}
	resp, err := client.GetFeePerByte(context.Background(), &pb.FeeLevelSelection{feeLevel})
	if err != nil {
		return err
	}
	fmt.Println(resp.Fee)
	return nil
}

type Spend struct{}

var spend Spend

func (x *Spend) Execute(args []string) error {
	client, conn, err := newGRPCClient()
	if err != nil {
		return err
	}
	defer conn.Close()
	var feeLevel pb.FeeLevel
	userSelection := ""
	if len(args) > 2 {
		userSelection = args[2]
	}
	switch strings.ToLower(userSelection) {
	case "economic":
		feeLevel = pb.FeeLevel_ECONOMIC
	case "normal":
		feeLevel = pb.FeeLevel_NORMAL
	case "priority":
		feeLevel = pb.FeeLevel_PRIORITY
	default:
		feeLevel = pb.FeeLevel_NORMAL
	}
	amt, err := strconv.Atoi(args[1])
	if err != nil {
		return err
	}
	resp, err := client.Spend(context.Background(), &pb.SpendInfo{args[0], uint64(amt), feeLevel})
	if err != nil {
		return err
	}
	fmt.Println(resp.Hash)
	return nil
}

type BumpFee struct{}

var bumpFee BumpFee

func (x *BumpFee) Execute(args []string) error {
	client, conn, err := newGRPCClient()
	if err != nil {
		return err
	}
	defer conn.Close()
	if len(args) <= 0 {
		return errors.New("Txid is required")
	}
	resp, err := client.BumpFee(context.Background(), &pb.Txid{args[0]})
	if err != nil {
		return err
	}
	fmt.Println(resp.Hash)
	return nil
}

type Peers struct{}

var peers Peers

func (x *Peers) Execute(args []string) error {
	client, conn, err := newGRPCClient()
	if err != nil {
		return err
	}
	defer conn.Close()
	resp, err := client.Peers(context.Background(), &pb.Empty{})
	if err != nil {
		return err
	}
	type peer struct {
		Address         string    `json:"address"`
		BytesSent       uint64    `json:"bytesSent"`
		BytesReceived   uint64    `json:"bytesReceived"`
		Connected       bool      `json:"connected"`
		ID              int32     `json:"id"`
		LastBlock       int32     `json:"lastBlock"`
		ProtocolVersion uint32    `json:"protocolVersion"`
		Services        string    `json:"services"`
		UserAgent       string    `json:"userAgent"`
		TimeConnected   time.Time `json:"timeConnected"`
	}
	var peers []peer
	for _, p := range resp.Peers {
		pi := peer{
			Address:         p.Address,
			BytesSent:       p.BytesSent,
			BytesReceived:   p.BytesReceived,
			Connected:       p.Connected,
			ID:              p.ID,
			LastBlock:       p.LastBlock,
			ProtocolVersion: p.ProtocolVersion,
			Services:        p.Services,
			UserAgent:       p.UserAgent,
			TimeConnected:   time.Unix(int64(p.TimeConnected.Seconds), int64(p.TimeConnected.Nanos)),
		}
		peers = append(peers, pi)
	}
	out, err := json.MarshalIndent(peers, "", "    ")
	fmt.Println(string(out))
	return nil
}
