package pool

import (
	"encoding/json"

	"github.com/viteshan/naive-vite/common"
	"github.com/viteshan/naive-vite/common/face"
	"github.com/viteshan/naive-vite/common/log"
	"github.com/viteshan/naive-vite/syncer"
	"github.com/viteshan/naive-vite/verifier"

	"sync"

	ch "github.com/viteshan/naive-vite/chain"
)

type BlockPool interface {
	face.PoolWriter
	face.PoolReader
	Start()
	Stop()
	Init(syncer.Fetcher)
}

type pool struct {
	pendingSc *snapshotPool
	pendingAc sync.Map
	fetcher   syncer.Fetcher
	bc        ch.BlockChain

	snapshotVerifier *verifier.SnapshotVerifier
	accountVerifier  *verifier.AccountVerifier

	rwMutex *sync.RWMutex
	acMu    sync.Mutex
}

func NewPool(bc ch.BlockChain, rwMutex *sync.RWMutex) BlockPool {
	self := &pool{bc: bc, rwMutex: rwMutex}
	return self
}

func (self *pool) Init(f syncer.Fetcher) {
	self.snapshotVerifier = verifier.NewSnapshotVerifier(self.bc)
	self.accountVerifier = verifier.NewAccountVerifier(self.bc)
	self.fetcher = f
	snapshotPool := newSnapshotPool("snapshotPool")
	snapshotPool.init(&snapshotCh{self.bc},
		self.snapshotVerifier,
		NewFetcher("", self.fetcher),
		self.rwMutex,
		self)

	self.pendingSc = snapshotPool
}
func (self *pool) Start() {
	self.pendingSc.Start()
}
func (self *pool) Stop() {
	self.pendingSc.Stop()
	self.pendingAc.Range(func(k, v interface{}) bool {
		p := v.(*accountPool)
		p.Stop()
		return true
	})
}

func (self *pool) AddSnapshotBlock(block *common.SnapshotBlock) error {
	self.pendingSc.AddBlock(block)
	return nil
}

func (self *pool) AddDirectSnapshotBlock(block *common.SnapshotBlock) error {
	return self.pendingSc.AddDirectBlock(block)
}

func (self *pool) AddAccountBlock(address string, block *common.AccountStateBlock) error {
	self.selfPendingAc(address).AddBlock(block)
	return nil
}

func (self *pool) AddDirectAccountBlock(address string, block *common.AccountStateBlock) error {
	return self.selfPendingAc(address).AddDirectBlock(block)
}

func (self *pool) ExistInPool(address string, requestHash string) bool {
	panic("implement me")
}

func (self *pool) ForkAccounts(keyPoint *common.SnapshotBlock, forkPoint *common.SnapshotBlock) error {
	tasks := make(map[string]*common.AccountHashH)
	self.pendingAc.Range(func(k, v interface{}) bool {
		a := v.(*accountPool)
		ok, block, err := a.FindRollbackPointByReferSnapshot(forkPoint.Height(), forkPoint.Hash())
		if err != nil {
			log.Error("%v", err)
			return true
		}
		if !ok {
			return true
		} else {
			h := common.NewAccountHashH(k.(string), block.Hash(), block.Height())
			tasks[h.Addr] = h
		}
		return true
		//}
	})
	waitRollbackAccounts := self.getWaitRollbackAccounts(tasks)

	for _, v := range waitRollbackAccounts {
		err := self.selfPendingAc(v.Addr).Rollback(v.Height, v.Hash)
		if err != nil {
			return err
		}
	}
	for _, v := range keyPoint.Accounts {
		self.ForkAccountTo(v)
	}
	return nil
}
func (self *pool) getWaitRollbackAccounts(tasks map[string]*common.AccountHashH) map[string]*common.AccountHashH {
	waitRollback := make(map[string]*common.AccountHashH)
	for {
		var sendBlocks []*common.AccountStateBlock
		for k, v := range tasks {
			delete(tasks, k)
			if canAdd(waitRollback, v) {
				waitRollback[v.Addr] = v
			}
			addWaitRollback(waitRollback, v)
			tmpBlocks, err := self.selfPendingAc(v.Addr).TryRollback(v.Height, v.Hash)
			if err == nil {
				for _, v := range tmpBlocks {
					sendBlocks = append(sendBlocks, v)
				}
			} else {
				log.Error("%v", err)
			}
		}
		for _, v := range sendBlocks {
			sourceHash := v.Hash()
			req := self.bc.GetAccountBySourceHash(v.To, sourceHash)
			h := &common.AccountHashH{Addr: req.Signer(), HashHeight: common.HashHeight{Hash: req.Hash(), Height: req.Height()}}
			if req != nil {
				if canAdd(tasks, h) {
					tasks[h.Addr] = h
				}
			}
		}
		if len(tasks) == 0 {
			break
		}
	}

	return waitRollback
}

