package spvwallet

import (
	"net"
	"strings"

	"github.com/btcsuite/btcd/chaincfg"
	"github.com/btcsuite/btcd/wire"
)

type ConnectionState int

const (
	CONNECTING = 0
	CONNECTED  = 1
	DEAD       = 2
)

// NewPeer creates a a new *Peer and begins communicating with it.
func NewPeer(remoteNode string, blockchain *Blockchain, inTs *TxStore, params *chaincfg.Params, userAgent string, diconnectChan chan string, downloadPeer bool) *Peer {
	// I should really merge SPVCon and TxStore, they're basically the same
	inTs.Param = params

	// format if ipv6 addr
	ip := net.ParseIP(remoteNode)
	if ip.To4() == nil {
		li := strings.LastIndex(remoteNode, ":")
		remoteNode = "[" + remoteNode[:li] + "]" + remoteNode[li:len(remoteNode)]
	}

	// create new peer
	return &Peer{
		TS: inTs, // copy pointer of txstore into peer

		blockchain:     blockchain,
		remoteAddress:  remoteNode,
		disconnectChan: diconnectChan,
		downloadPeer:   downloadPeer,
		OKTxids:        make(map[wire.ShaHash]int32),
		shutdownCh:     make(chan struct{}),

		// assign version bits for local node
		localVersion: VERSION,
		userAgent:    userAgent,
	}
}

// start begins communicating with the peer. It sends a version message and waits
// for a reply. If that handshake completes it sets the data returned and then
// spawns read/write loops.
func (p *Peer) start() error {
	var err error

	// open TCP connection
	p.con, err = net.Dial("tcp", p.remoteAddress)
	if err != nil {
		log.Debugf("Connection to %s failed", p.remoteAddress)
		p.disconnectChan <- p.remoteAddress
		return err
	}

	// prepare a version message for our node
	myMsgVer, err := wire.NewMsgVersionFromConn(p.con, 0, 0)
	if err != nil {
		p.disconnectChan <- p.remoteAddress
		return err
	}
	err = myMsgVer.AddUserAgent(p.userAgent, WALLET_VERSION)
	if err != nil {
		p.disconnectChan <- p.remoteAddress
		return err
	}
	myMsgVer.DisableRelayTx = true

	// send the message
	n, err := wire.WriteMessageN(p.con, myMsgVer, p.localVersion, p.TS.Param.Net)
	if err != nil {
		p.disconnectChan <- p.remoteAddress
		return err
	}
	p.WBytes += uint64(n)
	log.Debugf("Sent version message to %s\n", p.con.RemoteAddr().String())

	// read a response
	n, m, _, err := wire.ReadMessageN(p.con, p.localVersion, p.TS.Param.Net)
	if err != nil {
		p.disconnectChan <- p.remoteAddress
		return err
	}
	p.RBytes += uint64(n)
	log.Debugf("Received %s message from %s\n", m.Command(), p.con.RemoteAddr().String())

	// if the response is correct and supports bloom filtering we're connected
	mv, ok := m.(*wire.MsgVersion)
	if ok {
		log.Infof("Connected to %s on %s", mv.UserAgent, p.con.RemoteAddr().String())
	} else {
		p.disconnectChan <- p.remoteAddress
		return err
	}
	if !strings.Contains(mv.Services.String(), "SFNodeBloom") {
		p.disconnectChan <- p.remoteAddress
		return err
	}

	// set remote height and connected state
	p.remoteHeight = mv.LastBlock
	p.connectionState = CONNECTED

	// ack the received message
	mva := wire.NewMsgVerAck()
	n, err = wire.WriteMessageN(p.con, mva, p.localVersion, p.TS.Param.Net)
	if err != nil {
		p.disconnectChan <- p.remoteAddress
		return err
	}
	p.WBytes += uint64(n)

	// begin read/write loops
	go p.incomingMessageHandler()
	p.outMsgQueue = make(chan wire.Message)
	go p.outgoingMessageHandler()

	// create initial filter
	filt, err := p.TS.GimmeFilter()
	if err != nil {
		p.disconnectChan <- p.remoteAddress
		return err
	}

	// send filter
	p.SendFilter(filt)
	log.Debugf("Sent filter to %s\n", p.con.RemoteAddr().String())

	// create queues for blocks and false positives
	p.blockQueue = make(chan HashAndHeight, 32)
	p.fPositives = make(chan int32, 4000) // a block full, approx
	go p.fPositiveHandler()

	// if this peer is a downloadPeer ask it for headers
	if p.downloadPeer {
		log.Infof("Set %s as download peer", p.con.RemoteAddr().String())
		p.AskForHeaders()
	}

	return nil
}

// stop shuts down all communication and closes all goroutines
func (p *Peer) stop() {
	close(p.outMsgQueue)
	close(p.fPositives)
	close(p.shutdownCh)

	p.con.Close()
	p.connectionState = DEAD
}
