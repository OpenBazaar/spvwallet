package cli

import (
	"fmt"
	"github.com/OpenBazaar/spvwallet/api"
	"github.com/OpenBazaar/spvwallet/api/pb"
	"github.com/jessevdk/go-flags"
	"golang.org/x/net/context"
	"google.golang.org/grpc"
	"strings"
)

func SetupCli(parser *flags.Parser) {
	// Add commands to parser
	parser.AddCommand("stop",
		"stop the wallet",
		"The stop command disconnects from peers and shuts down the wallet",
		&stop)
	parser.AddCommand("currentaddress",
		"get a bitcoin address",
		"Returns the first unused address in the keychain\n\n"+
			"Args:\n"+
			"1. purpose       (string default=external) The purpose for the address. Can be external for receiving from outside parties or internal for example, for change.\n\n"+
			"Examples:\n"+
			"> spvwallet currentaddress\n"+
			"1DxGWC22a46VPEjq8YKoeVXSLzB7BA8sJS\n"+
			"> spvwallet currentaddress internal\n"+
			"18zAxgfKx4NuTUGUEuB8p7FKgCYPM15DfS\n",
		&currentAddress)
}

func newGRPCClient() (pb.APIClient, *grpc.ClientConn, error) {
	// Set up a connection to the server.
	conn, err := grpc.Dial(api.Port, grpc.WithInsecure())
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
