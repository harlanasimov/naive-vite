package syncer

import (
	"encoding/json"

	"github.com/prometheus/common/log"
	"github.com/vitelabs/go-vite/p2p"
	"github.com/viteshan/naive-vite/common"
)

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
	log.Info("send account hashes, address:%s, hashSize:%d, PId:%s, height:%d, prevCnt:%d", msg.Address, len(hashes), p.Id(), msg.Height, msg.PrevCnt)
	m := split(hashes, 1000)

	for _, m1 := range m {
		self.sender.SendAccountHashes(msg.Address, m1, p)
		log.Info("send account hashes, address:%s, hashSize:%d, PId:%s", msg.Address, len(m1), p.Id())
	}
}

func (self *reqAccountHashHandler) Id() string {
	return "default-request-account-hash-handler"
}
