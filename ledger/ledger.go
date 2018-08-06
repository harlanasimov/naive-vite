package ledger

import (
	"github.com/viteshan/naive-vite/common"
	"github.com/viteshan/naive-vite/ledger/pool"
	"github.com/viteshan/naive-vite/syncer"
	"github.com/viteshan/naive-vite/tools"
	"github.com/viteshan/naive-vite/verifier"
	"sync"
	"time"
)

type Ledger interface {
	// from other peer
	addSnapshotBlock(block *common.SnapshotBlock)
	// from self
	miningSnapshotBlock(block *common.SnapshotBlock)
	// from other peer
	addAccountBlock(account string, block *common.AccountStateBlock)
	// from self
	miningAccountBlock(block *common.SnapshotBlock)
}

type ledger struct {
	ac        map[string]*AccountChain
	sc        *Snapshotchain
	pendingSc *pool.SnapshotPool
	pendingAc map[string]*pool.AccountPool
}

func (self *ledger) addSnapshotBlock(block *common.SnapshotBlock) {
	self.pendingSc.AddBlock(block)
}

func (self *ledger) miningSnapshotBlock(block *common.SnapshotBlock) {
	self.pendingSc.AddDirectBlock(block)
}

func (self *ledger) addAccountBlock(account string, block *common.AccountStateBlock) {
	self.selfPendingAc(account).AddBlock(block)
}

func (self *ledger) miningAccountBlock(account string, block *common.SnapshotBlock) {
	self.selfPendingAc(account).AddDirectBlock(block)
}

func (self *ledger) selfAc(addr string) *AccountChain {
	return self.ac[addr]
}

func (self *ledger) selfPendingAc(addr string) *pool.AccountPool {
	return self.pendingAc[addr]
}

func (self *ledger) ForkAccounts(keyPoint *common.SnapshotBlock, forkPoint *common.SnapshotBlock) error {
	for _, v := range self.pendingAc {
		err := v.RollbackAndForkAccount(nil, forkPoint)
		if err != nil {
			return nil
		}
	}
	return nil
}

func (self *ledger) ForkAccountTo(h *common.AccountHashH) error {
	return self.selfPendingAc(h.Addr).ForkAccount(h)
}

func newLedger(syncer syncer.Syncer) *ledger {
	rwMutex := new(sync.RWMutex)
	ledger := &ledger{}

	sc := &Snapshotchain{}
	sc.head = genesisSnapshot

	snapshotVerifier := verifier.NewSnapshotVerifier(sc, ledger)
	accountVerifier := verifier.NewAccountVerifier(sc, ledger)

	snapshotPool := pool.NewSnapshotPool("snapshotPool")
	snapshotPool.Init(sc.insertChain,
		sc.removeChain,
		snapshotVerifier,
		syncer,
		sc,
		rwMutex,
		ledger)

	acPools := make(map[string]*pool.AccountPool)
	acs := make(map[string]*AccountChain)
	accounts := Accounts()
	for _, account := range accounts {
		ac := &AccountChain{}
		ac.head = GenAccountBlock(account)
		accountPool := pool.NewAccountPool(account)
		accountPool.Init(ac.insertChain, ac.removeChain, accountVerifier, syncer, ac, rwMutex.RLocker(), ac)
		acs[account] = ac
		acPools[account] = accountPool
	}
	ledger.ac = acs
	ledger.sc = sc
	ledger.pendingAc = acPools
	ledger.pendingSc = snapshotPool
	return ledger
}

func (self *ledger) GetFromChain(account string, hash string) *common.AccountStateBlock {
	return nil
}
func (self *ledger) GetByHFromChain(account string, height int) *common.AccountStateBlock {
	b := self.selfAc(account).GetBlock(height)
	if b == nil {
		return nil
	}
	block := b.(*common.AccountStateBlock)
	return block
}
func (self *ledger) GetReferred(account string, sourceHash string) *common.AccountStateBlock {
	self.selfAc(account).GetBySourceBlock(sourceHash)
	return nil
}

func Accounts() []string {
	return []string{"viteshan1", "viteshan2", "viteshan3"}
}
func GenAccountBlock(address string) *common.AccountStateBlock {
	//height int,
	//	hash string,
	//	preHash string,
	//	signer string,
	//	timestamp time.Time,
	//
	//	amount int,
	//	modifiedAmount int,
	//	snapshotHeight int,
	//	snapshotHash string,
	//	blockType BlockType,
	//	from string,
	//	to string,
	//	sourceHash string,
	block := common.NewAccountBlock(0, "", "", address, time.Unix(1533550878, 0),
		0, 0, 0, "460780b73084275422b520a42ebb9d4f8a8326e1522c79817a19b41ba69dca5b", common.CREATE, "", address, "")
	hash := tools.CalculateAccountHash(block)
	block.SetHash(hash)
	return block
}

//1533550878
var genesisSnapshot = common.NewSnapshotBlock(0, "460780b73084275422b520a42ebb9d4f8a8326e1522c79817a19b41ba69dca5b", "", "viteshan", time.Unix(1533550878, 0), nil)
