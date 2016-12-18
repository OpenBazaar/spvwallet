package spvwallet

import (
	"errors"
	"github.com/btcsuite/btcd/chaincfg/chainhash"
	"github.com/btcsuite/btcd/peer"
	"github.com/btcsuite/btcd/wire"
)

func (s *SPVWallet) askForTx(p *peer.Peer, txid chainhash.Hash) {
	gdata := wire.NewMsgGetData()
	inv := wire.NewInvVect(wire.InvTypeTx, &txid)
	gdata.AddInvVect(inv)
	p.QueueMessage(gdata, nil)
}

func (s *SPVWallet) askForMerkleBlock(p *peer.Peer, hash chainhash.Hash) {
	m := wire.NewMsgGetData()
	m.AddInvVect(wire.NewInvVect(wire.InvTypeFilteredBlock, &hash))
	p.QueueMessage(m, nil)
}

func (s *SPVWallet) askForHeaders(p *peer.Peer) {
	ghdr := wire.NewMsgGetHeaders()
	ghdr.ProtocolVersion = p.ProtocolVersion()

	ghdr.BlockLocatorHashes = s.blockchain.GetBlockLocatorHashes()

	log.Debugf("Sending getheaders message to peer%d\n", p.ID())
	p.QueueMessage(ghdr, nil)
}

// HashAndHeight is needed instead of just height in case a fullnode
// responds abnormally (?) by sending out of order merkleblocks.
// we cache a merkleroot:height pair in the queue so we don't have to
// look them up from the disk.
// Also used when inv messages indicate blocks so we can add the header
// and parse the txs in one request instead of requesting headers first.
type HashAndHeight struct {
	blockhash chainhash.Hash
	height    int32
	final     bool // indicates this is the last merkleblock requested
}

// NewRootAndHeight saves like 2 lines.
func NewRootAndHeight(b chainhash.Hash, h int32) (hah HashAndHeight) {
	hah.blockhash = b
	hah.height = h
	return
}

func (s *SPVWallet) askForBlocks(p *peer.Peer) error {
	headerTip, err := s.blockchain.db.Height()
	if err != nil {
		return err
	}

	walletTip := s.walletSyncHeight

	log.Debugf("WalletTip %d HeaderTip %d\n", walletTip, headerTip)
	if uint32(walletTip) > headerTip {
		return errors.New("Wallet tip greater than headers! shouldn't happen.")
	}

	if uint32(walletTip) == headerTip {
		// nothing to ask for; set wait state and return
		log.Debugf("No blocks to request, entering wait state\n")
		if s.blockchain.ChainState() != WAITING {
			log.Info("Blockchain fully synced")
		}
		s.txstore.SetDBSyncHeight(walletTip)
		s.blockchain.SetChainState(WAITING)
		// also advertise any unconfirmed txs here
		s.rebroadcast()
		return nil
	}

	log.Debugf("Will request blocks %d to %d\n", walletTip+1, headerTip)
	hashes := s.blockchain.GetNPrevBlockHashes(int(headerTip - uint32(walletTip)))

	// loop through all heights where we want merkleblocks.
	for i := len(hashes) - 1; i >= 0; i-- {
		walletTip++
		iv1 := wire.NewInvVect(wire.InvTypeFilteredBlock, hashes[i])
		gdataMsg := wire.NewMsgGetData()
		// add inventory
		err = gdataMsg.AddInvVect(iv1)
		if err != nil {
			return err
		}

		hah := NewRootAndHeight(*hashes[i], walletTip)
		if uint32(walletTip) == headerTip { // if this is the last block, indicate finality
			hah.final = true
		}
		// waits here most of the time for the queue to empty out
		s.blockQueue <- hah // push height and mroot of requested block on queue
		p.QueueMessage(gdataMsg, nil)
	}
	return nil
}

func (s *SPVWallet) onMerkleBlock(p *peer.Peer, m *wire.MsgMerkleBlock) {
	if s.blockchain.ChainState() == WAITING {
		go s.ingestBlockAndHeader(p, m)
	} else {
		go s.ingestMerkleBlock(p, m)
	}
}

