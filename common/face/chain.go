package face

import "github.com/viteshan/naive-vite/common"

type SnapshotChainReader interface {
	GetSnapshotBlocksByHashH(hashH common.HashHeight) *common.SnapshotBlock
}
type AccountChainReader interface {
	GetAccountBlocksByHashH(address string, hashH common.HashHeight) *common.AccountStateBlock
}

type ChainRw interface {
	GetSnapshotBlocksByHashH(hashH common.HashHeight) *common.SnapshotBlock
	GetAccountBlocksByHashH(address string, hashH common.HashHeight) *common.AccountStateBlock
	AddSnapshotBlock(block *common.SnapshotBlock)
	AddAccountBlock(account string, block *common.AccountStateBlock) error
}
