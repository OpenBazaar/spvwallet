package api

import (
	"errors"
	"github.com/OpenBazaar/spvwallet"
	"github.com/OpenBazaar/spvwallet/api/pb"
	"github.com/btcsuite/btcd/chaincfg"
	"github.com/btcsuite/btcd/chaincfg/chainhash"
	"github.com/btcsuite/btcutil"
	"github.com/golang/protobuf/ptypes"
	"golang.org/x/net/context"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
	"net"
)

const Addr = "127.0.0.1:8234"

type server struct {
	w *spvwallet.SPVWallet
}

func ServeAPI(w *spvwallet.SPVWallet) error {
	lis, err := net.Listen("tcp", Addr)
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

func (s *server) NewAddress(ctx context.Context, in *pb.KeySelection) (*pb.Address, error) {
	var purpose spvwallet.KeyPurpose
	if in.Purpose == pb.KeyPurpose_INTERNAL {
		purpose = spvwallet.INTERNAL
	} else if in.Purpose == pb.KeyPurpose_EXTERNAL {
		purpose = spvwallet.EXTERNAL
	} else {
		return nil, errors.New("Unknown key purpose")
	}
	addr := s.w.NewAddress(purpose)
	return &pb.Address{addr.String()}, nil
}

func (s *server) ChainTip(ctx context.Context, in *pb.Empty) (*pb.Height, error) {
	return &pb.Height{s.w.ChainTip()}, nil
}

func (s *server) Balance(ctx context.Context, in *pb.Empty) (*pb.Balances, error) {
	confirmed, unconfirmed := s.w.Balance()
	return &pb.Balances{uint64(confirmed), uint64(unconfirmed)}, nil
}

func (s *server) MasterPrivateKey(ctx context.Context, in *pb.Empty) (*pb.Key, error) {
	return &pb.Key{s.w.MasterPrivateKey().String()}, nil
}

func (s *server) MasterPublicKey(ctx context.Context, in *pb.Empty) (*pb.Key, error) {
	return &pb.Key{s.w.MasterPublicKey().String()}, nil
}

func (s *server) Params(ctx context.Context, in *pb.Empty) (*pb.NetParams, error) {
	return &pb.NetParams{s.w.Params().Name}, nil
}

func (s *server) HasKey(ctx context.Context, in *pb.Address) (*pb.BoolResponse, error) {
	params, err := s.Params(ctx, &pb.Empty{})
	if err != nil {
		return nil, err
	}
	var p chaincfg.Params
	switch params.Name {
	case chaincfg.TestNet3Params.Name:
		p = chaincfg.TestNet3Params
	case chaincfg.MainNetParams.Name:
		p = chaincfg.MainNetParams
	case chaincfg.RegressionNetParams.Name:
		p = chaincfg.RegressionNetParams
	default:
		return nil, errors.New("Unknown network parameters")
	}
	addr, err := btcutil.DecodeAddress(in.Addr, &p)
	if err != nil {
		return nil, err
	}
	return &pb.BoolResponse{s.w.HasKey(addr)}, nil
}

func (s *server) Transactions(ctx context.Context, in *pb.Empty) (*pb.TransactionList, error) {
	txs, err := s.w.Transactions()
	if err != nil {
		return nil, err
	}
	var list []*pb.Tx
	for _, tx := range txs {
		ts, err := ptypes.TimestampProto(tx.Timestamp)
		if err != nil {
			return nil, err
		}
		respTx := &pb.Tx{
			Txid:      tx.Txid,
			Value:     tx.Value,
			Height:    tx.Height,
			WatchOnly: tx.WatchOnly,
			Timestamp: ts,
			Raw:       tx.Bytes,
		}
		list = append(list, respTx)
	}
	return &pb.TransactionList{list}, nil
}

func (s *server) GetTransaction(ctx context.Context, in *pb.Txid) (*pb.Tx, error) {
	ch, err := chainhash.NewHashFromStr(in.Hash)
	if err != nil {
		return nil, err
	}
	tx, err := s.w.GetTransaction(*ch)
	if err != nil {
		return nil, err
	}
	ts, err := ptypes.TimestampProto(tx.Timestamp)
	if err != nil {
		return nil, err
	}
	respTx := &pb.Tx{
		Txid:      tx.Txid,
		Value:     tx.Value,
		Height:    tx.Height,
		WatchOnly: tx.WatchOnly,
		Timestamp: ts,
		Raw:       tx.Bytes,
	}
	return respTx, nil
}

func (s *server) GetFeePerByte(ctx context.Context, in *pb.FeeLevelSelection) (*pb.FeePerByte, error) {
	var feeLevel spvwallet.FeeLevel
	switch in.FeeLevel {
	case pb.FeeLevel_ECONOMIC:
		feeLevel = spvwallet.ECONOMIC
	case pb.FeeLevel_NORMAL:
		feeLevel = spvwallet.NORMAL
	case pb.FeeLevel_PRIORITY:
		feeLevel = spvwallet.PRIOIRTY
	default:
		return nil, errors.New("Unknown fee level")
	}
	return &pb.FeePerByte{s.w.GetFeePerByte(feeLevel)}, nil
}
