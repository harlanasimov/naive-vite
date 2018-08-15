package syncer

import (
	"github.com/viteshan/naive-vite/common/log"
	"github.com/viteshan/naive-vite/p2p"
)

type receiver struct {
	fetcher       *fetcher
	innerHandlers map[p2p.NetMsgType]MsgHandler
	handlers      map[p2p.NetMsgType]MsgHandler
}

func NewReceiver(fetcher *fetcher) *receiver {
	self := &receiver{}
	self.fetcher = fetcher
	self.innerHandlers = make(map[p2p.NetMsgType]MsgHandler)
	snapshotHashHandler := &SnapshotHashHandler{fetcher: fetcher}
	accountHashHandler := &AccountHashHandler{fetcher: fetcher}
	blocksHandler := &BlocksHandler{fetcher: fetcher}
	self.innerHandlers[p2p.AccountHashes] = accountHashHandler
	self.innerHandlers[p2p.SnapshotHashes] = snapshotHashHandler
	self.innerHandlers[p2p.SnapshotBlocks] = blocksHandler
	self.innerHandlers[p2p.AccountBlocks] = blocksHandler
	self.handlers = make(map[p2p.NetMsgType]MsgHandler)
	return self
}

type SnapshotHashHandler struct {
	fetcher *fetcher
}

func (self *SnapshotHashHandler) Id() string {
	return "default-snapshotHashHandler"
}

func (self *SnapshotHashHandler) Handle(t p2p.NetMsgType, msg interface{}, peer p2p.Peer) {
	hashesMsg := msg.(snapshotHashesMsg)
	self.fetcher.fetchSnapshotBlockByHash(hashesMsg.hashes)
}

type AccountHashHandler struct {
	fetcher *fetcher
}

func (self *AccountHashHandler) Handle(t p2p.NetMsgType, msg interface{}, peer p2p.Peer) {
	hashesMsg := msg.(accountHashesMsg)
	self.fetcher.fetchAccountBlockByHash(hashesMsg.address, hashesMsg.hashes)
}
func (self *AccountHashHandler) Id() string {
	return "default-accountHashHandler"
}

type BlocksHandler struct {
	fetcher *fetcher
}

func (self *BlocksHandler) Handle(t p2p.NetMsgType, msg interface{}, peer p2p.Peer) {
	hashesMsg := msg.(accountHashesMsg)
	self.fetcher.fetchAccountBlockByHash(hashesMsg.address, hashesMsg.hashes)
}

func (self *BlocksHandler) Id() string {
	return "default-blocksHandler"
}

func (self *receiver) Handle(t p2p.NetMsgType, msg interface{}, peer p2p.Peer) {
	handler := self.innerHandlers[t]
	if handler != nil {
		handler.Handle(t, msg, peer)
	}

	handler = self.handlers[t]
	if handler != nil {
		handler.Handle(t, msg, peer)
	}
}
func (self *receiver) RegisterHandler(t p2p.NetMsgType, handler MsgHandler) {
	self.handlers[t] = handler
	log.Info("register msg handler, type:%s, handler:%s", t, handler.Id())
}

func (self *receiver) UnRegisterHandler(t p2p.NetMsgType, handler MsgHandler) {
	delete(self.handlers, t)
	log.Info("unregister msg handler, type:%s, handler:%s", t, handler.Id())
}
