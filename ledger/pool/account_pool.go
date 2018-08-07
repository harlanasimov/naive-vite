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

func (self *AccountPool) RollbackAndForkAccount(target *common.AccountHashH, forkPoint *common.SnapshotBlock) error {
	err := self.rollbackDisk(forkPoint.Height(), forkPoint.Hash())
	if err != nil {
		return err
	}
	if target == nil {
		return nil
	}
	return self.currentModify(target)
}
func (self *AccountPool) ForkAccount(target *common.AccountHashH) error {
	return self.currentModify(target)
}

// rollback to current
func (self *AccountPool) rollbackDisk(height int, hash string) error {
	head := self.chainpool.diskChain.Head().(*common.AccountStateBlock)
	if head.SnapshotHeight < height {
		return nil
	}

	accountBlock := self.accountReader.FindBlockAboveSnapshotHeight(height)
	var err error
	if accountBlock == nil {
		err = self.RollbackAll()
	} else {
		err = self.Rollback(accountBlock)
	}
	if err != nil {
		return err
	}
	return nil
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
	self.CheckCurrentInsert(self.insertAccountFailCallback)
}
func (self *AccountPool) Start() {
	go self.loop()
}

func (self *AccountPool) insertAccountFailCallback(b common.Block, s verifier.BlockVerifyStat) {
	log.Info("do nothing. height:%d, hash:%s, pool:%s", b.Height(), b.Hash(), self.Id)
}
