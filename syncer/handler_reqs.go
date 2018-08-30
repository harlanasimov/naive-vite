package syncer

import (
	"encoding/json"

	"github.com/viteshan/naive-vite/common"
	"github.com/viteshan/naive-vite/common/face"
	"github.com/viteshan/naive-vite/common/log"
	"github.com/viteshan/naive-vite/p2p"
)

type reqAccountHashHandler struct {
	MsgHandler
	aReader face.ChainRw
	sender  Sender
}

func (self *reqAccountHashHandler) Types() []common.NetMsgType {
	return []common.NetMsgType{common.RequestAccountHash}
}

func (self *reqAccountHashHandler) Handle(t common.NetMsgType, d []byte, p p2p.Peer) {
	msg := &requestAccountHashMsg{}
	err := json.Unmarshal(d, msg)
	if err != nil {
		log.Error("[reqAccountHashHandler]Unmarshal fail.")
		return
	}
	var hashes []common.HashHeight
	hashH := common.HashHeight{Hash: msg.Hash, Height: msg.Height}

	for i := msg.PrevCnt; i > 0; i-- {
		block := self.aReader.GetAccountBlocksByHashH(msg.Address, hashH)
		if block == nil {
			break
		}
		hashes = append(hashes, hashH)
		hashH = common.HashHeight{Hash: block.PreHash(), Height: block.Height() - 1}
	}

	if len(hashes) == 0 {
		return
	}
	self.sender.SendAccountHashes(msg.Address, hashes, p)
}

func (self *reqAccountHashHandler) Id() string {
	return "default-request-account-hash-handler"
}

type reqSnapshotHashHandler struct {
	MsgHandler
	sReader face.ChainRw
	sender  Sender
}

func (self *reqSnapshotHashHandler) Types() []common.NetMsgType {
	return []common.NetMsgType{common.RequestSnapshotHash}
}

func (self *reqSnapshotHashHandler) Handle(t common.NetMsgType, d []byte, p p2p.Peer) {
	msg := &requestSnapshotHashMsg{}
	err := json.Unmarshal(d, msg)
	if err != nil {
		log.Error("[reqSnapshotHashHandler]Unmarshal fail.")
		return
	}

	var hashes []common.HashHeight
	hashH := common.HashHeight{Hash: msg.Hash, Height: msg.Height}

	for i := msg.PrevCnt; i > 0; i-- {
		block := self.sReader.GetSnapshotBlocksByHashH(hashH)
		if block == nil {
			break
		}
		hashes = append(hashes, hashH)
		hashH = common.HashHeight{Hash: block.PreHash(), Height: block.Height() - 1}
	}

	if len(hashes) == 0 {
		return
	}
	self.sender.SendSnapshotHashes(hashes, p)
}

func (self *reqSnapshotHashHandler) Id() string {
	return "default-request-snapshot-hash-handler"
}

type reqAccountBlocksHandler struct {
	MsgHandler
	aReader face.ChainRw
	sender  Sender
}

func (self *reqAccountBlocksHandler) Types() []common.NetMsgType {
	return []common.NetMsgType{common.RequestAccountBlocks}
}

func (self *reqAccountBlocksHandler) Handle(t common.NetMsgType, d []byte, p p2p.Peer) {
	msg := &requestAccountBlockMsg{}
	err := json.Unmarshal(d, msg)
	if err != nil {
		log.Error("[reqAccountBlocksHandler]Unmarshal fail.")
	}

	hashes := msg.Hashes
	if len(hashes) <= 0 {
		return
	}
	var blocks []*common.AccountStateBlock
	for _, v := range hashes {
		block := self.aReader.GetAccountBlocksByHashH(msg.Address, v)
		if block == nil {
			continue
		}
		blocks = append(blocks, block)
	}
	if len(blocks) > 0 {
		self.sender.SendAccountBlocks(msg.Address, blocks, p)
	}
}

func (*reqAccountBlocksHandler) Id() string {
	return "default-request-account-blocks-handler"
}

type reqSnapshotBlocksHandler struct {
	MsgHandler
	sReader face.ChainRw
	sender  Sender
}

func (self *reqSnapshotBlocksHandler) Types() []common.NetMsgType {
	return []common.NetMsgType{common.RequestSnapshotBlocks}
}

func (self *reqSnapshotBlocksHandler) Handle(t common.NetMsgType, d []byte, p p2p.Peer) {
	msg := &requestSnapshotBlockMsg{}
	err := json.Unmarshal(d, msg)
	if err != nil {
		log.Error("[reqSnapshotBlocksHandler]Unmarshal fail.")
	}

	hashes := msg.Hashes
	if len(hashes) <= 0 {
		return
	}
	var blocks []*common.SnapshotBlock
	for _, v := range hashes {
		block := self.sReader.GetSnapshotBlocksByHashH(v)
		if block == nil {
			continue
		}
		blocks = append(blocks, block)
	}
	if len(blocks) > 0 {
		self.sender.SendSnapshotBlocks(blocks, p)
	}
}

func (*reqSnapshotBlocksHandler) Id() string {
	return "default-request-snapshot-blocks-handler"
}
