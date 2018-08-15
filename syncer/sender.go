package syncer

import (
	"encoding/json"

	"github.com/pkg/errors"
	"github.com/viteshan/naive-vite/common"
	"github.com/viteshan/naive-vite/common/log"
	"github.com/viteshan/naive-vite/p2p"
)

type sender struct {
	net p2p.P2P
}

func (self *sender) BroadcastAccountBlocks(address string, blocks []*common.AccountStateBlock) error {
	bytM, err := json.Marshal(&accountBlocksMsg{address: address, blocks: blocks})
	msg := p2p.NewMsg(common.AccountBlocks, bytM)

	if err == nil {
		return errors.New("BroadcastAccountBlocks, format fail. err:" + err.Error())
	}
	peers, err := self.net.AllPeer()
	if err != nil {
		log.Error("BroadcastAccountBlocks, can't get all peer.%v", err)
		return err
	}

	for _, p := range peers {
		tmpE := p.Write(msg)
		if tmpE != nil {
			err = tmpE
			log.Error("BroadcastAccountBlocks, write data fail, peerId:%s, err:%v", p.Id(), err)
		}
	}
	return err
}

func (self *sender) BroadcastSnapshotBlocks(blocks []*common.SnapshotBlock) error {
	bytM, err := json.Marshal(&snapshotBlocksMsg{blocks: blocks})
	msg := p2p.NewMsg(common.SnapshotBlocks, bytM)

	if err == nil {
		return errors.New("BroadcastSnapshotBlocks, format fail. err:" + err.Error())
	}
	peers, err := self.net.AllPeer()
	if err != nil {
		log.Error("BroadcastSnapshotBlocks, can't get all peer.%v", err)
		return err
	}

	for _, p := range peers {
		tmpE := p.Write(msg)
		if tmpE != nil {
			err = tmpE
			log.Error("BroadcastSnapshotBlocks, write data fail, peerId:%s, err:%v", p.Id(), err)
		}
	}
	return err
}

func (self *sender) SendAccountBlocks(address string, blocks []*common.AccountStateBlock, peer p2p.Peer) error {
	bytM, err := json.Marshal(&accountBlocksMsg{address: address, blocks: blocks})
	if err == nil {
		return errors.New("SendAccountBlocks, format fail. err:" + err.Error())
	}
	msg := p2p.NewMsg(common.AccountBlocks, bytM)
	err = peer.Write(msg)
	if err != nil {
		log.Error("SendAccountBlocks, write peer fail. peer:%s, err:%v", peer.Id(), err)
	}
	return err
}

func (self *sender) SendSnapshotBlocks(blocks []*common.SnapshotBlock, peer p2p.Peer) error {
	bytM, err := json.Marshal(&snapshotBlocksMsg{blocks: blocks})
	if err == nil {
		return errors.New("SendSnapshotBlocks, format fail. err:" + err.Error())
	}
	msg := p2p.NewMsg(common.SnapshotBlocks, bytM)
	err = peer.Write(msg)
	if err != nil {
		log.Error("SendSnapshotBlocks, write peer fail. peer:%s, err:%v", peer.Id(), err)
	}
	return err

}

func (self *sender) sendAccountHashes(address string, hashes []common.HashHeight, peer p2p.Peer) error {
	bytM, err := json.Marshal(&accountHashesMsg{address: address, hashes: hashes})
	if err == nil {
		return errors.New("sendAccountHashes, format fail. err:" + err.Error())
	}
	msg := p2p.NewMsg(common.AccountHashes, bytM)
	err = peer.Write(msg)
	if err != nil {
		log.Error("sendAccountHashes, write peer fail. peer:%s, err:%v", peer.Id(), err)
	}
	return err
}

func (self *sender) sendSnapshotHashes(hashes []common.HashHeight, peer p2p.Peer) error {
	bytM, err := json.Marshal(&snapshotHashesMsg{hashes: hashes})
	if err == nil {
		return errors.New("sendSnapshotHashes, format fail. err:" + err.Error())
	}
	msg := p2p.NewMsg(common.SnapshotHashes, bytM)
	err = peer.Write(msg)
	if err != nil {
		log.Error("sendSnapshotHashes, write peer fail. peer:%s, err:%v", peer.Id(), err)
	}
	return err

}

func (self *sender) requestAccountHash(address string, height common.HashHeight, prevCnt int) error {
	peer, e := self.net.BestPeer()
	if e != nil {
		log.Error("sendAccountHash, can't get best peer. err:%v", e)
		return e
	}
	m := requestAccountHashMsg{address: address, height: height.Height, hash: height.Hash, prevCnt: prevCnt}
	bytM, err := json.Marshal(&m)
	if err == nil {
		return errors.New("sendAccountHash, format fail. err:" + err.Error())
	}
	msg := p2p.NewMsg(common.RequestAccountHash, bytM)
	err = peer.Write(msg)
	if err != nil {
		log.Error("sendAccountHash, write peer fail. peer:%s, err:%v", peer.Id(), err)
	}
	return err
}

func (self *sender) requestSnapshotHash(height common.HashHeight, prevCnt int) error {
	peer, e := self.net.BestPeer()
	if e != nil {
		log.Error("sendSnapshotHash, can't get best peer. err:%v", e)
		return e
	}
	m := requestSnapshotHashMsg{height: height.Height, hash: height.Hash, prevCnt: prevCnt}
	bytM, err := json.Marshal(&m)
	if err == nil {
		return errors.New("sendSnapshotHash, format fail. err:" + err.Error())
	}
	msg := p2p.NewMsg(common.RequestSnapshotHash, bytM)
	err = peer.Write(msg)
	if err != nil {
		log.Error("sendSnapshotHash, write peer fail. peer:%s, err:%v", peer.Id(), err)
	}
	return err
}

func (self *sender) requestAccountBlocks(address string, hashes []common.HashHeight) error {
	peer, e := self.net.BestPeer()
	if e != nil {
		log.Error("requestAccountBlocks, can't get best peer. err:%v", e)
		return e
	}
	m := requestAccountBlockMsg{address: address, hashes: hashes}
	bytM, err := json.Marshal(&m)
	if err == nil {
		return errors.New("requestAccountBlocks, format fail. err:" + err.Error())
	}
	msg := p2p.NewMsg(common.RequestAccountBlocks, bytM)
	err = peer.Write(msg)
	if err != nil {
		log.Error("requestAccountBlocks, write peer fail. peer:%s, err:%v", peer.Id(), err)
	}
	return err
}
func (self *sender) requestSnapshotBlocks(hashes []common.HashHeight) error {
	peer, e := self.net.BestPeer()
	if e != nil {
		log.Error("requestSnapshotBlocks, can't get best peer. err:%v", e)
		return e
	}
	m := requestSnapshotBlockMsg{hashes: hashes}
	bytM, err := json.Marshal(&m)
	if err == nil {
		return errors.New("requestSnapshotBlocks, format fail. err:" + err.Error())
	}
	msg := p2p.NewMsg(common.RequestSnapshotBlocks, bytM)
	err = peer.Write(msg)
	if err != nil {
		log.Error("requestSnapshotBlocks, write peer fail. peer:%s, err:%v", peer.Id(), err)
	}
	return err
}