func (s *SPVWallet) ingestBlockAndHeader(p *peer.Peer, m *wire.MsgMerkleBlock) {
	txids, err := checkMBlock(m) // check self-consistency
	if err != nil {
		log.Errorf("Merkle block error: %s\n", err.Error())
		return
	}

	success, err := s.blockchain.CommitHeader(m.Header)
	if err != nil {
		log.Error(err)
		return
	}
	var height uint32
	if success {
		h, err := s.blockchain.db.Height()
		height = h
		if err != nil {
			log.Error(err)
			return
		}
		s.walletSyncHeight = int32(h)
		s.txstore.SetDBSyncHeight(int32(h))
	} else {
		bestSH, err := s.blockchain.db.GetBestHeader()
		height = bestSH.height
		if err != nil {
			log.Error(err)
			return
		}
		headerHash := m.Header.BlockHash()
		tipHash := bestSH.header.BlockHash()
		if !tipHash.IsEqual(&headerHash) {
			return
		}
	}
	s.mutex.Lock()
	defer s.mutex.Unlock()
	for _, txid := range txids {
		s.toDownload[*txid] = int32(height)
	}
	log.Debugf("Received Merkle Block %s from peer%d", m.Header.BlockHash().String(), p.ID())
}

func (s *SPVWallet) ingestMerkleBlock(p *peer.Peer, m *wire.MsgMerkleBlock) {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	txids, err := checkMBlock(m) // check self-consistency
	if err != nil {
		log.Debugf("Merkle block error: %s\n", err.Error())
		return
	}
	var hah HashAndHeight
	select { // select here so we don't block on an unrequested mblock
	case hah = <-s.blockQueue: // pop height off mblock queue
		break
	default:
		log.Warning("Unrequested merkle block")
		return
	}

	// this verifies order, and also that the returned header fits
	// into our SPV header file
	newMerkBlockSha := m.Header.BlockHash()
	if !hah.blockhash.IsEqual(&newMerkBlockSha) {
		// This implies we may miss transactions in this block.
		log.Errorf("merkle block out of order got %s expect %s",
			m.Header.BlockHash().String(), hah.blockhash.String())
		return
	}
	for _, txid := range txids {
		s.toDownload[*txid] = hah.height
	}
	s.walletSyncHeight = hah.height
	if hah.height%2016 == 0 {
		s.txstore.SetDBSyncHeight(hah.height)
	}
	if hah.final {
		// don't set waitstate; instead, ask for headers again!
		// this way the only thing that triggers waitstate is asking for headers,
		// getting 0, calling AskForMerkBlocks(), and seeing you don't need any.
		// that way you are pretty sure you're synced up.
		s.askForHeaders(p)
	}
	log.Debugf("Ingested Merkle Block %s at height %d", m.Header.BlockHash().String(), hah.height)
	return
}

func (s *SPVWallet) onHeaders(p *peer.Peer, m *wire.MsgHeaders) {
	moar, err := s.IngestHeaders(p, m)
	if err != nil {
		log.Errorf("Header error: %s\n", err.Error())
		return
	}
	// more to get? if so, ask for them and return
	if moar {
		s.askForHeaders(p)
		return
	}

	// no moar, done w/ headers, get blocks
	go s.askForBlocks(p)
}

// TxHandler takes in transaction messages that come in from either a request
// after an inv message or after a merkle block message.
func (s *SPVWallet) onTx(p *peer.Peer, m *wire.MsgTx) {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	height, ok := s.toDownload[m.TxHash()]
	if !ok {
		log.Warningf("Received unknown tx: %s", m.TxHash().String())
		return
	}
	hits, err := s.txstore.Ingest(m, height)
	if err != nil {
		log.Errorf("Incoming Tx error: %s\n", err.Error())
		return
	}
	delete(s.toDownload, m.TxHash())
	if hits == 0 {
		log.Debugf("Tx %s from peer%d had no hits, filter false positive.",
			m.TxHash().String(), p.ID())
		s.fPositives <- p // add one false positive to chan
		return
	}
	s.UpdateFilterAndSend(p)
	log.Noticef("Tx %s ingested and matches %d utxo/adrs.", m.TxHash().String(), hits)
}

