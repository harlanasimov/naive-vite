package syncer

import (
	"encoding/json"

	"github.com/viteshan/naive-vite/common"
	"github.com/viteshan/naive-vite/common/log"
	"github.com/viteshan/naive-vite/p2p"
)

func split(buf []common.HashHeight, lim int) [][]common.HashHeight {
	var chunk []common.HashHeight
	chunks := make([][]common.HashHeight, 0, len(buf)/lim+1)
	for len(buf) >= lim {
		chunk, buf = buf[:lim], buf[lim:]
		chunks = append(chunks, chunk)
	}
	if len(buf) > 0 {
		chunks = append(chunks, buf[:len(buf)])
	}
	return chunks
}

type reqAccountHashHandler struct {
	MsgHandler
	aReader *chainRw
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
		if i < 0 {
			break
		}
		block := self.aReader.GetAccountByHashH(msg.Address, hashH)
		if block == nil {
			break
		}
		hashes = append(hashes, hashH)
		hashH = common.HashHeight{Hash: block.PreHash(), Height: block.Height() - 1}
	}

	if len(hashes) == 0 {
		return
	}
	log.Info("send account hashes, address:%s, hashSize:%d, PId:%s", msg.Address, len(hashes), p.Id())
	m := split(hashes, 1000)

	for _, m1 := range m {
		self.sender.SendAccountHashes(msg.Address, m1, p)
		log.Info("send account hashes, address:%s, hashSize:%d, PId:%s", msg.Address, len(m1), p.Id())
	}
}

func (self *reqAccountHashHandler) Id() string {
	return "default-request-account-hash-handler"
}

type reqSnapshotHashHandler struct {
	MsgHandler
	sReader *chainRw
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
		if i < 0 {
			break
		}
		block := self.sReader.GetSnapshotByHashH(hashH)
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
	aReader *chainRw
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
		block := self.aReader.GetAccountByHashH(msg.Address, v)
		if block == nil {
			continue
		}
		blocks = append(blocks, block)
	}
	if len(blocks) > 0 {
		log.Info("send account blocks, address:%s, blockSize:%d, PId:%s", msg.Address, len(blocks), p.Id())
		self.sender.SendAccountBlocks(msg.Address, blocks, p)
	}
}

func (*reqAccountBlocksHandler) Id() string {
	return "default-request-account-blocks-handler"
}

type reqSnapshotBlocksHandler struct {
	MsgHandler
	sReader *chainRw
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
		block := self.sReader.GetSnapshotByHashH(v)
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
