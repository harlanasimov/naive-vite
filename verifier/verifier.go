package verifier

import (
	"time"

	"github.com/viteshan/naive-vite/common"
	"github.com/viteshan/naive-vite/common/face"
	"github.com/viteshan/naive-vite/version"
)

type VerifyResult int

const (
	PENDING VerifyResult = iota
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

type Callback func(block common.Block, stat BlockVerifyStat)
type Verifier interface {
	VerifyReferred(block common.Block) (BlockVerifyStat, Task)
}

type BlockVerifyStat interface {
	VerifyResult() VerifyResult
	ErrMsg() string
}

type Task interface {
	Done() bool
	Requests() []face.FetchRequest
}

type verifyTask struct {
	tasks   []Task
	version int
	reader  face.ChainReader
	v       *version.Version
	fail    bool
	t       time.Time
}

func (self *verifyTask) Requests() []face.FetchRequest {
	var reqs []face.FetchRequest
	for _, t := range self.tasks {
		for _, r := range t.Requests() {
			reqs = append(reqs, r)
		}
	}
	return reqs
}

func (self *verifyTask) Done() bool {
	// if version increase.  task has done.
	if self.v.Val() != self.version {
		return true
	}

	if time.Now().After(self.t.Add(time.Second * 5)) {
		return true
	}
	// if because fail, must wait for version increase.
	if self.fail {
		return false
	}
	// pending task check
	for _, t := range self.tasks {
		if !t.Done() {
			return false
		}
	}
	// all pending done
	return true
}

type accountPendingTask struct {
	reader  face.AccountReader
	result  bool
	request face.FetchRequest
}

func (self *accountPendingTask) Requests() []face.FetchRequest {
	var reqs []face.FetchRequest
	return append(reqs, self.request)
}
func (self *accountPendingTask) Done() bool {
	if self.result {
		return true
	}
	block, e := self.reader.HeadAccount(self.request.Chain)
	if e == nil && block != nil && block.Height() >= self.request.Height {
		self.result = true
		return true
	}
	return false
}

type snapshotPendingTask struct {
	reader  face.SnapshotReader
	result  bool
	request face.FetchRequest
}

func (self *snapshotPendingTask) Requests() []face.FetchRequest {
	var reqs []face.FetchRequest
	return append(reqs, self.request)
}

func (self *snapshotPendingTask) Done() bool {
	if self.result {
		return true
	}

	block, e := self.reader.HeadSnapshot()
	if e == nil && block != nil && block.Height() >= self.request.Height {
		self.result = true
		return true
	}
	return false
}

func (self *verifyTask) pendingSnapshot(hash string, height int) {
	request := face.FetchRequest{Chain: "", Hash: hash, Height: height, PrevCnt: 1}
	self.tasks = append(self.tasks, &snapshotPendingTask{self.reader, false, request})
}
func (self *verifyTask) pendingAccount(addr string, height int, hash string, prevCnt int) {
	request := face.FetchRequest{Chain: addr, Hash: hash, Height: height, PrevCnt: prevCnt}
	self.tasks = append(self.tasks, &accountPendingTask{self.reader, false, request})
}
