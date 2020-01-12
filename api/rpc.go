package api

import (
	"encoding/hex"
	"errors"
	"math/big"
	"net"
	"strconv"
	"sync"

	"github.com/OpenBazaar/wallet-interface"
	"github.com/btcsuite/btcd/chaincfg"
	"github.com/btcsuite/btcd/chaincfg/chainhash"
	"github.com/btcsuite/btcutil"
	"github.com/btcsuite/btcutil/hdkeychain"
	"github.com/golang/protobuf/ptypes"
	"github.com/golang/protobuf/ptypes/timestamp"
	"golang.org/x/net/context"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"

	"github.com/OpenBazaar/spvwallet"
	"github.com/OpenBazaar/spvwallet/api/pb"
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
	var purpose wallet.KeyPurpose
	if in.Purpose == pb.KeyPurpose_INTERNAL {
		purpose = wallet.INTERNAL
	} else if in.Purpose == pb.KeyPurpose_EXTERNAL {
		purpose = wallet.EXTERNAL
	} else {
		return nil, errors.New("Unknown key purpose")
	}
	addr := s.w.CurrentAddress(purpose)
	return &pb.Address{addr.String()}, nil
}

func (s *server) NewAddress(ctx context.Context, in *pb.KeySelection) (*pb.Address, error) {
	var purpose wallet.KeyPurpose
	if in.Purpose == pb.KeyPurpose_INTERNAL {
		purpose = wallet.INTERNAL
	} else if in.Purpose == pb.KeyPurpose_EXTERNAL {
		purpose = wallet.EXTERNAL
	} else {
		return nil, errors.New("Unknown key purpose")
	}
	addr := s.w.NewAddress(purpose)
	return &pb.Address{addr.String()}, nil
}

