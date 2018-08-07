package consensus

import (
	"github.com/viteshan/naive-vite/common"
)

type SnapshotReader interface {
}

type SnapshotHeader struct {
	Timestamp uint64
	Producer  common.Address
}

type Verifier interface {
	Verify(reader SnapshotReader, block *common.SnapshotBlock) (bool, error)
}

type Seal interface {
	Seal() error
}

type AccountsConsensus interface {
	ForkAccounts(keyPoint *common.SnapshotBlock, forkPoint *common.SnapshotBlock) error
	ForkAccountTo(h *common.AccountHashH) error
}
