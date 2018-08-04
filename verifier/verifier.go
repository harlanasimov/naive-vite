package verifier

import (
	"github.com/viteshan/naive-vite/common"
	"github.com/viteshan/naive-vite/ledger"
)

type VerifyResult int

const (
	NONE VerifyResult = iota
	PENDING
	FAIL
	SUCCESS
)

func (self VerifyResult) Done() bool {
	if self == FAIL || self == SUCCESS {
		return true
	} else {
		return false
	}
}

type VerifyType int

const (
	VerifyReferred VerifyType = iota
)

type Verifier interface {
	VerifyReferred(block common.Block, stat BlockVerifyStat)
	NewVerifyStat(t VerifyType, block common.Block) BlockVerifyStat
}

type BlockVerifyStat interface {
	VerifyResult() VerifyResult
	Reset()
}

type SnapshotVerifier struct {
	snapshotReader SnapshotReader
	accountReader  AccountReader
}

func (self *SnapshotVerifier) VerifyReferred(b common.Block, s BlockVerifyStat) {
	var block *ledger.SnapshotBlock

	var stat *SnapshotBlockVerifyStat

	accounts := block.Accounts
	if stat.result.Done() {
		return
	}
	i := 0
	for _, v := range accounts {
		result := stat.results[v.Addr]
		if result.Done() {
			i++
			continue
		}

		addr := v.Addr
		hash := v.Hash
		height := v.Height
		block := self.accountReader.getByHFromChain(addr, height)
		if block == nil {
			stat.results[v.Addr] = PENDING
		} else {
			if block.Hash() == hash {
				i++
				stat.results[v.Addr] = SUCCESS
			} else {
				stat.results[v.Addr] = FAIL
				stat.result = FAIL
				return
			}
		}
	}
	if i == len(accounts) {
		stat.result = SUCCESS
		return
	}
}

type SnapshotBlockVerifyStat struct {
	result   VerifyResult
	accounts []*common.AccountHashH
	results  map[string]VerifyResult
}

func (self *SnapshotBlockVerifyStat) VerifyResult() VerifyResult {
	return self.result
}

func (self *SnapshotBlockVerifyStat) Reset() {
	self.result = PENDING
	self.results = make(map[string]VerifyResult)
}

func (self *SnapshotVerifier) NewVerifyStat(t VerifyType, b common.Block) BlockVerifyStat {
	var block *ledger.SnapshotBlock
	return &SnapshotBlockVerifyStat{result: NONE, accounts: block.Accounts}
}

type AccountVerifier struct {
	snapshotReader SnapshotReader
	accountReader  AccountReader
}

type SnapshotReader interface {
	contains(height int, hash string) bool
	Accounts(accountHash string) []common.AccountHashH
}

type AccountReader interface {
	getFromChain(account string, hash string) *ledger.AccountStateBlock
	getByHFromChain(account string, height int) *ledger.AccountStateBlock
	getReferred(account string, sourceHash string) *ledger.AccountStateBlock
}

func (self *AccountVerifier) VerifyReferred(b common.Block, s BlockVerifyStat) {
	var block *ledger.AccountStateBlock
	var stat *AccountBlockVerifyStat

	// referred snapshot
	snapshotHeight := block.SnapshotHeight
	snapshotHash := block.SnapshotHash

	if !stat.referredSnapshotResult.Done() {
		snapshotR := self.snapshotReader.contains(snapshotHeight, snapshotHash)
		if snapshotR {
			stat.referredSnapshotResult = SUCCESS
		} else {
			stat.referredSnapshotResult = PENDING
		}
	}

	// self amount and response
	if !stat.referredSelfResult.Done() {
		if block.BlockType == ledger.RECEIVED {
			same := self.accountReader.getReferred(block.To, block.SourceHash)
			if same != nil {
				stat.referredSelfResult = FAIL
				return
			}
		}
		selfAmount := self.checkSelfAmount(block)
		stat.referredSelfResult = selfAmount
		if selfAmount == FAIL {
			return
		}
	}
	// from amount
	if !stat.referredFromResult.Done() {
		if block.BlockType == ledger.RECEIVED {

			fromAmount := self.checkFromAmount(block)
			stat.referredFromResult = fromAmount
			if fromAmount == FAIL {
				return
			}
		} else {
			stat.referredFromResult = SUCCESS
		}
	}
}

type AccountBlockVerifyStat struct {
	referredSnapshotResult VerifyResult
	referredSelfResult     VerifyResult
	referredFromResult     VerifyResult
}

func (self *AccountBlockVerifyStat) VerifyResult() VerifyResult {
	if self.referredSelfResult == FAIL ||
		self.referredFromResult == FAIL ||
		self.referredSnapshotResult == FAIL {
		return FAIL
	}
	if self.referredSelfResult == SUCCESS &&
		self.referredFromResult == SUCCESS &&
		self.referredSnapshotResult == SUCCESS {
		return FAIL
	}
	return PENDING
}

func (self *AccountBlockVerifyStat) Reset() {
	self.referredFromResult = PENDING
	self.referredSnapshotResult = PENDING
	self.referredSelfResult = PENDING
}

func (self *AccountVerifier) NewVerifyStat(t VerifyType, block common.Block) BlockVerifyStat {
	return &AccountBlockVerifyStat{}
}
func (self *AccountVerifier) checkSelfAmount(block *ledger.AccountStateBlock) VerifyResult {
	last := self.accountReader.getReferred(block.To, block.PreHash())
	if last == nil {
		return PENDING
	}

	if last.SnapshotHeight > block.SnapshotHeight {
		return FAIL
	}

	if last.Amount+block.ModifiedAmount == block.Amount &&
		block.Amount > 0 {
		return SUCCESS
	} else {
		return FAIL
	}
}
func (self *AccountVerifier) checkFromAmount(block *ledger.AccountStateBlock) VerifyResult {
	source := self.accountReader.getFromChain(block.From, block.SourceHash)
	if source == nil {
		return PENDING
	}

	if block.SnapshotHeight < source.SnapshotHeight {
		return FAIL
	}
	if source.ModifiedAmount+block.ModifiedAmount == 0 {
		return SUCCESS
	} else {
		return FAIL
	}
}
