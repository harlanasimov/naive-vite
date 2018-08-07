package ledger

import "github.com/viteshan/naive-vite/common"

// just only unreceived transactions

type Req struct {
	reqHash string
}

type reqPool struct {
	accounts map[string]*reqAccountPool
}

type reqAccountPool struct {
	reqs map[string]*Req
}

func newReqPool() *reqPool {
	pool := &reqPool{}
	pool.accounts = make(map[string]*reqAccountPool)
	return pool
}

func (self *reqPool) blockInsert(block *common.AccountStateBlock) {
	if block.BlockType == common.SEND {
		req := &Req{reqHash: block.Hash()}
		self.account(block.To).reqs[req.reqHash] = req
	} else if block.BlockType == common.RECEIVED {
		delete(self.account(block.To).reqs, block.SourceHash)
	}
}

func (self *reqPool) blockRollback(block *common.AccountStateBlock) {
	if block.BlockType == common.SEND {
		delete(self.account(block.To).reqs, block.Hash())
	} else if block.BlockType == common.RECEIVED {
		req := &Req{reqHash: block.SourceHash}
		self.account(block.To).reqs[req.reqHash] = req
	}
}

func (self *reqPool) account(address string) *reqAccountPool {
	pool := self.accounts[address]
	if pool == nil {
		pool = &reqAccountPool{reqs: make(map[string]*Req)}
		self.accounts[address] = pool
	}
	return pool
}

func (self *reqPool) getReqs(address string) []*Req {
	account := self.account(address)
	result := make([]*Req, len(account.reqs))
	i := 0
	for _, req := range account.reqs {
		result[i] = req
		i++
	}
	return result
}
