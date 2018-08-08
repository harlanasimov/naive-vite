package ledger

import "github.com/viteshan/naive-vite/common"

// just only unreceived transactions

type Req struct {
	reqHash string
	acc     *common.AccountHashH
	state   int // 0:dirty  1:confirmed  2:unconfirmed

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
		req := &Req{reqHash: block.Hash(), state: 2}
		self.account(block.To).reqs[req.reqHash] = req
	} else if block.BlockType == common.RECEIVED {
		//delete(self.account(block.To).reqs, block.SourceHash)
		req := self.getReq(block.To, block.SourceHash)
		req.state = 1
		req.acc = &common.AccountHashH{Addr: block.To, Hash: block.Hash(), Height: block.Height()}
	}
}

func (self *reqPool) blockRollback(block *common.AccountStateBlock) {
	if block.BlockType == common.SEND {
		//delete(self.account(block.To).reqs, block.Hash())
		self.getReq(block.To, block.Hash()).state = 0
	} else if block.BlockType == common.RECEIVED {
		//req := &Req{reqHash: block.SourceHash}
		//self.account(block.To).reqs[req.reqHash] = req
		self.getReq(block.To, block.SourceHash).state = 2
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
func (self *reqPool) getReq(address string, sourceHash string) *Req {
	account := self.account(address)
	return account.reqs[sourceHash]
}

func (self *reqPool) confirmed(address string, sourceHash string) *common.AccountHashH {
	account := self.account(address)
	if account == nil {
		return nil
	}
	req := account.reqs[sourceHash]

	if req != nil && req.state == 1 {
		return req.acc
	}
	return nil
}
