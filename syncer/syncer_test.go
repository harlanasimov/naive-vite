package syncer

import (
	"testing"

	"github.com/viteshan/naive-vite/common"
	"github.com/viteshan/naive-vite/p2p"
)

type TestP2P struct {
}

func (self *TestP2P) BestPeer() (p2p.Peer, error) {
	panic("implement me")
}

func (self *TestP2P) AllPeer() ([]p2p.Peer, error) {
	panic("implement me")
}

func TestSyncer(t *testing.T) {
	p := &TestP2P{}
	syncer := NewSyncer(p)
	fetcher := syncer.Fetcher()
	address := "viteshan"

	hashHeight := common.HashHeight{"5", 5}
	fetcher.FetchAccount(address, hashHeight, 5)
}
