package pool

import (
	"errors"
	"sync"
	"time"

	"github.com/viteshan/naive-vite/common"
	"github.com/viteshan/naive-vite/common/log"
	"github.com/viteshan/naive-vite/tools"
	"github.com/viteshan/naive-vite/verifier"
)

type accountPool struct {
	BCPool
	mu     sync.Locker
	closed chan struct{}
	wg     sync.WaitGroup
	rw     *accountCh
}

func newAccountPool(name string, rw *accountCh) *accountPool {
	pool := &accountPool{}
	pool.Id = name
	pool.closed = make(chan struct{})
	pool.verifierFailcallback = pool.insertAccountFailCallback
	pool.verifierSuccesscallback = pool.insertAccountSuccessCallback
	pool.rw = rw
	return pool
}

func (self *accountPool) Init(
	verifier verifier.Verifier,
	syncer *fetcher,
	mu sync.Locker) {

	self.mu = mu
	self.BCPool.init(self.rw, verifier, syncer)
}

// 1. must be in diskchain
func (self *accountPool) TryRollback(rollbackHeight int, rollbackHash string) ([]*common.AccountStateBlock, error) {
	{ // check logic
		w := self.chainpool.diskChain.getBlock(rollbackHeight, false)
		if w == nil || w.block.Hash() != rollbackHash {
			return nil, errors.New("error rollback cmd.")
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

// rollback to current
func (self *accountPool) FindRollbackPointByReferSnapshot(snapshotHeight int, snapshotHash string) (bool, *common.AccountStateBlock, error) {
	head := self.chainpool.diskChain.Head().(*common.AccountStateBlock)
	if head.SnapshotHeight < snapshotHeight {
		return false, nil, nil
	}

	accountBlock := self.rw.findAboveSnapshotHeight(snapshotHeight)
	if accountBlock == nil {
		return true, nil, nil
	} else {
		return true, accountBlock, nil
	}
}

func (self *accountPool) FindRollbackPointForAccountHashH(height int, hash string) (bool, *common.AccountStateBlock, Chain, error) {
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

func (self *accountPool) loop() {
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

func (self *accountPool) loopCheckCurrentInsert() {
	if self.chainpool.current.size() == 0 {
		return
	}
	self.mu.Lock()
	defer self.mu.Unlock()
	self.CheckCurrentInsert()
}

func (self *accountPool) accountTryInsert() {
	cp := self.chainpool
	current := cp.current
	minH := current.tailHeight + 1
	headH := current.headHeight
L:
	for i := minH; i <= headH; i++ {
		wrapper := current.getBlock(i, false)
		block := wrapper.block
		stat := wrapper.verifyStat
		//if !wrapper.checkForkVersion() {
		wrapper.reset()
		//}
		tools.CalculateAccountHash()
		cp.verifier.VerifyReferred(block, stat)
		if !wrapper.checkForkVersion() {
			wrapper.reset()
			continue
		}
		result := stat.VerifyResult()
		switch result {
		case verifier.PENDING:

		case verifier.FAIL:
			log.Error("forkedChain forked. verify result is %s. block info:account[%s],hash[%s],height[%d]",
				result, block.Signer(), block.Hash(), block.Height())
			if verifierFailcallback != nil {
				verifierFailcallback(block, stat)
			}
			break L
		case verifier.SUCCESS:
			if block.Height() == current.tailHeight+1 {
				err := cp.writeToChain(current, wrapper)
				if err == nil && insertSuccessCallback != nil {
					insertSuccessCallback(block, stat)
				}
			}
		default:
			log.Error("Unexpected things happened. verify result is %d. block info:account[%s],hash[%s],height[%d]",
				result, block.Signer(), block.Hash(), block.Height())
		}
	}
}

func (self *accountPool) Start() {
	self.wg.Add(1)
	go self.loop()
	log.Info("account_pool[%s] started", self.Id)
}

func (self *accountPool) Stop() {
	close(self.closed)
	self.wg.Wait()
	log.Info("account_pool[%s] stopped", self.Id)
}

func (self *accountPool) insertAccountFailCallback(b common.Block, s verifier.BlockVerifyStat) {
	log.Info("do nothing. height:%d, hash:%s, pool:%s", b.Height(), b.Hash(), self.Id)
}

func (self *accountPool) insertAccountSuccessCallback(b common.Block, s verifier.BlockVerifyStat) {
	log.Info("do nothing. height:%d, hash:%s, pool:%s", b.Height(), b.Hash(), self.Id)
}
func (self *accountPool) FindInChain(hash string, height int) bool {

	for _, c := range self.chainpool.chains {
		b := c.heightBlocks[height]
		if b == nil {
			continue
		} else {
			if b.block.Hash() == hash {
				return true
			}
		}
	}
	return false
}
