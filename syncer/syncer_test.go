package syncer

import (
	"testing"

	"time"

	"github.com/viteshan/naive-vite/common"
	"github.com/viteshan/naive-vite/common/log"
	"github.com/viteshan/naive-vite/p2p"
)

type TestPeer struct {
	fn p2p.MsgHandle
}

func (self *TestPeer) Write(msg *p2p.Msg) error {
	log.Info("msgType:%d", msg.Type())
	return nil
}

func (self *TestPeer) Id() string {
	return "testPeer"
}

type TestP2P struct {
	bestPeer *TestPeer
}

func (self *TestP2P) SetHandlerFn(fn p2p.MsgHandle) {
	self.bestPeer.fn = fn
}

func (self *TestP2P) BestPeer() (p2p.Peer, error) {
	return self.bestPeer, nil
}

func (self *TestP2P) AllPeer() ([]p2p.Peer, error) {
	return []p2p.Peer{self.bestPeer}, nil
}

func TestSyncer(t *testing.T) {
	peer := &TestPeer{}
	p := &TestP2P{}
	p.bestPeer = peer
	syncer := NewSyncer(p)
	fetcher := syncer.Fetcher()
	address := "viteshan"

	hashHeight := common.HashHeight{"5", 5}
	fetcher.FetchAccount(address, hashHeight, 5)

	time.Sleep(2 * time.Second)
}
