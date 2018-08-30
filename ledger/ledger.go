package ledger

import (
	"errors"
	"sync"
	"time"

	"github.com/viteshan/naive-vite/common"
	"github.com/viteshan/naive-vite/common/face"
	"github.com/viteshan/naive-vite/common/log"
	"github.com/viteshan/naive-vite/ledger/pool"
	"github.com/viteshan/naive-vite/syncer"
	"github.com/viteshan/naive-vite/tools"
	"github.com/viteshan/naive-vite/verifier"
)

type Ledger interface {
	face.SnapshotChainReader
	face.AccountChainReader
	// from other peer
	AddSnapshotBlock(block *common.SnapshotBlock)
	// from self
	MiningSnapshotBlock(address string, timestamp int64) error
	// from other peer
	AddAccountBlock(account string, block *common.AccountStateBlock) error
	// from self
	//RequestAccountBlock(address string, block *common.AccountStateBlock) error
	RequestAccountBlock(from string, to string, amount int) error
	ResponseAccountBlock(from string, to string, reqHash string) error
	// create account genesis block
	CreateAccount(address string) error
	HeadAccount(address string) (*common.AccountStateBlock, error)
	HeadSnapshost() (*common.SnapshotBlock, error)
	GetAccountBalance(address string) int
	ExistAccount(address string) bool
	ListRequest(address string) []*Req
	Start()
	Stop()
	Init(syncer syncer.Syncer)
}

type ledger struct {
	ac        map[string]*AccountChain
	sc        *Snapshotchain
	pendingSc *pool.SnapshotPool
	pendingAc map[string]*pool.AccountPool
	reqPool   *reqPool

	snapshotVerifier *verifier.SnapshotVerifier
	accountVerifier  *verifier.AccountVerifier
	syncer           syncer.Syncer
	rwMutex          *sync.RWMutex
}

func (self *ledger) GetAccountBlocksByHashH(address string, hashH common.HashHeight) *common.AccountStateBlock {
	ac := self.selfAc(address)
	if ac == nil || ac.Head() == nil {
		return nil
	}
	return ac.GetBlockByHashH(hashH)
}

func (self *ledger) GetSnapshotBlocksByHashH(hashH common.HashHeight) *common.SnapshotBlock {
	return self.sc.GetBlockByHashH(hashH)
}

func (self *ledger) HeadAccount(address string) (*common.AccountStateBlock, error) {
	ac := self.selfAc(address)
	if ac == nil {
		return nil, errors.New("account not exist.")
	}
	head := ac.Head()
	if head == nil {
		return nil, errors.New("head not exist.")
	}
	block := head.(*common.AccountStateBlock)
	return block, nil
}

func (self *ledger) HeadSnapshost() (*common.SnapshotBlock, error) {
	head := self.sc.Head()
	if head == nil {
		return nil, errors.New("head not exist.")
	}
	block := head.(*common.SnapshotBlock)
	return block, nil
}

func (self *ledger) CreateAccount(address string) error {
	head := self.sc.Head()
	if self.ac[address] != nil {
		log.Warn("exist account for %s.", address)
		return errors.New("exist account " + address)
	}
	accountChain := NewAccountChain(address, self.reqPool, head.Height(), head.Hash())
	accountPool := pool.NewAccountPool("accountChainPool-" + address)

	accountPool.Init(accountChain.insertChain, accountChain.removeChain, self.accountVerifier, pool.NewFetcher(address, self.syncer.Fetcher()), accountChain, self.rwMutex.RLocker(), accountChain)
	self.ac[address] = accountChain
	self.pendingAc[address] = accountPool
	accountPool.Start()
	return nil
}

func (self *ledger) ExistAccount(address string) bool {
	return self.selfAc(address) != nil
}
func (self *ledger) GetAccountBalance(address string) int {
	ac := self.selfAc(address)
	if ac == nil || ac.Head() == nil {
		return 0
	}
	return ac.Head().(*common.AccountStateBlock).Amount
}

func (self *ledger) AddSnapshotBlock(block *common.SnapshotBlock) {
	log.Info("snapshot block[%s] add.", block.Hash())
	self.pendingSc.AddBlock(block)
}

