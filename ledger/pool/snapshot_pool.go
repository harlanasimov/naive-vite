package pool

import (
	"sync"
	"time"

	"github.com/viteshan/naive-vite/common"
	"github.com/viteshan/naive-vite/common/log"
	"github.com/viteshan/naive-vite/consensus"
	"github.com/viteshan/naive-vite/verifier"
	"github.com/viteshan/naive-vite/version"
)

type SnapshotPool struct {
	BCPool
	rwMu      *sync.RWMutex
	consensus consensus.AccountsConsensus
	closed    chan struct{}
	wg        sync.WaitGroup
}

func NewSnapshotPool(name string) *SnapshotPool {
	pool := &SnapshotPool{}
	pool.Id = name
	return pool
}

func (self *SnapshotPool) Init(insertChainFn insertChainForkCheck,
	removeChainFn removeChainForkCheck,
	verifier verifier.Verifier,
	syncer *fetcher,
	reader ChainReader,
	rwMu *sync.RWMutex,
	accountsConsensus consensus.AccountsConsensus) {
	self.rwMu = rwMu
	self.consensus = accountsConsensus
	self.BCPool.init(insertChainFn, removeChainFn, verifier, syncer, reader)
}

func (self *SnapshotPool) loopCheckFork() {
	defer self.wg.Done()
	for {
		select {
		case <-self.closed:
			return
		default:
			self.checkFork()
			// check fork every 2 sec.
			time.Sleep(2 * time.Second)
		}
	}
}

func (self *SnapshotPool) checkFork() {
	longest := self.LongestChain()
	current := self.CurrentChain()
	if longest.ChainId() == current.ChainId() {
		return
	}
	self.snapshotFork(longest, current)

}

func (self *SnapshotPool) snapshotFork(longest Chain, current Chain) {
	log.Warn("[try]snapshot chain start fork.longest chain:%s, currentchain:%s", longest.ChainId(), current.ChainId())
	self.rwMu.Lock()
	defer self.rwMu.Unlock()
	log.Warn("[lock]snapshot chain start fork.longest chain:%s, currentchain:%s", longest.ChainId(), current.ChainId())

	k, f, err := self.getForkPoint(longest, current)
	if err != nil {
		log.Error("get snapshot forkPoint err. err:%v", err)
		return
	}
	forkPoint := f.(*common.SnapshotBlock)
	keyPoint := k.(*common.SnapshotBlock)

	startAcs, endAcs := self.getUnlockAccountSnapshot(forkPoint)

	err = self.consensus.UnLockAccounts(startAcs, endAcs)
	if err != nil {
		log.Error("unlock accounts fail. err:%v", err)
		return
	}
	err = self.consensus.ForkAccounts(keyPoint, forkPoint)
	if err != nil {
		log.Error("rollback accounts fail. err:%v", err)
		return
	}
	err = self.Rollback(forkPoint.Height(), forkPoint.Hash())
	if err != nil {
		log.Error("rollback snapshot fail. err:%v", err)
		return
	}
	err = self.CurrentModifyToChain(longest)
	if err != nil {
		log.Error("snapshot modify current fail. err:%v", err)
		return
	}
	version.IncForkVersion()
}

func (self *SnapshotPool) loop() {
	defer self.wg.Done()
	for {
		select {
		case <-self.closed:
			return
		default:
			self.LoopGenSnippetChains()
			self.LoopAppendChains()
			self.LoopFetchForSnippets()
			self.loopCheckCurrentInsert()
			time.Sleep(time.Second)
		}

	}
}

func (self *SnapshotPool) loopCheckCurrentInsert() {
	self.rwMu.RLock()
	defer self.rwMu.RUnlock()
	self.CheckCurrentInsert(self.insertSnapshotFailCallback, self.insertSnapshotSuccessCallback)
}
func (self *SnapshotPool) Start() {
	self.wg.Add(1)
	go self.loop()
	self.wg.Add(1)
	go self.loopCheckFork()
	log.Info("snapshot_pool[%s] started.", self.Id)
}
func (self *SnapshotPool) Stop() {
	close(self.closed)
	self.wg.Wait()
	log.Info("snapshot_pool[%s] stopped.", self.Id)
}

func (self *SnapshotPool) insertSnapshotFailCallback(b common.Block, s verifier.BlockVerifyStat) {
	block := b.(*common.SnapshotBlock)
	stat := s.(*verifier.SnapshotBlockVerifyStat)
	results := stat.Results()

	for _, account := range block.Accounts {
		result := results[account.Addr]
		if result == verifier.FAIL {
			self.consensus.ForkAccountTo(account)
		}
	}
}

func (self *SnapshotPool) insertSnapshotSuccessCallback(b common.Block, s verifier.BlockVerifyStat) {
	block := b.(*common.SnapshotBlock)

	for _, account := range block.Accounts {
		self.consensus.SnapshotAccount(block, account)
	}
}
func (self *SnapshotPool) getUnlockAccountSnapshot(block *common.SnapshotBlock) (map[string]*common.SnapshotPoint, map[string]*common.SnapshotPoint) {
	h := self.chainpool.diskChain.Head()
	head := h.(*common.SnapshotBlock)
	startAcs := make(map[string]*common.SnapshotPoint)
	endAcs := make(map[string]*common.SnapshotPoint)

	self.accounts(startAcs, endAcs, head)
	for i := head.Height() - 1; i > block.Height(); i-- {
		b := self.chainpool.diskChain.getBlock(i, false)
		if b != nil {
			block := b.block.(*common.SnapshotBlock)
			self.accounts(startAcs, endAcs, block)
		}
	}
	return startAcs, endAcs
}

func (self *SnapshotPool) accounts(start map[string]*common.SnapshotPoint, end map[string]*common.SnapshotPoint, block *common.SnapshotBlock) {
	hs := block.Accounts
	for _, v := range hs {
		point := &common.SnapshotPoint{SnapshotHeight: block.Height(), SnapshotHash: block.Hash(), AccountHeight: v.Height, AccountHash: v.Hash}
		s := start[v.Addr]
		if s == nil {
			start[v.Addr] = point
		}
		end[v.Addr] = point
	}
}