// h is closer to genesis
func canAdd(hs map[string]*common.AccountHashH, h *common.AccountHashH) bool {
	hashH := hs[h.Addr]
	if hashH == nil {
		return true
	}

	if h.Height < hashH.Height {
		return true
	}
	return false
}
func addWaitRollback(hs map[string]*common.AccountHashH, h *common.AccountHashH) {
	hashH := hs[h.Addr]
	if hashH == nil {
		hs[h.Addr] = h
		return
	}

	if hashH.Height < h.Height {
		hs[h.Addr] = h
		return
	}
}
func (self *pool) PendingAccountTo(h *common.AccountHashH) error {
	this := self.selfPendingAc(h.Addr)

	inChain := this.FindInChain(h.Hash, h.Height)
	bytes, _ := json.Marshal(h)
	log.Info("inChain:%v, accounts:%s", inChain, string(bytes))
	if !inChain {
		self.fetcher.FetchAccount(h.Addr, common.HashHeight{Hash: h.Hash, Height: h.Height}, 5)
		return nil
	}
	return nil
}

func (self *pool) ForkAccountTo(h *common.AccountHashH) error {
	this := self.selfPendingAc(h.Addr)

	inChain := this.FindInChain(h.Hash, h.Height)
	bytes, _ := json.Marshal(h)
	log.Info("inChain:%v, accounts:%s", inChain, string(bytes))
	if !inChain {
		self.fetcher.FetchAccount(h.Addr, common.HashHeight{Hash: h.Hash, Height: h.Height}, 5)
		return nil
	}
	ok, block, chain, err := this.FindRollbackPointForAccountHashH(h.Height, h.Hash)
	if err != nil {
		log.Error("%v", err)
	}
	if !ok {
		return nil
	}

	tasks := make(map[string]*common.AccountHashH)
	tasks[h.Addr] = common.NewAccountHashH(h.Addr, block.Hash(), block.Height())
	waitRollback := self.getWaitRollbackAccounts(tasks)
	for _, v := range waitRollback {
		self.selfPendingAc(v.Addr).Rollback(v.Height, v.Hash)
	}
	err = this.CurrentModifyToChain(chain)
	if err != nil {
		log.Error("%v", err)
	}
	return err
}

func (self *pool) UnLockAccounts(startAcs map[string]*common.SnapshotPoint, endAcs map[string]*common.SnapshotPoint) error {
	for k, v := range startAcs {
		err := self.bc.RollbackSnapshotPoint(k, v, endAcs[k])
		if err != nil {
			return err
		}
	}
	return nil
}

func (self *pool) selfPendingAc(addr string) *accountPool {
	chain, ok := self.pendingAc.Load(addr)

	if ok {
		return chain.(*accountPool)
	}

	p := newAccountPool("accountChainPool-"+addr, &accountCh{address: addr, bc: self.bc})
	p.Init(self.accountVerifier, NewFetcher(addr, self.fetcher), self.rwMutex.RLocker())
	p.Start()

	self.acMu.Lock()
	defer self.acMu.Unlock()
	chain, ok = self.pendingAc.Load(addr)
	if ok {
		return chain.(*accountPool)
	}
	self.pendingAc.Store(addr, p)
	return p

}
