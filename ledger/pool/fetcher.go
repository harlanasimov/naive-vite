package pool

import (
	"github.com/viteshan/naive-vite/common"
	"github.com/viteshan/naive-vite/syncer"
)

type fetcher struct {
	// if address == "", snapshot fetcher
	// else account fetcher
	address string
	fetcher syncer.Fetcher
}

func NewFetcher(address string, f syncer.Fetcher) *fetcher {
	self := &fetcher{}
	self.address = address
	self.fetcher = f
	return self
}

func (self *fetcher) fetch(hashHeight common.HashHeight, prevCnt int) {
	if self.address == "" {
		self.fetcher.FetchSnapshot(hashHeight, prevCnt)
	} else {
		self.fetcher.FetchAccount(self.address, hashHeight, prevCnt)
	}
}
