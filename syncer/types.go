package syncer

import "github.com/viteshan/naive-vite/common"

type accountBlocksMsg struct {
	address string
	blocks  []*common.AccountStateBlock
}
type snapshotBlocksMsg struct {
	blocks []*common.SnapshotBlock
}

type accountHashesMsg struct {
	address string
	hashes  []common.HashHeight
}
type snapshotHashesMsg struct {
	hashes []common.HashHeight
}

type requestAccountHashMsg struct {
	address string
	height  int
	hash    string
	prevCnt int
}

type requestSnapshotHashMsg struct {
	height  int
	hash    string
	prevCnt int
}

type requestAccountBlockMsg struct {
	address string
	hashes  []common.HashHeight
}

type requestSnapshotBlockMsg struct {
	hashes []common.HashHeight
}
