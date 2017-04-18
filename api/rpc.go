package api

import (
	"errors"
	"github.com/OpenBazaar/spvwallet"
	"github.com/OpenBazaar/spvwallet/api/pb"
	"golang.org/x/net/context"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
	"net"
)

const Port = ":8234"

type server struct {
	w *spvwallet.SPVWallet
}

func ServeAPI(w *spvwallet.SPVWallet) error {
	lis, err := net.Listen("tcp", Port)
	if err != nil {
		return err
	}
	s := grpc.NewServer()
	pb.RegisterAPIServer(s, &server{w})
	reflection.Register(s)
	if err := s.Serve(lis); err != nil {
		return err
	}
	return nil
}

func (s *server) Stop(ctx context.Context, in *pb.Empty) (*pb.Empty, error) {
	s.w.Close()
	return &pb.Empty{}, nil
}

func (s *server) CurrentAddress(ctx context.Context, in *pb.KeySelection) (*pb.Address, error) {
	var purpose spvwallet.KeyPurpose
	if in.Purpose == pb.KeyPurpose_INTERNAL {
		purpose = spvwallet.INTERNAL
	} else if in.Purpose == pb.KeyPurpose_EXTERNAL {
		purpose = spvwallet.EXTERNAL
	} else {
		return nil, errors.New("Unknown key purpose")
	}
	addr := s.w.CurrentAddress(purpose)
	return &pb.Address{addr.String()}, nil
}
