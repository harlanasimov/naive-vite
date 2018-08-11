package syncer

import (
	"github.com/viteshan/naive-vite/common"
)


type blockHandle interface {
	handle(blocks []common.Block)
}

type receiver struct {
	fetcher     *fetcher
	blockHandle blockHandle
}

func (self *receiver) handleHash(tasks []hashTask) {
	go self.fetcher.fetchBlockByHash(tasks)
}

func (self *receiver) handleBlock(blocks []common.Block) {
	for _, block := range blocks {
		self.fetcher.done(block.Hash(), block.Height())
	}
	go self.blockHandle.handle(blocks)
}
