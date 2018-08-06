package verifier

import (
	"github.com/viteshan/naive-vite/common"
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

type VerifierFailCallback func(block common.Block, stat BlockVerifyStat)
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

func NewSnapshotVerifier(snapshotReader SnapshotReader, accountReader AccountReader) *SnapshotVerifier {
	verifier := &SnapshotVerifier{}
	verifier.snapshotReader = snapshotReader
	verifier.accountReader = accountReader
	return verifier
}

func (self *SnapshotVerifier) VerifyReferred(b common.Block, s BlockVerifyStat) {
	block := b.(*common.SnapshotBlock)

	stat := s.(*SnapshotBlockVerifyStat)

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
		block := self.accountReader.GetByHFromChain(addr, height)
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

func (self *SnapshotBlockVerifyStat) Results() map[string]VerifyResult {
	return self.results
}

func (self *SnapshotBlockVerifyStat) VerifyResult() VerifyResult {
	return self.result
}

func (self *SnapshotBlockVerifyStat) Reset() {
	self.result = PENDING
	self.results = make(map[string]VerifyResult)
}

func (self *SnapshotVerifier) NewVerifyStat(t VerifyType, b common.Block) BlockVerifyStat {
	var block *common.SnapshotBlock
	return &SnapshotBlockVerifyStat{result: NONE, accounts: block.Accounts}
}

type AccountVerifier struct {
	snapshotReader SnapshotReader
	accountReader  AccountReader
}

func NewAccountVerifier(snapshotReader SnapshotReader, accountReader AccountReader) *AccountVerifier {
	verifier := &AccountVerifier{}
	verifier.snapshotReader = snapshotReader
	verifier.accountReader = accountReader
	return verifier
}

type SnapshotReader interface {
	Contains(height int, hash string) bool
}

type AccountReader interface {
	GetFromChain(account string, hash string) *common.AccountStateBlock
	GetByHFromChain(account string, height int) *common.AccountStateBlock
	GetReferred(account string, sourceHash string) *common.AccountStateBlock
}

func (self *AccountVerifier) VerifyReferred(b common.Block, s BlockVerifyStat) {
	block := b.(*common.AccountStateBlock)
	stat := s.(*AccountBlockVerifyStat)

	// referred snapshot
	snapshotHeight := block.SnapshotHeight
	snapshotHash := block.SnapshotHash

	if !stat.referredSnapshotResult.Done() {
		snapshotR := self.snapshotReader.Contains(snapshotHeight, snapshotHash)
		if snapshotR {
			stat.referredSnapshotResult = SUCCESS
		} else {
			stat.referredSnapshotResult = PENDING
		}
	}

	// self amount and response
	if !stat.referredSelfResult.Done() {
		if block.BlockType == common.RECEIVED {
			same := self.accountReader.GetReferred(block.To, block.SourceHash)
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
		if block.BlockType == common.RECEIVED {

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
func (self *AccountVerifier) checkSelfAmount(block *common.AccountStateBlock) VerifyResult {
	last := self.accountReader.GetByHFromChain(block.To, block.Height()-1)

	if last == nil {
		return PENDING
	}
	if last.Hash() != block.PreHash() {
		return FAIL
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
func (self *AccountVerifier) checkFromAmount(block *common.AccountStateBlock) VerifyResult {
	source := self.accountReader.GetFromChain(block.From, block.SourceHash)
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
