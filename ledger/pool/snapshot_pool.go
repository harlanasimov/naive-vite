package pool

import (
	"github.com/viteshan/naive-vite/common"
	"github.com/viteshan/naive-vite/common/log"
	"github.com/viteshan/naive-vite/consensus"
	"github.com/viteshan/naive-vite/syncer"
	"github.com/viteshan/naive-vite/verifier"
	"github.com/viteshan/naive-vite/version"
	"sync"
	"time"
)

type SnapshotPool struct {
	BCPool
	rwMu      *sync.RWMutex
	consensus consensus.AccountsConsensus
}

func NewSnapshotPool(name string) *SnapshotPool {
	pool := &SnapshotPool{}
	pool.Id = name
	return pool
}

func (self *SnapshotPool) Init(insertChainFn insertChainForkCheck,
	removeChainFn removeChainForkCheck,
	verifier verifier.Verifier,
	syncer syncer.Syncer,
	reader ChainReader,
	rwMu *sync.RWMutex,
	accountsConsensus consensus.AccountsConsensus) {
	self.rwMu = rwMu
	self.consensus = accountsConsensus
	self.BCPool.init(insertChainFn, removeChainFn, verifier, syncer, reader)
}

func (self *SnapshotPool) loopCheckFork() {
	for {
		self.checkFork()
		// check fork every 2 sec.
		time.Sleep(2 * time.Second)
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
	log.Warn("snapshot chain start fork.longest chain:%s, currentchain:%s", longest.ChainId(), current.ChainId())

	keyPoint, forkPoint, err := self.getForkPoint(longest, current)
	if err != nil {
		return
	}
	self.consensus.ForkAccounts(keyPoint, forkPoint)
	self.CurrentModifyToChain(longest)
	version.IncForkVersion()
}
func (self *SnapshotPool) getForkPoint(longest Chain, current Chain) (*common.SnapshotBlock, *common.SnapshotBlock, error) {
	curHeadHeight := current.HeadHeight()

	i := curHeadHeight
	var forkedBlock common.Block

	for {
		block := longest.GetBlock(i)
		curBlock := current.GetBlock(i)
		if block == nil {
			log.Error("longest chain is not longest. chainId:%s. height:%d", longest.ChainId(), i)
			return nil, nil, common.StrError{"longest chain error."}
		}

		if curBlock == nil {
			log.Error("current chain is wrong. chainId:%s. height:%d", current.ChainId(), i)
			return nil, nil, common.StrError{"current chain error."}
		}

		if block.Hash() == curBlock.Hash() {
			forkedBlock = block
			keyPoint := longest.GetBlock(i + 1)
			key := keyPoint.(*common.SnapshotBlock)
			forked := forkedBlock.(*common.SnapshotBlock)
			return key, forked, nil
		}
		i = i - 1
	}
	return nil, nil, common.StrError{"can't find fork point"}
}

func (self *SnapshotPool) loop() {
	for {
		self.LoopGenSnippetChains()
		self.LoopAppendChains()
		self.LoopFetchForSnippets()
		self.CheckCurrentInsert(self.insertSnapshotFailCallback)
		time.Sleep(time.Second)
	}
}
func (self *SnapshotPool) Start() {
	go self.loop()
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
