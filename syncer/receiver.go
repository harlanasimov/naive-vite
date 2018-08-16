package syncer

import (
	"encoding/json"

	"github.com/viteshan/naive-vite/common"
	"github.com/viteshan/naive-vite/common/log"
	"github.com/viteshan/naive-vite/p2p"
)

type receiver struct {
	fetcher       *fetcher
	innerHandlers map[common.NetMsgType]MsgHandler
	handlers      map[common.NetMsgType]MsgHandler
}

func NewReceiver(fetcher *fetcher) *receiver {
	self := &receiver{}
	self.fetcher = fetcher
	self.innerHandlers = make(map[common.NetMsgType]MsgHandler)
	snapshotHashHandler := &snapshotHashHandler{fetcher: fetcher}
	accountHashHandler := &accountHashHandler{fetcher: fetcher}
	snapshotBlocksHandler := &snapshotBlocksHandler{fetcher: fetcher}
	accountBlocksHandler := &accountBlocksHandler{fetcher: fetcher}
	self.innerHandlers[common.AccountHashes] = accountHashHandler
	self.innerHandlers[common.SnapshotHashes] = snapshotHashHandler
	self.innerHandlers[common.SnapshotBlocks] = snapshotBlocksHandler
	self.innerHandlers[common.AccountBlocks] = accountBlocksHandler
	self.handlers = make(map[common.NetMsgType]MsgHandler)
	return self
}

type stateHandler struct {
}

func (self *stateHandler) Id() string {
	return "default-state-handler"
}

func (self *stateHandler) Handle(t common.NetMsgType, msg []byte, peer p2p.Peer) {
	stateMsg := &stateMsg{}

	err := json.Unmarshal(msg, stateMsg)
	if err != nil {
		log.Error("stateHandler.Handle unmarshal fail.")
		return
	}
	prevState := peer.GetState()
	if prevState == nil {
		peer.SetState(&peerState{height: stateMsg.Height})
	} else {
		state := prevState.(*peerState)
		state.height = stateMsg.Height
	}
}

type snapshotHashHandler struct {
	fetcher *fetcher
}

func (self *snapshotHashHandler) Id() string {
	return "default-snapshotHashHandler"
}

func (self *snapshotHashHandler) Handle(t common.NetMsgType, msg []byte, peer p2p.Peer) {
	hashesMsg := &snapshotHashesMsg{}

	err := json.Unmarshal(msg, hashesMsg)
	if err != nil {
		log.Error("snapshotHashHandler.Handle unmarshal fail.")
	}
	self.fetcher.fetchSnapshotBlockByHash(hashesMsg.Hashes)
}

type accountHashHandler struct {
	fetcher *fetcher
}

func (self *accountHashHandler) Handle(t common.NetMsgType, msg []byte, peer p2p.Peer) {
	hashesMsg := &accountHashesMsg{}
	err := json.Unmarshal(msg, hashesMsg)
	if err != nil {
		log.Error("accountHashHandler.Handle unmarshal fail.")
	}
	self.fetcher.fetchAccountBlockByHash(hashesMsg.Address, hashesMsg.Hashes)
}
func (self *accountHashHandler) Id() string {
	return "default-accountHashHandler"
}

type snapshotBlocksHandler struct {
	fetcher *fetcher
}

func (self *snapshotBlocksHandler) Handle(t common.NetMsgType, msg []byte, peer p2p.Peer) {
	hashesMsg := &snapshotBlocksMsg{}
	err := json.Unmarshal(msg, hashesMsg)
	if err != nil {
		log.Error("snapshotBlocksHandler.Handle unmarshal fail.")
	}
	for _, v := range hashesMsg.Blocks {
		self.fetcher.done(v.Hash(), v.Height())
	}
}
func (self *snapshotBlocksHandler) Id() string {
	return "default-snapshotBlocksHandler"
}

type accountBlocksHandler struct {
	fetcher *fetcher
}

func (self *accountBlocksHandler) Handle(t common.NetMsgType, msg []byte, peer p2p.Peer) {
	hashesMsg := &accountBlocksMsg{}
	err := json.Unmarshal(msg, hashesMsg)
	if err != nil {
		log.Error("accountBlocksHandler.Handle unmarshal fail.")
	}
	for _, v := range hashesMsg.blocks {
		self.fetcher.done(v.Hash(), v.Height())
	}
}

func (self *accountBlocksHandler) Id() string {
	return "default-accountBlocksHandler"
}

func (self *receiver) Handle(t common.NetMsgType, msg []byte, peer p2p.Peer) {
	handler := self.innerHandlers[t]
	if handler != nil {
		handler.Handle(t, msg, peer)
	}

	handler = self.handlers[t]
	if handler != nil {
		handler.Handle(t, msg, peer)
	}
}

func (self *receiver) RegisterHandler(t common.NetMsgType, handler MsgHandler) {
	self.handlers[t] = handler
	log.Info("register msg handler, type:%s, handler:%s", t, handler.Id())
}

func (self *receiver) UnRegisterHandler(t common.NetMsgType, handler MsgHandler) {
	delete(self.handlers, t)
	log.Info("unregister msg handler, type:%s, handler:%s", t, handler.Id())
}
