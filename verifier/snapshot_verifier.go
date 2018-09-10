package verifier

import (
	"fmt"

	"time"

	"github.com/viteshan/naive-vite/common"
	"github.com/viteshan/naive-vite/common/face"
	"github.com/viteshan/naive-vite/version"
)

type SnapshotVerifier struct {
	reader face.ChainReader
	v      *version.Version
}

func NewSnapshotVerifier(r face.ChainReader, v *version.Version) *SnapshotVerifier {
	verifier := &SnapshotVerifier{reader: r, v: v}
	return verifier
}

func (self *SnapshotVerifier) VerifyReferred(b common.Block) BlockVerifyStat {
	block := b.(*common.SnapshotBlock)
	stat := self.newVerifyStat(VerifyReferred, block)
	accounts := block.Accounts

	task := &verifyTask{v: self.v, version: self.v.Val(), reader: self.reader, t: time.Now()}

	i := 0
	for _, v := range accounts {
		addr := v.Addr
		hash := v.Hash
		height := v.Height
		block := self.reader.GetAccountByHeight(addr, height)
		if block == nil {
			stat.results[v.Addr] = PENDING
			task.pendingAccount(v.Addr, v.Height, v.Hash, 1)
		} else {
			if block.Hash() == hash {
				i++
				stat.results[v.Addr] = SUCCESS
			} else {
				stat.errMsg = fmt.Sprintf("account block[%s][%d][%s] error.",
					v.Addr, v.Height, v.Hash)
				stat.results[v.Addr] = FAIL
				stat.result = FAIL
				return stat
			}
		}
	}
	if i == len(accounts) {
		stat.result = SUCCESS
		return stat
	}
	stat.task = task
	return stat
}

type SnapshotBlockVerifyStat struct {
	result   VerifyResult
	accounts []*common.AccountHashH
	results  map[string]VerifyResult
	errMsg   string
	task     Task
}

func (self *SnapshotBlockVerifyStat) Task() Task {
	if self.task == nil {
		return nil
	} else {
		return self.task
	}
}

func (self *SnapshotBlockVerifyStat) ErrMsg() string {
	return self.errMsg
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

func (self *SnapshotVerifier) newVerifyStat(t VerifyType, b common.Block) *SnapshotBlockVerifyStat {
	block := b.(*common.SnapshotBlock)

	stat := &SnapshotBlockVerifyStat{result: PENDING, accounts: block.Accounts}
	stat.results = make(map[string]VerifyResult)
	return stat
}
