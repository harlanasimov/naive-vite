package ledger

import (
	"github.com/viteshan/naive-vite/common"
	"github.com/viteshan/naive-vite/common/log"
	"github.com/viteshan/naive-vite/ledger/pool"
)

// account block chain
type AccountChain struct {
	head            *common.AccountStateBlock
	accountDB       map[string]*common.AccountStateBlock
	accountHeightDB map[int]*common.AccountStateBlock
	pending         *pool.AccountPool

	txpool *txpool
}

func (self *AccountChain) Head() common.Block {
	return self.head
}

func (self *AccountChain) GetBlock(height int) common.Block {
	return self.accountHeightDB[height]
}

func (self *AccountChain) insertChain(b common.Block, forkVersion int) (bool, error) {
	log.Info("insert to account Chain: %s", b)
	block := b.(*common.AccountStateBlock)
	self.accountDB[block.Hash()] = block
	self.accountHeightDB[block.Height()] = block
	self.head = block
	return true, nil
}
func (self *AccountChain) removeChain(b common.Block) (bool, error) {
	log.Info("remove from account Chain: %s", b)
	block := b.(*common.AccountStateBlock)

	head := self.accountDB[block.PreHash()]
	delete(self.accountDB, block.Hash())
	delete(self.accountHeightDB, block.Height())
	self.head = head
	return true, nil
}

func (self *AccountChain) FindBlockAboveSnapshotHeight(snapshotHeight int) *common.AccountStateBlock {
	// todo

	for i := self.head.Height(); i > 0; i-- {
		block := self.accountHeightDB[i]
		if block.SnapshotHeight <= snapshotHeight {
			return block
		}
	}
	return nil
}
func (self *AccountChain) GetBySourceBlock(sourceHash string) *common.AccountStateBlock {
	for _, v := range self.accountHeightDB {
		if v.SourceHash == sourceHash {
			return v
		}
	}
	return nil
}
