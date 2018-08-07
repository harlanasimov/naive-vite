package ledger

import (
	"github.com/viteshan/naive-vite/common"
	"github.com/viteshan/naive-vite/common/log"
	"github.com/viteshan/naive-vite/ledger/pool"
	"github.com/viteshan/naive-vite/tools"
	"time"
)

// account block chain
type AccountChain struct {
	address         string
	head            *common.AccountStateBlock
	accountDB       map[string]*common.AccountStateBlock
	accountHeightDB map[int]*common.AccountStateBlock
	pending         *pool.AccountPool

	reqPool *reqPool
}

var blank = common.NewAccountBlock(-1, "", "", "", time.Unix(1533550870, 0),
	0, 0, GetGenesisSnapshot().Height(), GetGenesisSnapshot().Hash(), common.CREATE, "", "", "")

func NewAccountChain(address string, snapshotHeight int, snapshotHash string) *AccountChain {
	self := &AccountChain{}
	self.address = address
	self.head = common.NewAccountBlock(0, "", "", address, time.Now(),
		0, 0, snapshotHeight, snapshotHash, common.CREATE, address, address, "")
	self.head.SetHash(tools.CalculateAccountHash(self.head))
	self.accountDB = make(map[string]*common.AccountStateBlock)
	self.accountHeightDB = make(map[int]*common.AccountStateBlock)
	return self
}

func (self *AccountChain) SetPending(pool *pool.AccountPool) {
	self.pending = pool
}

func (self *AccountChain) Head() common.Block {
	if self.head == nil {

	}
	return self.head
}

func (self *AccountChain) GetBlock(height int) common.Block {
	if height < 0 {
		log.Error("can't request height 0 block. account:%s", self.address)
		return nil
	}
	return self.accountHeightDB[height]
}

func (self *AccountChain) insertChain(b common.Block, forkVersion int) (bool, error) {
	log.Info("insert to account Chain: %s", b)
	block := b.(*common.AccountStateBlock)
	self.accountDB[block.Hash()] = block
	self.accountHeightDB[block.Height()] = block
	self.head = block
	self.reqPool.blockInsert(block)
	return true, nil
}
func (self *AccountChain) removeChain(b common.Block) (bool, error) {
	log.Info("remove from account Chain: %s", b)
	block := b.(*common.AccountStateBlock)

	head := self.accountDB[block.PreHash()]
	delete(self.accountDB, block.Hash())
	delete(self.accountHeightDB, block.Height())
	self.reqPool.blockRollback(block)
	if head == nil && block.PreHash() == "" && block.Height() == 0 {
		self.head = blank
	} else {
		self.head = head
	}
	return true, nil
}

func (self *AccountChain) FindBlockAboveSnapshotHeight(snapshotHeight int) *common.AccountStateBlock {
	for i := self.head.Height(); i >= 0; i-- {
		block := self.accountHeightDB[i]
		if block.SnapshotHeight <= snapshotHeight {
			return block
		}
	}
	return nil
}
func (self *AccountChain) GetBySourceBlock(sourceHash string) *common.AccountStateBlock {
	height := self.head.Height()
	for i := height; i > 0; i-- {
		// first block(i==0) is create block
		v := self.accountHeightDB[i]
		if v.SourceHash == sourceHash {
			return v
		}
	}
	return nil
}
