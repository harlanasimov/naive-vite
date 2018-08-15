package test

import (
	"strconv"
	"time"

	"github.com/viteshan/naive-vite/common"
	"github.com/viteshan/naive-vite/common/log"
	"github.com/viteshan/naive-vite/syncer"
)

type TestBlock struct {
	Thash      string
	Theight    int
	TpreHash   string
	Tsigner    string
	Ttimestamp time.Time
}

func (self *TestBlock) Height() int {
	return self.Theight
}

func (self *TestBlock) Hash() string {
	return self.Thash
}

func (self *TestBlock) PreHash() string {
	return self.TpreHash
}

func (self *TestBlock) Signer() string {
	return self.Tsigner
}
func (self *TestBlock) Timestamp() time.Time {
	return self.Ttimestamp
}
func (self *TestBlock) String() string {
	return "Theight:[" + strconv.Itoa(self.Theight) + "]\tThash:[" + self.Thash + "]\tTpreHash:[" + self.TpreHash + "]\tTsigner:[" + self.Tsigner + "]"
}

type TestSyncer struct {
	Blocks map[string]*TestBlock
	f      syncer.Fetcher
}

func NewTestSync() *TestSyncer {
	testSyncer := &TestSyncer{Blocks: make(map[string]*TestBlock)}
	testSyncer.f = &TestFetcher{}
	return testSyncer
}

func (self *TestSyncer) Fetcher() syncer.Fetcher {
	return self.f
}

func (self *TestSyncer) Sender() syncer.Sender {
	panic("implement me")
}

func (self *TestSyncer) Handlers() syncer.Handlers {
	panic("implement me")
}

type TestFetcher struct {
}

func (*TestFetcher) FetchAccount(address string, hash common.HashHeight, prevCnt int) {
	log.Info("fetch request,cnt:%d, hash:%v", prevCnt, hash)
}

func (*TestFetcher) FetchSnapshot(hash common.HashHeight, prevCnt int) {
	log.Info("fetch request,cnt:%d, hash:%v", prevCnt, hash)
}
