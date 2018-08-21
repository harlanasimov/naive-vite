package face

import "github.com/viteshan/naive-vite/common"

type SnapshotChainReader interface {
	GetSnapshotBlocksByHashH(hashH common.HashHeight) *common.SnapshotBlock
}
type AccountChainReader interface {
	GetAccountBlocksByHashH(address string, hashH common.HashHeight) *common.AccountStateBlock
}
