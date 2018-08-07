package ledger

import "github.com/viteshan/naive-vite/common"

// just only unreceived transactions

type Req struct {
	reqHash string
}

type reqPool struct {
	reqs map[string]*Req
}

func (self *reqPool) blockInsert(block *common.AccountStateBlock) {
	if block.BlockType == common.SEND {
		req := &Req{reqHash: block.Hash()}
		self.reqs[req.reqHash] = req
	} else if block.BlockType == common.RECEIVED {
		delete(self.reqs, block.SourceHash)
	}
}

func (self *reqPool) blockRollback(block *common.AccountStateBlock) {
	if block.BlockType == common.SEND {
		delete(self.reqs, block.Hash())
	} else if block.BlockType == common.RECEIVED {
		req := &Req{reqHash: block.SourceHash}
		self.reqs[req.reqHash] = req
	}
}
