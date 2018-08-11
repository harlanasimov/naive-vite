package syncer

import "github.com/viteshan/naive-vite/common"

type Sender interface {
	// when new block create
	BroadcastAccountBlocks([]*common.AccountStateBlock) error
	BroadcastSnapshotBlocks([]*common.SnapshotBlock) error

	// when fetch block message be arrived
	SendAccountBlocks([]*common.AccountStateBlock) error
	SendSnapshotBlocks([]*common.SnapshotBlock) error
}

