package syncer

import (
	"encoding/json"

	"github.com/viteshan/naive-vite/common"
	"github.com/viteshan/naive-vite/common/log"
	"github.com/viteshan/naive-vite/p2p"
)

type reqAccountHashHandler struct {
}

func (self *reqAccountHashHandler) Handle(t common.NetMsgType, d []byte, p p2p.Peer) {
	msg := &requestAccountHashMsg{}
	err := json.Unmarshal(d, msg)
	if err != nil {
		log.Error("[reqAccountHashHandler]Unmarshal fail.")
	}
	panic("implement me")
}

func (self *reqAccountHashHandler) Id() string {
	return "default-request-account-hash-handler"
}

type reqSnapshotHashHandler struct {
}

func (self *reqSnapshotHashHandler) Handle(t common.NetMsgType, d []byte, p p2p.Peer) {
	msg := &requestSnapshotHashMsg{}
	err := json.Unmarshal(d, msg)
	if err != nil {
		log.Error("[reqSnapshotHashHandler]Unmarshal fail.")
	}
}

func (self *reqSnapshotHashHandler) Id() string {
	return "default-request-snapshot-hash-handler"
}

type reqAccountBlocksHandler struct {
}

func (*reqAccountBlocksHandler) Handle(t common.NetMsgType, d []byte, p p2p.Peer) {
	msg := &requestAccountBlockMsg{}
	err := json.Unmarshal(d, msg)
	if err != nil {
		log.Error("[reqAccountBlocksHandler]Unmarshal fail.")
	}
	panic("implement me")
}

func (*reqAccountBlocksHandler) Id() string {
	return "default-request-account-blocks-handler"
}

type reqSnapshotBlocksHandler struct {
}

func (*reqSnapshotBlocksHandler) Handle(t common.NetMsgType, d []byte, p p2p.Peer) {
	msg := &requestSnapshotBlockMsg{}
	err := json.Unmarshal(d, msg)
	if err != nil {
		log.Error("[reqSnapshotBlocksHandler]Unmarshal fail.")
	}
	panic("implement me")
}

func (*reqSnapshotBlocksHandler) Id() string {
	return "default-request-snapshot-blocks-handler"
}
