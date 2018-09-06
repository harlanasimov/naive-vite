package pool

import (
	"errors"
	"sync"

	"github.com/viteshan/naive-vite/common"
	"github.com/viteshan/naive-vite/common/log"
	"github.com/viteshan/naive-vite/verifier"
	"github.com/viteshan/naive-vite/version"
)

type accountPool struct {
	BCPool
	mu         sync.Locker
	rw         *accountCh
	verifyTask verifier.Task
}

func newAccountPool(name string, rw *accountCh, v *version.Version) *accountPool {
	pool := &accountPool{}
	pool.Id = name
	pool.rw = rw
	pool.version = v
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

func (self *accountPool) loop() int {
	sum := 0
	sum = sum + self.loopGenSnippetChains()
	sum = sum + self.loopAppendChains()
	sum = sum + self.loopFetchForSnippets()
	sum = sum + self.loopAccountTryInsert()
	return sum
}

func (self *accountPool) loopAccountTryInsert() int {
	if self.chainpool.current.size() <= 0 {
		return 0
	}
	if self.verifyTask != nil && !self.verifyTask.Done() {
		return 0
	}
	self.mu.Lock()
	defer self.mu.Unlock()
	sum := self.accountTryInsert()
	if sum > 0 && self.verifyTask != nil {
		reqs := self.verifyTask.Requests()
		self.syncer.fetchReqs(reqs)
	}
	return sum
}

/**
1. fail    something is wrong.
2. pending
	2.1 pending for snapshot
	2.2 pending for other account chain(specific block height)
3. success



fail: If no fork version increase, don't do anything.
pending:
	pending(2.1): If snapshot height is not reached, don't do anything.
	pending(2.2): If other account chain height is not reached, don't do anything.
success:
	really insert to chain.
*/
func (self *accountPool) accountTryInsert() int {
	self.rMu.Lock()
	defer self.rMu.Unlock()
	cp := self.chainpool
	current := cp.current
	minH := current.tailHeight + 1
	headH := current.headHeight
	n := 0
L:
	for i := minH; i <= headH; i++ {
		wrapper := current.getBlock(i, false)
		block := wrapper.block
		wrapper.reset()
		n++
		stat, task := cp.verifier.VerifyReferred(block)
		if !wrapper.checkForkVersion() {
			wrapper.reset()
			break L
		}
		result := stat.VerifyResult()
		switch result {
		case verifier.PENDING:
			self.verifyTask = task
			break L
		case verifier.FAIL:
			log.Error("account block verify fail. block info:account[%s],hash[%s],height[%d]",
				result, block.Signer(), block.Hash(), block.Height())
			self.verifyTask = task
			break L
		case verifier.SUCCESS:
			self.verifyTask = nil
			if block.Height() == current.tailHeight+1 {
				err := cp.writeToChain(current, wrapper)
				if err != nil {
					log.Error("account block write fail. block info:account[%s],hash[%s],height[%d], err:%v",
						result, block.Signer(), block.Hash(), block.Height(), err)
					break L
				}
			} else {
				break L
			}
		default:
			// shutdown process
			log.Fatal("Unexpected things happened. verify result is %d. block info:account[%s],hash[%s],height[%d]",
				result, block.Signer(), block.Hash(), block.Height())
			break L
		}
	}

	return n
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