func (s *SPVWallet) onInv(p *peer.Peer, m *wire.MsgInv) {
	for _, thing := range m.InvList {
		if thing.Type == wire.InvTypeTx {
			// new tx, OK it at 0 and request
			s.mutex.Lock()
			s.toDownload[thing.Hash] = 0
			s.askForTx(p, thing.Hash)
			s.mutex.Unlock()
		}
		if thing.Type == wire.InvTypeBlock { // new block what to do?
			switch {
			case s.blockchain.ChainState() == WAITING:
				s.askForMerkleBlock(p, thing.Hash)
			default:
				// drop it as if its component particles had high thermal energies
				log.Debug("Received inv block but ignoring; not synched\n")
			}
		}
	}
}

func (s *SPVWallet) GetDataHandler(p *peer.Peer, m *wire.MsgGetData) {
	log.Debugf("Received getdata request from peer%d\n", p.ID())
	var sent int32
	for _, thing := range m.InvList {
		if thing.Type == wire.InvTypeTx {
			tx, err := s.txstore.Txns().Get(thing.Hash)
			if err != nil {
				log.Errorf("Error getting tx %s: %s",
					thing.Hash.String(), err.Error())
			}
			p.QueueMessage(tx, nil)
			sent++
			continue
		}
		// Didn't match, so it's not something we're responding to
		log.Debugf("We only respond to tx requests, ignoring")

	}
	log.Debugf("Sent %d of %d requested items to peer%d", sent, len(m.InvList), p.ID())
}

// IngestHeaders takes in a bunch of headers and appends them to the
// local header file, checking that they fit.  If there's no headers,
// it assumes we're done and returns false.  If it worked it assumes there's
// more to request and returns true.
func (s *SPVWallet) IngestHeaders(p *peer.Peer, m *wire.MsgHeaders) (bool, error) {
	gotNum := int64(len(m.Headers))
	if gotNum > 0 {
		log.Debugf("Received %d headers from peer%d, validating...", gotNum, p.ID())
	} else {
		log.Debugf("Received 0 headers from peer%d, we're probably synced up", p.ID())
		if s.blockchain.ChainState() == SYNCING {
			log.Info("Headers fully synced")
		}
		return false, nil
	}
	for _, resphdr := range m.Headers {
		_, err := s.blockchain.CommitHeader(*resphdr)
		if err != nil {
			// probably should disconnect from spv node at this point,
			// since they're giving us invalid headers.
			return true, errors.New("Returned header didn't fit in chain")
		}
	}
	height, _ := s.blockchain.db.Height()
	log.Debugf("Headers to height %d OK.", height)
	return true, nil
}

func (s *SPVWallet) fPositiveHandler() {
	for {
		peer := <-s.fPositives // blocks here

		totalFP, _ := s.fpAccumulator[peer.ID()]
		totalFP++
		if totalFP > 7 {
			s.UpdateFilterAndSend(peer)

			log.Debugf("Reset %d false positives for peer%d\n", totalFP, peer.ID())
			// reset accumulator
			totalFP = 0
		}
		s.fpAccumulator[peer.ID()] = totalFP
	}
}

func (s *SPVWallet) UpdateFilterAndSend(p *peer.Peer) {
	filt, err := s.txstore.GimmeFilter()
	if err != nil {
		log.Errorf("Filter creation error: %s\n", err.Error())
		return
	}
	// send filter
	p.QueueMessage(filt.MsgFilterLoad(), nil)
	log.Debugf("Sent filter to peer%d\n", p.ID())
}

// Rebroadcast sends an inv message of all the unconfirmed txs the db is
// aware of.  This is called after every sync.
func (s *SPVWallet) rebroadcast() {
	// get all unconfirmed txs
	invMsg, err := s.txstore.GetPendingInv()
	if err != nil {
		log.Errorf("Rebroadcast error: %s", err.Error())
	}
	if len(invMsg.InvList) == 0 { // nothing to broadcast, so don't
		return
	}
	for _, p := range s.PeerManager.ConnectedPeers() {
		p.QueueMessage(invMsg, nil)
	}
	return
}
