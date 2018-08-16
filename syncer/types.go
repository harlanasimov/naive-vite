package syncer

import "github.com/viteshan/naive-vite/common"

type stateMsg struct {
	Height int
}

type accountBlocksMsg struct {
	Address string
	blocks  []*common.AccountStateBlock
}
type snapshotBlocksMsg struct {
	Blocks []*common.SnapshotBlock
}

type accountHashesMsg struct {
	Address string
	Hashes  []common.HashHeight
}
type snapshotHashesMsg struct {
	Hashes []common.HashHeight
}

type requestAccountHashMsg struct {
	Address string
	Height  int
	Hash    string
	PrevCnt int
}

type requestSnapshotHashMsg struct {
	Height  int
	Hash    string
	PrevCnt int
}

type requestAccountBlockMsg struct {
	Address string
	Hashes  []common.HashHeight
}

type requestSnapshotBlockMsg struct {
	Hashes []common.HashHeight
}

type peerState struct {
	height int
}
