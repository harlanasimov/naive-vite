package pool

import (
	"github.com/viteshan/naive-vite/common"
	"github.com/viteshan/naive-vite/common/log"
	"github.com/viteshan/naive-vite/syncer"
	"github.com/viteshan/naive-vite/verifier"
	"sync"
	"time"
)

type accountReader interface {
	FindBlockAboveSnapshotHeight(height int) *common.AccountStateBlock
}

type AccountPool struct {
	BCPool
	accountReader accountReader
	mu            sync.Locker
}

func NewAccountPool(name string) *AccountPool {
	pool := &AccountPool{}
	pool.Id = name
	return pool
}

func (self *AccountPool) Init(insertChainFn insertChainForkCheck,
	removeChainFn removeChainForkCheck,
	verifier verifier.Verifier,
	syncer syncer.Syncer,
	reader ChainReader,
	mu sync.Locker,
	accountReader accountReader) {
	self.mu = mu
	self.accountReader = accountReader
	self.BCPool.init(insertChainFn, removeChainFn, verifier, syncer, reader)
}

// 1. must be in diskchain
func (self *AccountPool) TryRollback(rollbackHeight int, rollbackHash string) ([]*common.AccountStateBlock, error) {
	{ // check logic
		w := self.chainpool.diskChain.getBlock(rollbackHeight, false)
		if w == nil || w.block.Hash() != rollbackHash {
			return nil, common.StrError{"error rollback cmd."}
		}
	}

	head := self.chainpool.diskChain.Head()

	var sendBlocks []*common.AccountStateBlock

	headHeight := head.Height()
	for i := headHeight; i > rollbackHeight; i-- {
		w := self.chainpool.diskChain.getBlock(i, false)
		if w == nil {
			continue
		}
		block := w.block.(*common.AccountStateBlock)
		if block.BlockType == common.SEND {
			sendBlocks = append(sendBlocks, block)
		}
	}
	return sendBlocks, nil
}

//func (self *AccountPool) RollbackAndForkAccount(target *common.AccountHashH, forkPoint *common.SnapshotBlock) error {
//	err := self.rollbackDisk(forkPoint.Height(), forkPoint.Hash())
//	if err != nil {
//		return err
//	}
//	if target == nil {
//		return nil
//	}
//	return self.currentModify(target.Height, target.Hash)
//}
func (self *AccountPool) ForkAccount(target *common.AccountHashH) error {

	//return self.currentModify(target)
	return nil
}

// rollback to current
//func (self *AccountPool) rollbackDisk(snapshotHeight int, snapshotHash string) error {
//	head := self.chainpool.diskChain.Head().(*common.AccountStateBlock)
//	if head.SnapshotHeight < snapshotHeight {
//		return nil
//	}
//
//	accountBlock := self.accountReader.FindBlockAboveSnapshotHeight(snapshotHeight)
//	var err error
//	if accountBlock == nil {
//		err = self.RollbackAll()
//	} else {
//		err = self.Rollback(accountBlock)
//	}
//	if err != nil {
//		return err
//	}
//	return nil
//}

// rollback to current
func (self *AccountPool) FindRollbackPointByReferSnapshot(snapshotHeight int, snapshotHash string) (bool, *common.AccountStateBlock, error) {
	head := self.chainpool.diskChain.Head().(*common.AccountStateBlock)
	if head.SnapshotHeight < snapshotHeight {
		return false, nil, nil
	}

	accountBlock := self.accountReader.FindBlockAboveSnapshotHeight(snapshotHeight)
	if accountBlock == nil {
		return true, nil, nil
	} else {
		return true, accountBlock, nil
	}
}

func (self *AccountPool) FindRollbackPointForAccountHashH(height int, hash string) (bool, *common.AccountStateBlock, Chain, error) {
	chain := self.whichChain(height, hash)
	if chain == nil {
		return false, nil, nil, nil
	}
	if chain.id() == self.chainpool.current.id() {
		return false, nil, nil, nil
	}
	_, forkPoint, err := self.getForkPointByChains(chain, self.chainpool.current)
	if err != nil {
		return false, nil, nil, err
	}
	return true, forkPoint.(*common.AccountStateBlock), chain, nil
}

func (self *AccountPool) loop() {
	for {
		self.LoopGenSnippetChains()
		self.LoopAppendChains()
		self.LoopFetchForSnippets()
		self.loopCheckCurrentInsert()
		time.Sleep(time.Second)
	}
}

func (self *AccountPool) loopCheckCurrentInsert() {
	self.mu.Lock()
	defer self.mu.Unlock()
	self.CheckCurrentInsert(self.insertAccountFailCallback, self.insertAccountSuccessCallback)
}
func (self *AccountPool) Start() {
	go self.loop()
}

func (self *AccountPool) insertAccountFailCallback(b common.Block, s verifier.BlockVerifyStat) {
	log.Info("do nothing. height:%d, hash:%s, pool:%s", b.Height(), b.Hash(), self.Id)
}

func (self *AccountPool) insertAccountSuccessCallback(b common.Block, s verifier.BlockVerifyStat) {
	log.Info("do nothing. height:%d, hash:%s, pool:%s", b.Height(), b.Hash(), self.Id)
}
