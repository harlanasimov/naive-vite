package test

import (
	"time"
	"strconv"
	"github.com/viteshan/naive-vite/common"
	"github.com/viteshan/naive-vite/common/log"
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
}


func (self *TestSyncer) Fetch(hash common.HashHeight, prevCnt int) {
	log.Info("fetch request,cnt:%d, hash:%v", prevCnt, hash)
}
