package ledger

import (
	"github.com/viteshan/naive-vite/common"
	"github.com/viteshan/naive-vite/common/log"
	"github.com/viteshan/naive-vite/ledger/pool"
)

// snapshot block chain
type Snapshotchain struct {
	head             *common.SnapshotBlock
	snapshotDB       map[string]*common.SnapshotBlock
	snapshotHeightDB map[int]*common.SnapshotBlock
	pending          *pool.SnapshotPool
}

func newSnapshotChain() *Snapshotchain {
	chain := &Snapshotchain{}
	chain.snapshotDB = make(map[string]*common.SnapshotBlock)
	chain.snapshotHeightDB = make(map[int]*common.SnapshotBlock)
	return chain
}

func (self *Snapshotchain) Contains(height int, hash string) bool {
	block := self.GetBlock(height)
	if block == nil {
		return false
	}
	if block.Hash() != hash {
		return false
	}
	return true
}

func (self *Snapshotchain) Head() common.Block {
	return self.head
}

func (self *Snapshotchain) GetBlock(height int) common.Block {
	return self.snapshotHeightDB[height]
}

func (self *Snapshotchain) insertChain(b common.Block, forkVersion int) (bool, error) {
	log.Info("insert to snapshot Chain: %s", b)
	block := b.(*common.SnapshotBlock)
	self.snapshotDB[block.Hash()] = block
	self.snapshotHeightDB[block.Height()] = block
	self.head = block
	return true, nil
}
func (self *Snapshotchain) removeChain(b common.Block) (bool, error) {
	log.Info("remove from snapshot Chain: %s", b)
	block := b.(*common.SnapshotBlock)

	head := self.snapshotDB[block.PreHash()]
	delete(self.snapshotDB, block.Hash())
	delete(self.snapshotHeightDB, block.Height())
	self.head = head
	return true, nil
}