func (s *server) ChainTip(ctx context.Context, in *pb.Empty) (*pb.Height, error) {
	h, _ := s.w.ChainTip()
	return &pb.Height{h}, nil
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
		val0, _ := strconv.ParseInt(tx.Value, 10, 64)
		respTx := &pb.Tx{
			Txid:      tx.Txid,
			Value:     val0,
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
	val0, _ := strconv.ParseInt(tx.Value, 10, 64)
	respTx := &pb.Tx{
		Txid:      tx.Txid,
		Value:     val0,
		Height:    tx.Height,
		WatchOnly: tx.WatchOnly,
		Timestamp: ts,
		Raw:       tx.Bytes,
	}
	return respTx, nil
}

func (s *server) GetFeePerByte(ctx context.Context, in *pb.FeeLevelSelection) (*pb.FeePerByte, error) {
	var feeLevel wallet.FeeLevel
	switch in.FeeLevel {
	case pb.FeeLevel_ECONOMIC:
		feeLevel = wallet.ECONOMIC
	case pb.FeeLevel_NORMAL:
		feeLevel = wallet.NORMAL
	case pb.FeeLevel_PRIORITY:
		feeLevel = wallet.PRIOIRTY
	default:
		return nil, errors.New("Unknown fee level")
	}
	return &pb.FeePerByte{s.w.GetFeePerByte(feeLevel)}, nil
}

func (s *server) Spend(ctx context.Context, in *pb.SpendInfo) (*pb.Txid, error) {
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
	var feeLevel wallet.FeeLevel
	switch in.FeeLevel {
	case pb.FeeLevel_ECONOMIC:
		feeLevel = wallet.ECONOMIC
	case pb.FeeLevel_NORMAL:
		feeLevel = wallet.NORMAL
	case pb.FeeLevel_PRIORITY:
		feeLevel = wallet.PRIOIRTY
	default:
		return nil, errors.New("Unknown fee level")
	}
	addr, err := btcutil.DecodeAddress(in.Address, &p)
	if err != nil {
		return nil, err
	}
	txid, err := s.w.Spend(int64(in.Amount), addr, feeLevel, "", false)
	if err != nil {
		return nil, err
	}
	return &pb.Txid{txid.String()}, nil
}

func (s *server) BumpFee(ctx context.Context, in *pb.Txid) (*pb.Txid, error) {
	ch, err := chainhash.NewHashFromStr(in.Hash)
	if err != nil {
		return nil, err
	}
	txid, err := s.w.BumpFee(*ch)
	if err != nil {
		return nil, err
	}
	return &pb.Txid{txid.String()}, nil
}

func (s *server) Peers(ctx context.Context, in *pb.Empty) (*pb.PeerList, error) {
	var peers []*pb.Peer
	for _, peer := range s.w.ConnectedPeers() {
		ts, err := ptypes.TimestampProto(peer.TimeConnected())
		if err != nil {
			return nil, err
		}
		p := &pb.Peer{
			Address:         peer.Addr(),
			BytesSent:       peer.BytesSent(),
			BytesReceived:   peer.BytesReceived(),
			Connected:       peer.Connected(),
			ID:              peer.ID(),
			LastBlock:       peer.LastBlock(),
			ProtocolVersion: peer.ProtocolVersion(),
			Services:        peer.Services().String(),
			UserAgent:       peer.UserAgent(),
			TimeConnected:   ts,
		}
		peers = append(peers, p)
	}
	return &pb.PeerList{peers}, nil
}

func (s *server) AddWatchedAddress(ctx context.Context, in *pb.Address) (*pb.Empty, error) {
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
	return nil, s.w.AddWatchedAddress(addr)
}

func (s *server) GetConfirmations(ctx context.Context, in *pb.Txid) (*pb.Confirmations, error) {
	ch, err := chainhash.NewHashFromStr(in.Hash)
	if err != nil {
		return nil, err
	}
	confirms, _, err := s.w.GetConfirmations(*ch)
	if err != nil {
		return nil, err
	}
	return &pb.Confirmations{confirms}, nil
}

func (s *server) SweepAddress(ctx context.Context, in *pb.SweepInfo) (*pb.Txid, error) {
	var ins []wallet.TransactionInput
	for _, u := range in.Utxos {
		h, err := chainhash.NewHashFromStr(u.Txid)
		if err != nil {
			return nil, err
		}
		in := wallet.TransactionInput{
			OutpointHash:  h.CloneBytes(),
			OutpointIndex: u.Index,
			Value:         *big.NewInt(int64(u.Value)),
		}
		ins = append(ins, in)
	}
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
	var addr *btcutil.Address
	if in.Address != "" {
		a, err := btcutil.DecodeAddress(in.Address, &p)
		if err != nil {
			return nil, err
		}
		addr = &a
	} else {
		addr = nil
	}
	var key *hdkeychain.ExtendedKey
	wif, err := btcutil.DecodeWIF(in.Key)
	if err == nil {
		key = hdkeychain.NewExtendedKey(
			p.HDPrivateKeyID[:],
			wif.PrivKey.Serialize(),
			make([]byte, 32),
			make([]byte, 4),
			0,
			0,
			true)
	} else {
		keyBytes, err := hex.DecodeString(in.Key)
		if err == nil {
			key = hdkeychain.NewExtendedKey(
				p.HDPrivateKeyID[:],
				keyBytes,
				make([]byte, 32),
				make([]byte, 4),
				0,
				0,
				true)
		} else {
			key, err = hdkeychain.NewKeyFromString(in.Key)
			if err != nil {
				return nil, err
			}
		}
	}
	var rs *[]byte
	if len(in.RedeemScript) > 0 {
		rs = &in.RedeemScript
	}
	var feeLevel wallet.FeeLevel
	switch in.FeeLevel {
	case pb.FeeLevel_ECONOMIC:
		feeLevel = wallet.ECONOMIC
	case pb.FeeLevel_NORMAL:
		feeLevel = wallet.NORMAL
	case pb.FeeLevel_PRIORITY:
		feeLevel = wallet.PRIOIRTY
	default:
		return nil, errors.New("Unknown fee level")
	}
	newTxid, err := s.w.SweepAddress(ins, addr, key, rs, feeLevel)
	if err != nil {
		return nil, err
	}
	return &pb.Txid{newTxid.String()}, nil
}

func (s *server) ReSyncBlockchain(ctx context.Context, in *timestamp.Timestamp) (*pb.Empty, error) {
	t, err := ptypes.Timestamp(in)
	if err != nil {
		return nil, err
	}
	s.w.ReSyncBlockchain(t)
	return &pb.Empty{}, nil
}

func (s *server) CreateMultisigSignature(ctx context.Context, in *pb.CreateMultisigInfo) (*pb.SignatureList, error) {
	var ins []wallet.TransactionInput
	for _, input := range in.Inputs {
		h, err := hex.DecodeString(input.Txid)
		if err != nil {
			return nil, err
		}
		i := wallet.TransactionInput{
			OutpointHash:  h,
			OutpointIndex: input.Index,
		}
		ins = append(ins, i)
	}
	var outs []wallet.TransactionOutput
	for _, output := range in.Outputs {
		addr, err := s.w.ScriptToAddress(output.ScriptPubKey)
		if err != nil {
			return nil, err
		}
		o := wallet.TransactionOutput{
			Address: addr,
			Value:   *big.NewInt(int64(output.Value)),
		}
		outs = append(outs, o)
	}
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
	var key *hdkeychain.ExtendedKey
	wif, err := btcutil.DecodeWIF(in.Key)
	if err == nil {
		key = hdkeychain.NewExtendedKey(
			p.HDPrivateKeyID[:],
			wif.PrivKey.Serialize(),
			make([]byte, 32),
			make([]byte, 4),
			0,
			0,
			true)
	} else {
		keyBytes, err := hex.DecodeString(in.Key)
		if err == nil {
			key = hdkeychain.NewExtendedKey(
				p.HDPrivateKeyID[:],
				keyBytes,
				make([]byte, 32),
				make([]byte, 4),
				0,
				0,
				true)
		} else {
			key, err = hdkeychain.NewKeyFromString(in.Key)
			if err != nil {
				return nil, err
			}
		}
	}
	sigs, err := s.w.CreateMultisigSignature(ins, outs, key, in.RedeemScript, in.FeePerByte)
	if err != nil {
		return nil, err
	}
	var retSigs []*pb.Signature
	for _, s := range sigs {
		sig := &pb.Signature{
			Index:     s.InputIndex,
			Signature: s.Signature,
		}
		retSigs = append(retSigs, sig)
	}
	return &pb.SignatureList{retSigs}, nil
}

func (s *server) Multisign(ctx context.Context, in *pb.MultisignInfo) (*pb.RawTx, error) {
	var ins []wallet.TransactionInput
	for _, input := range in.Inputs {
		h, err := hex.DecodeString(input.Txid)
		if err != nil {
			return nil, err
		}
		i := wallet.TransactionInput{
			OutpointHash:  h,
			OutpointIndex: input.Index,
		}
		ins = append(ins, i)
	}
	var outs []wallet.TransactionOutput
	for _, output := range in.Outputs {
		addr, err := s.w.ScriptToAddress(output.ScriptPubKey)
		if err != nil {
			return nil, err
		}
		o := wallet.TransactionOutput{
			Address: addr,
			Value:   *big.NewInt(int64(output.Value)),
		}
		outs = append(outs, o)
	}
	var sig1 []wallet.Signature
	for _, s := range in.Sig1 {
		sig := wallet.Signature{
			InputIndex: s.Index,
			Signature:  s.Signature,
		}
		sig1 = append(sig1, sig)
	}
	var sig2 []wallet.Signature
	for _, s := range in.Sig2 {
		sig := wallet.Signature{
			InputIndex: s.Index,
			Signature:  s.Signature,
		}
		sig2 = append(sig2, sig)
	}
	tx, err := s.w.Multisign(ins, outs, sig1, sig2, in.RedeemScript, in.FeePerByte, in.Broadcast)
	if err != nil {
		return nil, err
	}
	return &pb.RawTx{tx}, nil
}

func (s *server) EstimateFee(ctx context.Context, in *pb.EstimateFeeData) (*pb.Fee, error) {
	var ins []wallet.TransactionInput
	for _, input := range in.Inputs {
		h, err := hex.DecodeString(input.Txid)
		if err != nil {
			return nil, err
		}
		i := wallet.TransactionInput{
			OutpointHash:  h,
			OutpointIndex: input.Index,
		}
		ins = append(ins, i)
	}
	var outs []wallet.TransactionOutput
	for _, output := range in.Outputs {
		addr, err := s.w.ScriptToAddress(output.ScriptPubKey)
		if err != nil {
			return nil, err
		}
		o := wallet.TransactionOutput{
			Address: addr,
			Value:   *big.NewInt(int64(output.Value)),
		}
		outs = append(outs, o)
	}
	fee := s.w.EstimateFee(ins, outs, in.FeePerByte)
	return &pb.Fee{fee}, nil
}

func (s *server) WalletNotify(in *pb.Empty, stream pb.API_WalletNotifyServer) error {
	cb := func(tx wallet.TransactionCallback) {
		ts, err := ptypes.TimestampProto(tx.Timestamp)
		if err != nil {
			return
		}
		resp := &pb.Tx{
			Txid:      tx.Txid,
			Value:     tx.Value.Int64(),
			Height:    tx.Height,
			Timestamp: ts,
			WatchOnly: tx.WatchOnly,
		}
		if err := stream.Send(resp); err != nil {
			return
		}
	}
	s.w.AddTransactionListener(cb)
	// Keep the connection open to continue streaming
	var wg sync.WaitGroup
	wg.Add(1)
	wg.Wait()
	return nil
}

type HeaderWriter struct {
	stream pb.API_DumpHeadersServer
}

func (h *HeaderWriter) Write(p []byte) (n int, err error) {
	hdr := &pb.Header{string(p)}
	if err := h.stream.Send(hdr); err != nil {
		return 0, err
	}
	return 0, nil
}

func (s *server) DumpHeaders(in *pb.Empty, stream pb.API_DumpHeadersServer) error {
	writer := HeaderWriter{stream}
	s.w.DumpHeaders(&writer)
	return nil
}

func (s *server) GetKey(ctx context.Context, in *pb.Address) (*pb.Key, error) {
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
	key, err := s.w.GetKey(addr)
	if err != nil {
		return nil, err
	}
	wif, err := btcutil.NewWIF(key, &p, true)
	if err != nil {
		return nil, err
	}
	return &pb.Key{wif.String()}, nil
}

func (s *server) ListAddresses(ctx context.Context, in *pb.Empty) (*pb.Addresses, error) {
	addrs := s.w.ListAddresses()
	var list []*pb.Address
	for _, addr := range addrs {
		ret := new(pb.Address)
		ret.Addr = addr.String()
		list = append(list, ret)
	}
	return &pb.Addresses{list}, nil
}

func (s *server) ListKeys(ctx context.Context, in *pb.Empty) (*pb.Keys, error) {
	keys := s.w.ListKeys()
	var list []*pb.Key
	for _, key := range keys {
		ret := new(pb.Key)
		wif, err := btcutil.NewWIF(&key, s.w.Params(), true)
		if err != nil {
			return nil, err
		}
		ret.Key = wif.String()
		list = append(list, ret)
	}
	return &pb.Keys{list}, nil
}
