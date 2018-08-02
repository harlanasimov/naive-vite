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

type Verifier interface {
	//verifyParent(parent *AccountStateBlock, block *AccountStateBlock) VerifyResult
	VerifyReferred(block common.Block, stat BlockVerifyStat)
	NewVerifyStat(t VerifyType) BlockVerifyStat
}

type BlockVerifyStat interface {
	VerifyResult() VerifyResult
	Reset()
}