func (self *ledger) MiningSnapshotBlock(address string, timestamp int64) error {
	//self.pendingSc.AddDirectBlock(block)
	self.rwMutex.Lock()
	defer self.rwMutex.Unlock()
	head := self.sc.head
	//common.SnapshotBlock{}
	var accounts []*common.AccountHashH
	for k, v := range self.ac {
		i, s := v.NextSnapshotPoint()
		if i < 0 {
			continue
		}
		accounts = append(accounts, common.NewAccountHashH(k, s, i))
	}
	if len(accounts) == 0 {
		accounts = nil
	}
	block := common.NewSnapshotBlock(head.Height()+1, "", head.Hash(), address, time.Unix(timestamp, 0), accounts)
	block.SetHash(tools.CalculateSnapshotHash(block))
	err := self.pendingSc.AddDirectBlock(block)
	if err != nil {
		log.Error("add direct block error. ", err)
		return err
	}
	self.syncer.Sender().BroadcastSnapshotBlocks([]*common.SnapshotBlock{block})
	return nil
}

func (self *ledger) AddAccountBlock(account string, block *common.AccountStateBlock) error {
	log.Info("account[%s] block[%s] add.", block.Signer(), block.Hash())
	self.selfPendingAc(account).AddBlock(block)
	return nil
}

func (self *ledger) RequestAccountBlock(from string, to string, amount int) error {
	headAccount, _ := self.HeadAccount(from)
	headSnaphost, _ := self.HeadSnapshost()

	newBlock := common.NewAccountBlockFrom(headAccount, from, time.Now(), amount, headSnaphost,
		common.SEND, from, to, "")
	newBlock.SetHash(tools.CalculateAccountHash(newBlock))
	return self.selfPendingAc(from).AddDirectBlock(newBlock)
}
func (self *ledger) ResponseAccountBlock(from string, to string, reqHash string) error {
	fromAc := self.selfAc(from)
	if fromAc == nil {
		return errors.New("not exist for account[" + from + "]")
	}
	toAc := self.selfAc(to)
	if toAc == nil {
		return errors.New("not exist for account[" + to + "]")
	}
	b := fromAc.GetBlockByHash(reqHash)
	if b == nil {
		return errors.New("not exist for account[" + from + "]block[" + reqHash + "]")
	}

	reqBlock := b

	prev := toAc.Head().(*common.AccountStateBlock)
	snapshostBlock, _ := self.HeadSnapshost()

	modifiedAmount := -reqBlock.ModifiedAmount
	block := common.NewAccountBlock(prev.Height()+1, "", prev.Hash(), to, time.Now(), prev.Amount+modifiedAmount, modifiedAmount, snapshostBlock.Height(), snapshostBlock.Hash(), common.RECEIVED, from, to, reqHash)
	block.SetHash(tools.CalculateAccountHash(block))
	return self.selfPendingAc(to).AddDirectBlock(block)
}

func (self *ledger) selfAc(addr string) *AccountChain {
	chain, ok := self.ac[addr]
	if !ok {
		self.initAc(addr)
		return self.ac[addr]
	}
	return chain
}

func (self *ledger) initAc(address string) {
	accountChain := NewAccountChain(address, self.reqPool)
	accountPool := pool.NewAccountPool("accountChainPool-" + address)

	accountPool.Init(accountChain.insertChain, accountChain.removeChain, self.accountVerifier, pool.NewFetcher(address, self.syncer.Fetcher()), accountChain, self.rwMutex.RLocker(), accountChain)
	accountPool.Start()

	self.ac[address] = accountChain
	self.pendingAc[address] = accountPool
}

func (self *ledger) selfPendingAc(addr string) *pool.AccountPool {
	p, ok := self.pendingAc[addr]
	if !ok {
		self.initAc(addr)
		return self.pendingAc[addr]
	}
	return p
}

