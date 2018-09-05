package face

import "github.com/viteshan/naive-vite/common"

type ChainReader interface {
	SnapshotReader
	AccountReader
}
type SnapshotReader interface {
	GenesisSnapshot() (*common.SnapshotBlock, error)
	HeadSnapshot() (*common.SnapshotBlock, error)
	GetSnapshotByHashH(hashH common.HashHeight) *common.SnapshotBlock
	GetSnapshotByHash(hash string) *common.SnapshotBlock
	GetSnapshotByHeight(height int) *common.SnapshotBlock
	//ListSnapshotBlock(limit int) []*common.SnapshotBlock
}
type AccountReader interface {
	HeadAccount(address string) (*common.AccountStateBlock, error)
	GetAccountByHashH(address string, hashH common.HashHeight) *common.AccountStateBlock
	GetAccountByHash(address string, hash string) *common.AccountStateBlock
	GetAccountByHeight(address string, height int) *common.AccountStateBlock
	//ListAccountBlock(address string, limit int) []*common.AccountStateBlock

	GetAccountBySourceHash(address string, source string) *common.AccountStateBlock
	NextAccountSnapshot() (common.HashHeight, []*common.AccountHashH, error)
	FindAccountAboveSnapshotHeight(address string, snapshotHeight int) *common.AccountStateBlock
}

type SnapshotWriter interface {
	InsertSnapshotBlock(block *common.SnapshotBlock) error
	RemoveSnapshotHead(block *common.SnapshotBlock) error
}
type AccountWriter interface {
	InsertAccountBlock(address string, block *common.AccountStateBlock) error
	RemoveAccountHead(address string, block *common.AccountStateBlock) error
	RollbackSnapshotPoint(address string, start *common.SnapshotPoint, end *common.SnapshotPoint) error
}

type ChainListener interface {
	SnapshotInsertCallback(block *common.SnapshotBlock)
	SnapshotRemoveCallback(block *common.SnapshotBlock)
	AccountInsertCallback(address string, block *common.AccountStateBlock)
	AccountRemoveCallback(address string, block *common.AccountStateBlock)
}

type SyncStatus interface {
	Done() bool
}
