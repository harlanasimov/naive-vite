package syncer

import "strconv"

type Block struct {
	height int
	hash   string
	prev   string
}

func (self *Block) String() string {
	return "height[" + strconv.Itoa(self.height) + "]\thash:[" + self.hash + "]\tprev:[" + self.prev + "]"
}

type blockHandle interface {
	handle(blocks []Block)
}

type receiver struct {
	fetcher     *fetcher
	blockHandle blockHandle
}

func (self *receiver) handleHash(tasks []hashTask) {
	go self.fetcher.fetchBlockByHash(tasks)
}

func (self *receiver) handleBlock(blocks []Block) {
	for _, block := range blocks {
		self.fetcher.done(block.hash, block.height)
	}
	go self.blockHandle.handle(blocks)
}
