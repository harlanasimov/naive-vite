package consensus

import "github.com/viteshan/naive-vite/common"

type AccountsConsensus interface {
	ForkAccounts(keyPoint *common.SnapshotBlock, forkPoint *common.SnapshotBlock) error
	ForkAccountTo(h *common.AccountHashH) error
}