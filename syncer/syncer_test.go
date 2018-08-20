package syncer

import (
	"strconv"
	"testing"
	"time"

	"encoding/json"

	"github.com/viteshan/naive-vite/common"
	"github.com/viteshan/naive-vite/common/log"
	"github.com/viteshan/naive-vite/p2p"
)

type TestPeer struct {
	fn    p2p.MsgHandle
	state interface{}
}

func (self *TestPeer) SetState(state interface{}) {
	self.state = state
}

func (self *TestPeer) GetState() interface{} {
	return self.state
}

func (self *TestPeer) Write(msg *p2p.Msg) error {
	log.Info("write msg, msgType:%s", msg.Type())
	self.fn(msg.Type(), msg.Data(), self)
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

type TestAccountReader struct {
}

func (self *TestAccountReader) getBlocksByHeightHash(address string, hashH common.HashHeight) *common.AccountStateBlock {
	log.Info("TestAccountReader#getBlocksByHeightHash, address:%s, hash:%s, height:%d", address, hashH.Hash, hashH.Height)
	return genAccountBlock(address, hashH)
}

type TestSnapshotReader struct {
}

func (self *TestSnapshotReader) getBlocksByHeightHash(hashH common.HashHeight) *common.SnapshotBlock {
	log.Info("TestSnapshotReader#getBlocksByHeightHash, hash:%s, height:%d", hashH.Hash, hashH.Height)
	return genSnapshotBlock(hashH)
}

func TestSyncer(t *testing.T) {
	peer := &TestPeer{}
	p := &TestP2P{}
	snapshotReader := &TestSnapshotReader{}
	accountReader := &TestAccountReader{}
	p.bestPeer = peer
	syncer := NewSyncer(p, accountReader, snapshotReader)
	fetcher := syncer.Fetcher()
	address := "viteshan"
	testHandler := &TestHandler{}
	syncer.Handlers().RegisterHandler(testHandler)

	peer.fn = syncer.DefaultHandler().Handle

	var hashHeight common.HashHeight
	hashHeight = genHashHeight(5)
	fetcher.FetchAccount(address, hashHeight, 5)
	hashHeight = genHashHeight(6)
	fetcher.FetchAccount(address, hashHeight, 5)

	time.Sleep(2 * time.Second)
	if testHandler.cnt != 6 {
		t.Error("error number.", testHandler.cnt)
	}
}

type TestHandler struct {
	cnt int
}

func (self *TestHandler) Handle(t common.NetMsgType, msg []byte, peer p2p.Peer) {
	if t == common.SnapshotBlocks {
		hashesMsg := &snapshotBlocksMsg{}
		err := json.Unmarshal(msg, hashesMsg)
		if err != nil {
			log.Error("TestHandler.Handle unmarshal fail.")
		}
		self.cnt = self.cnt + len(hashesMsg.Blocks)
	} else if t == common.AccountBlocks {
		hashesMsg := &accountBlocksMsg{}
		err := json.Unmarshal(msg, hashesMsg)
		if err != nil {
			log.Error("TestHandler.Handle unmarshal fail.")
		}
		self.cnt = self.cnt + len(hashesMsg.Blocks)
	}
}

func (self *TestHandler) Types() []common.NetMsgType {
	return []common.NetMsgType{common.SnapshotBlocks, common.AccountBlocks}
}

func (self *TestHandler) Id() string {
	return "testHandler"
}

func genHashHeight(height int) common.HashHeight {
	return common.HashHeight{Hash: strconv.Itoa(height), Height: height}
}

func genSnapshotBlock(hashH common.HashHeight) *common.SnapshotBlock {
	preHashH := genHashHeight(hashH.Height - 1)
	return common.NewSnapshotBlock(hashH.Height, hashH.Hash, preHashH.Hash, "viteshan", time.Now(), nil)
}
func genAccountBlock(address string, hashH common.HashHeight) *common.AccountStateBlock {
	preHashH := genHashHeight(hashH.Height - 1)
	return common.NewAccountBlock(hashH.Height, hashH.Hash, preHashH.Hash, address, time.Now(), 0, 0, 0, "0", common.SEND, address, "viteshan2", "")
}
