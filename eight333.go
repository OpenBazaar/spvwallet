package spvwallet

import (
	"errors"
	"github.com/btcsuite/btcd/chaincfg/chainhash"
	"github.com/btcsuite/btcd/peer"
	"github.com/btcsuite/btcd/wire"
	"sync"
)

type Eight333 struct {
	*Blockchain
	*TxStore
	blockQueue chan HashAndHeight
	toDownload map[*chainhash.Hash]int32
	mutex      *sync.Mutex
}

func (e *Eight333) askForTx(p *peer.Peer, txid chainhash.Hash) {
	gdata := wire.NewMsgGetData()
	inv := wire.NewInvVect(wire.InvTypeTx, &txid)
	gdata.AddInvVect(inv)
	p.QueueMessage(gdata, nil)
}

func (e *Eight333) askForMerkleBlock(p *peer.Peer, hash chainhash.Hash) {
	m := wire.NewMsgGetData()
	m.AddInvVect(wire.NewInvVect(wire.InvTypeFilteredBlock, &hash))
	p.QueueMessage(m, nil)
}

func (e *Eight333) askForHeaders(p *peer.Peer) {
	ghdr := wire.NewMsgGetHeaders()
	ghdr.ProtocolVersion = p.ProtocolVersion()

	ghdr.BlockLocatorHashes = e.GetBlockLocatorHashes()

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

func (e *Eight333) askForBlocks(p *peer.Peer) error {
	headerTip, err := e.db.Height()
	if err != nil {
		return err
	}

	walletTip, err := e.GetDBSyncHeight()
	if err != nil {
		return err
	}

	log.Debugf("WalletTip %d HeaderTip %d\n", walletTip, headerTip)
	if uint32(walletTip) > headerTip {
		return errors.New("Wallet tip greater than headers! shouldn't happen.")
	}

	if uint32(walletTip) == headerTip {
		// nothing to ask for; set wait state and return
		log.Debugf("No blocks to request, entering wait state\n")
		if e.ChainState() != WAITING {
			log.Info("Blockchain fully synced")
		}
		e.SetChainState(WAITING)
		// also advertise any unconfirmed txs here
		//TODO p.Rebroadcast()
		return nil
	}

	log.Debugf("Will request blocks %d to %d\n", walletTip+1, headerTip)
	hashes := e.GetNPrevBlockHashes(int(headerTip - uint32(walletTip)))

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
		e.blockQueue <- hah // push height and mroot of requested block on queue
		p.QueueMessage(gdataMsg, nil)
		p.q
	}
	return nil
}

func (e *Eight333) IngestBlockAndHeader(p *peer.Peer, m *wire.MsgMerkleBlock) {
	txids, err := checkMBlock(m) // check self-consistency
	if err != nil {
		log.Errorf("Merkle block error: %s\n", err.Error())
		return
	}

	success, err := e.CommitHeader(m.Header)
	if err != nil {
		log.Error(err)
		return
	}
	var height uint32
	if success {
		h, err := e.db.Height()
		height = h
		if err != nil {
			log.Error(err)
			return
		}
		e.SetDBSyncHeight(int32(h))
	} else {
		bestSH, err := e.db.GetBestHeader()
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
	e.mutex.Lock()
	for _, txid := range txids {
		e.toDownload[txid] = int32(height)
	}
	e.mutex.Unlock()
	log.Debugf("Received Merkle Block %s from peer%d", m.Header.BlockHash().String(), p.ID())
}

func (e *Eight333) onMerkleBlock(p *peer.Peer, m *wire.MsgMerkleBlock) {
	// TODO: maybe acquire lock?
	txids, err := checkMBlock(m) // check self-consistency
	if err != nil {
		log.Debugf("Merkle block error: %s\n", err.Error())
		return
	}
	var hah HashAndHeight
	select { // select here so we don't block on an unrequested mblock
	case hah = <-e.blockQueue: // pop height off mblock queue
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
		e.toDownload[txid] = hah.height
	}
	// write to db that we've sync'd to the height indicated in the
	// merkle block.  This isn't QUITE true since we haven't actually gotten
	// the txs yet but if there are problems with the txs we should backtrack.
	err = e.SetDBSyncHeight(hah.height)
	if err != nil {
		log.Errorf("Merkle block error: %s\n", err.Error())
		return
	}
	if hah.final {
		// don't set waitstate; instead, ask for headers again!
		// this way the only thing that triggers waitstate is asking for headers,
		// getting 0, calling AskForMerkBlocks(), and seeing you don't need any.
		// that way you are pretty sure you're synced up.
		e.askForHeaders(p)
	}
	log.Debugf("Ingested Merkle Block %s at height %d", m.Header.BlockHash().String(), hah.height)
	return
}

func (e *Eight333) onHeaders(p *peer.Peer, m *wire.MsgHeaders) {
	moar, err := e.IngestHeaders(p, m)
	if err != nil {
		log.Errorf("Header error: %s\n", err.Error())
		return
	}
	// more to get? if so, ask for them and return
	if moar {
		e.askForHeaders(p)
		return
	}

	// no moar, done w/ headers, get blocks
	err = e.askForBlocks(p)
	if err != nil {
		log.Errorf("AskForBlocks error: %s", err.Error())
		return
	}
}

// IngestHeaders takes in a bunch of headers and appends them to the
// local header file, checking that they fit.  If there's no headers,
// it assumes we're done and returns false.  If it worked it assumes there's
// more to request and returns true.
func (e *Eight333) IngestHeaders(p *peer.Peer, m *wire.MsgHeaders) (bool, error) {
	gotNum := int64(len(m.Headers))
	if gotNum > 0 {
		log.Debugf("Received %d headers from peer%d, validating...", gotNum, p.ID())
	} else {
		log.Debugf("Received 0 headers from peer%d, we're probably synced up", p.ID())
		if e.ChainState() == SYNCING {
			log.Info("Headers fully synced")
		}
		return false, nil
	}
	for _, resphdr := range m.Headers {
		_, err := e.CommitHeader(*resphdr)
		if err != nil {
			// probably should disconnect from spv node at this point,
			// since they're giving us invalid headers.
			return true, errors.New("Returned header didn't fit in chain")
		}
	}
	height, _ := e.db.Height()
	log.Debugf("Headers to height %d OK.", height)
	return true, nil
}

func onRead(p *peer.Peer, bytesRead int, msg wire.Message, err error) {
	log.Noticef("%t", msg)
}

func onWrite(p *peer.Peer, bytesWritten int, msg wire.Message, err error) {
	log.Warningf("%t", msg)
}
