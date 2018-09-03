package face

import "github.com/viteshan/naive-vite/common"

type ChainRw interface {
	GetSnapshotBlocksByHashH(hashH common.HashHeight) *common.SnapshotBlock
	GetAccountBlocksByHashH(address string, hashH common.HashHeight) *common.AccountStateBlock
	AddSnapshotBlock(block *common.SnapshotBlock)
	AddAccountBlock(account string, block *common.AccountStateBlock) error
	HeadSnapshost() (*common.SnapshotBlock, error)
	GenesisSnapshost() (*common.SnapshotBlock, error)
}

type SyncStatus interface {
	Done() bool
}
