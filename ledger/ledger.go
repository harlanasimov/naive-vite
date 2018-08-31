package ledger

import (
	"errors"
	"sync"
	"time"

	"encoding/json"

	"github.com/viteshan/naive-vite/common"
	"github.com/viteshan/naive-vite/common/face"
	"github.com/viteshan/naive-vite/common/log"
	"github.com/viteshan/naive-vite/ledger/pool"
	"github.com/viteshan/naive-vite/syncer"
	"github.com/viteshan/naive-vite/tools"
	"github.com/viteshan/naive-vite/verifier"
)

type Ledger interface {
	face.ChainRw
	GetSnapshotBlocksByHeight(height int) *common.SnapshotBlock
	GetAccountBlockByHeight(address string, height int) *common.AccountStateBlock
	// from self
	MiningSnapshotBlock(address string, timestamp int64) error
	// from self
	//RequestAccountBlock(address string, block *common.AccountStateBlock) error
	RequestAccountBlock(from string, to string, amount int) error
	ResponseAccountBlock(from string, to string, reqHash string) error
	// create account genesis block
	HeadAccount(address string) (*common.AccountStateBlock, error)
	HeadSnapshost() (*common.SnapshotBlock, error)
	GetAccountBalance(address string) int
	ListRequest(address string) []*Req
	Start()
	Stop()
	Init(syncer syncer.Syncer)
	ListAccountBlock(address string) []*common.AccountStateBlock
	ListSnapshotBlock() []*common.SnapshotBlock
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
func (self *ledger) GetAccountBlockByHeight(address string, height int) *common.AccountStateBlock {
	ac := self.selfAc(address)
	if ac == nil || ac.Head() == nil {
		return nil
	}
	return ac.GetBlockByHeight(height)
}

func (self *ledger) GetSnapshotBlocksByHashH(hashH common.HashHeight) *common.SnapshotBlock {
	return self.sc.GetBlockByHashH(hashH)
}

func (self *ledger) GetSnapshotBlocksByHeight(height int) *common.SnapshotBlock {
	return self.sc.GetBlockHeight(height)
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

func (self *ledger) GetAccountBalance(address string) int {
	ac := self.selfAc(address)
	if ac == nil || ac.Head() == nil {
		return 0
	}
	return ac.Head().(*common.AccountStateBlock).Amount
}

func (self *ledger) AddSnapshotBlock(block *common.SnapshotBlock) {
	log.Info("snapshot block[%d][%s] add.", block.Height(), block.Hash())
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
	log.Info("account[%s] block[%d][%s] add.", block.Signer(), block.Height(), block.Hash())
	self.selfPendingAc(account).AddBlock(block)
	return nil
}

func (self *ledger) RequestAccountBlock(from string, to string, amount int) error {
	headAccount, _ := self.HeadAccount(from)
	headSnaphost, _ := self.HeadSnapshost()

	newBlock := common.NewAccountBlockFrom(headAccount, from, time.Now(), amount, headSnaphost,
		common.SEND, from, to, "")
	newBlock.SetHash(tools.CalculateAccountHash(newBlock))
	err := self.selfPendingAc(from).AddDirectBlock(newBlock)
	if err == nil {
		self.syncer.Sender().BroadcastAccountBlocks(from, []*common.AccountStateBlock{newBlock})
	}
	return err
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
	err := self.selfPendingAc(to).AddDirectBlock(block)
	if err == nil {
		self.syncer.Sender().BroadcastAccountBlocks(to, []*common.AccountStateBlock{block})
	}
	return err
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
func (self *ledger) PendingAccountTo(h *common.AccountHashH) error {
	this := self.selfPendingAc(h.Addr)

	inChain := this.FindInChain(h.Hash, h.Height)
	bytes, _ := json.Marshal(h)
	log.Info("inChain:%v, accounts:%s", inChain, string(bytes))
	if !inChain {
		self.syncer.Fetcher().FetchAccount(h.Addr, common.HashHeight{Hash: h.Hash, Height: h.Height}, 5)
		return nil
	}
	return nil
}

func (self *ledger) ForkAccountTo(h *common.AccountHashH) error {
	this := self.selfPendingAc(h.Addr)

	inChain := this.FindInChain(h.Hash, h.Height)
	bytes, _ := json.Marshal(h)
	log.Info("inChain:%v, accounts:%s", inChain, string(bytes))
	if !inChain {
		self.syncer.Fetcher().FetchAccount(h.Addr, common.HashHeight{Hash: h.Hash, Height: h.Height}, 5)
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
		ac := NewAccountChain(account, self.reqPool)
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

func (self *ledger) ListAccountBlock(address string) []*common.AccountStateBlock {
	var blocks []*common.AccountStateBlock
	ac := self.selfAc(address)
	head := ac.Head()
	for i := 0; i < head.Height(); i++ {
		blocks = append(blocks, ac.GetBlockByHeight(i))
	}
	if head.Height() > 0 {
		blocks = append(blocks, head.(*common.AccountStateBlock))
	}
	return blocks
}
func (self *ledger) ListSnapshotBlock() []*common.SnapshotBlock {
	var blocks []*common.SnapshotBlock
	ac := self.sc
	head := ac.Head()
	for i := 0; i < head.Height(); i++ {
		blocks = append(blocks, ac.GetBlockHeight(i))
	}
	if head.Height() > 0 {
		blocks = append(blocks, head.(*common.SnapshotBlock))
	}
	return blocks
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
