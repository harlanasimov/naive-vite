package verifier

import (
	"fmt"

	"time"

	"github.com/viteshan/naive-vite/common"
	"github.com/viteshan/naive-vite/common/face"
	"github.com/viteshan/naive-vite/monitor"
	"github.com/viteshan/naive-vite/version"
)

type AccountVerifier struct {
	reader face.ChainReader
	v      *version.Version
}

func NewAccountVerifier(r face.ChainReader, v *version.Version) *AccountVerifier {
	verifier := &AccountVerifier{reader: r, v: v}
	return verifier
}
func (self *AccountVerifier) verifyGenesis(block *common.AccountStateBlock, stat *AccountBlockVerifyStat) bool {
	defer monitor.LogTime("verify", "accountGenesis", time.Now())
	genesis, _ := self.reader.GenesisSnapshot()
	for _, a := range genesis.Accounts {
		if a.Hash == block.Hash() && a.Height == block.Height() {
			stat.referredFromResult = SUCCESS
			stat.referredSelfResult = SUCCESS
			stat.referredSnapshotResult = SUCCESS
			return true
		}
	}
	stat.referredSelfResult = FAIL
	return true
}
func (self *AccountVerifier) verifySelf(block *common.AccountStateBlock, stat *AccountBlockVerifyStat) bool {
	defer monitor.LogTime("verify", "accountSelf", time.Now())
	// self amount and response
	if block.BlockType == common.RECEIVED && block.Height() == 0 {
		// check genesis block logic
		genesisCheck := self.checkGenesis(block)
		stat.referredSelfResult = genesisCheck
		if genesisCheck == FAIL {
			stat.errMsg = fmt.Sprintf("block[%s][%d][%s] error, genesis check fail.",
				block.Signer(), block.Height(), block.Hash())
			return true
		}

	} else {
		if block.BlockType == common.RECEIVED {
			//check if it has been received
			same := self.reader.GetAccountBySourceHash(block.To, block.SourceHash)
			if same != nil {
				stat.errMsg = fmt.Sprintf("block[%s][%d][%s] error, send block has received.",
					block.Signer(), block.Height(), block.Hash())
				stat.referredSelfResult = FAIL
				return true
			}
		}
		selfAmount := self.checkSelfAmount(block, stat)
		stat.referredSelfResult = selfAmount
		if selfAmount == FAIL {
			return true
		}
	}
	return false
}

func (self *AccountVerifier) verifyFrom(block *common.AccountStateBlock, stat *AccountBlockVerifyStat) bool {
	defer monitor.LogTime("verify", "accountFrom", time.Now())
	// from amount
	if block.BlockType == common.RECEIVED {

		fromAmount := self.checkFromAmount(block, stat)
		stat.referredFromResult = fromAmount
		if fromAmount == FAIL {
			return true
		}
	} else {
		stat.referredFromResult = SUCCESS
	}
	return false
}

func (self *AccountVerifier) verifySnapshot(block *common.AccountStateBlock, stat *AccountBlockVerifyStat) bool {
	defer monitor.LogTime("verify", "accountSnapshot", time.Now())
	// referred snapshot
	snapshotHeight := block.SnapshotHeight
	snapshotHash := block.SnapshotHash

	{ // check snapshot referred
		snapshotR := self.reader.GetSnapshotByHashH(common.HashHeight{Hash: snapshotHash, Height: snapshotHeight})
		if snapshotR != nil {
			stat.referredSnapshotResult = SUCCESS
		} else {
			stat.referredSnapshotResult = PENDING
			stat.task.pendingSnapshot(snapshotHash, snapshotHeight)
		}
	}
	return false
}
func (self *AccountVerifier) VerifyReferred(b common.Block) BlockVerifyStat {
	defer monitor.LogTime("verify", "accountBlock", time.Now())
	block := b.(*common.AccountStateBlock)
	stat := self.newVerifyStat(VerifyReferred, b)

	// genesis account block
	if block.BlockType == common.GENESIS {
		if self.verifyGenesis(block, stat) {
			return stat
		}
	}

	// check snapshot
	if self.verifySnapshot(block, stat) {
		return stat
	}

	// check self
	if self.verifySelf(block, stat) {
		return stat
	}
	// check from
	if self.verifyFrom(block, stat) {
		return stat
	}
	return stat
}

type AccountBlockVerifyStat struct {
	referredSnapshotResult VerifyResult
	referredSelfResult     VerifyResult
	referredFromResult     VerifyResult
	errMsg                 string
	task                   *verifyTask
}

func (self *AccountBlockVerifyStat) Task() Task {
	if self.task == nil {
		return nil
	} else {
		return self.task
	}
}