func (self *ledger) ForkAccounts(keyPoint *common.SnapshotBlock, forkPoint *common.SnapshotBlock) error {
	tasks := make(map[string]*common.AccountHashH)

	for k, v := range self.pendingAc {
		ok, block, err := v.FindRollbackPointByReferSnapshot(forkPoint.Height(), forkPoint.Hash())
		if err != nil {
			log.Error("%v", err)
			continue
		}
		if !ok {
			continue
		} else {
			h := common.NewAccountHashH(k, block.Hash(), block.Height())
			tasks[h.Addr] = h
		}
	}
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
func (self *ledger) getWaitRollbackAccounts(tasks map[string]*common.AccountHashH) map[string]*common.AccountHashH {
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
			sourceHash := v.SourceHash
			signer := v.Signer()
			req := self.reqPool.confirmed(signer, sourceHash)
			if req != nil {
				if canAdd(tasks, req) {
					tasks[req.Addr] = req
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

func (self *ledger) ForkAccountTo(h *common.AccountHashH) error {
	this := self.selfPendingAc(h.Addr)
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

func (self *ledger) SnapshotAccount(block *common.SnapshotBlock, h *common.AccountHashH) {
	self.selfAc(h.Addr).SnapshotPoint(block.Height(), block.Hash(), h)
}
func (self *ledger) UnLockAccounts(startAcs map[string]*common.SnapshotPoint, endAcs map[string]*common.SnapshotPoint) error {
	for k, v := range startAcs {
		err := self.selfAc(k).RollbackSnapshotPoint(v, endAcs[k])
		if err != nil {
			return err
		}
	}
	return nil
}
func NewLedger() *ledger {
	ledger := &ledger{}
	return ledger
}

func (self *ledger) Init(syncer syncer.Syncer) {
	self.rwMutex = new(sync.RWMutex)

	sc := NewSnapshotChain()
	self.snapshotVerifier = verifier.NewSnapshotVerifier(sc, self)
	self.accountVerifier = verifier.NewAccountVerifier(sc, self)
	self.syncer = syncer

	snapshotPool := pool.NewSnapshotPool("snapshotPool")
	snapshotPool.Init(sc.insertChain,
		sc.removeChain,
		self.snapshotVerifier,
		pool.NewFetcher("", syncer.Fetcher()),
		sc,
		self.rwMutex,
		self)
	self.reqPool = newReqPool()

	acPools := make(map[string]*pool.AccountPool)
	acs := make(map[string]*AccountChain)
	accounts := Accounts()
	for _, account := range accounts {
		ac := NewAccountChain(account, self.reqPool, sc.head.Height(), sc.head.Hash())
		accountPool := pool.NewAccountPool("accountChainPool-" + account)
		accountPool.Init(ac.insertChain, ac.removeChain, self.accountVerifier, pool.NewFetcher(account, syncer.Fetcher()), ac, self.rwMutex.RLocker(), ac)
		acs[account] = ac
		acPools[account] = accountPool
	}

	self.ac = acs
	self.sc = sc
	self.pendingAc = acPools
	self.pendingSc = snapshotPool
}

func (self *ledger) GetFromChain(account string, hash string) *common.AccountStateBlock {
	b := self.selfAc(account).GetBlockByHash(hash)
	if b == nil {
		return nil
	}
	return b
}
func (self *ledger) GetByHFromChain(account string, height int) *common.AccountStateBlock {
	b := self.selfAc(account).GetBlock(height)
	if b == nil {
		return nil
	}
	block := b.(*common.AccountStateBlock)
	return block
}
func (self *ledger) ListRequest(address string) []*Req {
	reqs := self.reqPool.getReqs(address)
	return reqs
}
func (self *ledger) GetReferred(account string, sourceHash string) *common.AccountStateBlock {
	self.selfAc(account).GetBySourceBlock(sourceHash)
	return nil
}
func (self *ledger) Start() {
	for _, pending := range self.pendingAc {
		pending.Start()
	}
	self.pendingSc.Start()
}
func (self *ledger) Stop() {
	self.pendingSc.Stop()
	for _, pending := range self.pendingAc {
		pending.Stop()
	}
}

func Accounts() []string {
	//return []string{"viteshan1", "viteshan2", "viteshan3"}
	return []string{}
}
