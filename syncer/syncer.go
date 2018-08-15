package syncer

import (
	"github.com/viteshan/naive-vite/common"
	"github.com/viteshan/naive-vite/p2p"
)

type BlockHash struct {
	Height int
	Hash   string
}

type Syncer interface {
	Fetch(hash common.HashHeight, prevCnt int)
}

type Fetcher interface {
	FetchAccount(address string, hash common.HashHeight, prevCnt int)
	FetchSnapshot(hash common.HashHeight, prevCnt int)
}

type Sender interface {
	// when new block create
	BroadcastAccountBlocks(string, []*common.AccountStateBlock) error
	BroadcastSnapshotBlocks([]*common.SnapshotBlock) error

	// when fetch block message be arrived
	SendAccountBlocks(string, []*common.AccountStateBlock, p2p.Peer) error
	SendSnapshotBlocks([]*common.SnapshotBlock, p2p.Peer) error

	sendAccountHashes(string, []common.HashHeight, p2p.Peer) error
	sendSnapshotHashes([]common.HashHeight, p2p.Peer) error

	requestAccountHash(string, common.HashHeight, int) error
	requestSnapshotHash(common.HashHeight, int) error
	requestAccountBlocks(string, []common.HashHeight) error
	requestSnapshotBlocks([]common.HashHeight) error
}
type MsgHandler interface {
	Handle(p2p.NetMsgType, interface{}, p2p.Peer)
	Id() string
}

type Handlers interface {
	RegisterHandler(p2p.NetMsgType, MsgHandler)
	UnRegisterHandler(p2p.NetMsgType, MsgHandler)
}