func (self *AccountBlockVerifyStat) ErrMsg() string {
	return self.errMsg
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
		return SUCCESS
	}
	return PENDING
}

func (self *AccountBlockVerifyStat) Reset() {
	self.referredFromResult = PENDING
	self.referredSnapshotResult = PENDING
	self.referredSelfResult = PENDING
}
func (self *AccountVerifier) newVerifyStat(t VerifyType, block common.Block) *AccountBlockVerifyStat {
	task := &verifyTask{v: self.v, version: self.v.Val(), reader: self.reader, t: time.Now()}
	return &AccountBlockVerifyStat{task: task}
}
func (self *AccountVerifier) checkSelfAmount(block *common.AccountStateBlock, stat *AccountBlockVerifyStat) VerifyResult {
	last, _ := self.reader.HeadAccount(block.Signer())

	if last == nil {
		stat.errMsg = fmt.Sprintf("block[%s][%d][%s] error, last block is nil.",
			block.Signer(), block.Height(), block.Hash())
		return FAIL
	}
	if last.Hash() != block.PreHash() {
		stat.errMsg = fmt.Sprintf("block[%s][%d][%s] preHash[%s] error, last block hash is %s.",
			block.Signer(), block.Height(), block.Hash(), block.PreHash(), last.Hash())
		return FAIL
	}

	if last.SnapshotHeight > block.SnapshotHeight {
		stat.errMsg = fmt.Sprintf("block[%s][%d][%s] snapshot height[%d] error, last block snapshot height is %d.",
			block.Signer(), block.Height(), block.Hash(), block.SnapshotHeight, last.SnapshotHeight)
		return FAIL
	}

	if block.BlockType == common.SEND && block.ModifiedAmount > 0 {
		stat.errMsg = fmt.Sprintf("send block[%s][%d][%s] modifiedAmount[%d] error.",
			block.Signer(), block.Height(), block.Hash(), block.ModifiedAmount)
		return FAIL
	}
	if block.BlockType == common.RECEIVED && block.ModifiedAmount < 0 {
		stat.errMsg = fmt.Sprintf("RECEIVED block[%s][%d][%s] modifiedAmount[%d] error.",
			block.Signer(), block.Height(), block.Hash(), block.ModifiedAmount)
		return FAIL
	}
	if last.Amount+block.ModifiedAmount == block.Amount &&
		block.Amount > 0 {
		return SUCCESS
	} else {
		stat.errMsg = fmt.Sprintf("block amount[%s][%d][%s] cal error. modifiedAmount:%d, Amount:%d, lastAmount:%d",
			block.Signer(), block.Height(), block.Hash(), block.ModifiedAmount, block.Amount, last.Amount)
		return FAIL
	}
}

func (self *AccountVerifier) checkGenesis(block *common.AccountStateBlock) VerifyResult {
	head, _ := self.reader.HeadAccount(block.Signer())
	if head != nil {
		return FAIL
	}
	if block.PreHash() != "" || block.ModifiedAmount != block.Amount {
		return FAIL
	}
	return SUCCESS
}

func (self *AccountVerifier) checkFromAmount(block *common.AccountStateBlock, stat *AccountBlockVerifyStat) VerifyResult {
	source := self.reader.GetAccountByHeight(block.From, block.SourceHeight)
	source2 := self.reader.GetAccountByHash(block.From, block.SourceHash)
	if source != nil && source2 != nil {
		if source2.Hash() != source.Hash() {
			return FAIL
		}
	}
	if source == nil {
		stat.task.pendingAccount(block.From, block.SourceHeight, block.SourceHash, 1)
		return PENDING
	}
	if source.Hash() != block.SourceHash {
		stat.errMsg = fmt.Sprintf("block[%s][%d][%s] error, source hash[%s][%s] error.",
			block.Signer(), block.Height(), block.Hash(), block.SourceHash, source.Hash())
		return FAIL
	}

	if block.SnapshotHeight < source.SnapshotHeight {
		stat.errMsg = fmt.Sprintf("block[%s][%d][%s] error, [received snapshot height]%d must be greater or equal to [send snapshot height]%d.",
			block.Signer(), block.Height(), block.Hash(), block.SnapshotHeight, source.SnapshotHeight)
		return FAIL
	}
	if source.ModifiedAmount+block.ModifiedAmount == 0 {
		return SUCCESS
	} else {
		stat.errMsg = fmt.Sprintf("block[%s][%d][%s] error, modifiedAmount[%d][%d] cal fail.",
			block.Signer(), block.Height(), block.Hash(), source.ModifiedAmount, block.ModifiedAmount)
		return FAIL
	}
}
