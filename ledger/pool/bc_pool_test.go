package pool

import (
	"github.com/viteshan/naive-vite/common"
	"github.com/viteshan/naive-vite/common/log"
	"github.com/viteshan/naive-vite/syncer"
	"github.com/viteshan/naive-vite/verifier"
	"strconv"
	"testing"
	"time"
)

type TestVerifier struct {
}

type TestBlockVerifyStat struct {
	verifier.BlockVerifyStat
	result verifier.VerifyResult
}

func (self *TestBlockVerifyStat) VerifyResult() verifier.VerifyResult {
	return self.result
}

func (self *TestBlockVerifyStat) Reset() {
	self.result = verifier.PENDING
}

func (self *TestVerifier) VerifyReferred(block common.Block, stat verifier.BlockVerifyStat) {
	switch stat.(type) {
	case *TestBlockVerifyStat:
		testStat := stat.(*TestBlockVerifyStat)
		switch testStat.result {
		case verifier.NONE:
			testStat.result = verifier.PENDING
		case verifier.PENDING:
			testStat.result = verifier.SUCCESS
		}
	}
}

func (self *TestVerifier) NewVerifyStat(t verifier.VerifyType) verifier.BlockVerifyStat {
	return &TestBlockVerifyStat{result: verifier.NONE}
}

type TestBlock struct {
	hash    string
	height  int
	preHash string
	signer  string
}

func (self *TestBlock) Height() int {
	return self.height
}

func (self *TestBlock) Hash() string {
	return self.hash
}

func (self *TestBlock) PreHash() string {
	return self.preHash
}

func (self *TestBlock) Signer() string {
	return self.signer
}
func (self *TestBlock) String() string {
	return "height:[" + strconv.Itoa(self.height) + "]\thash:[" + self.hash + "]\tpreHash:[" + self.preHash + "]\tsigner:[" + self.signer + "]"
}

type TestSyncer struct {
	pool *BcPool
}

type TestChainReader struct {
	head common.Block
}

func (self *TestChainReader) Head() common.Block {
	return self.head
}
func (self *TestChainReader) insertChain(block common.Block, forkVersion int) (bool, error) {
	log.Info("insert to chain: %s", block)
	self.head = block
	return true, nil
}

func (self *TestSyncer) Fetch(hash syncer.BlockHash, prevCnt int) {
	log.Info("fetch request,cnt:%d, hash:%v", prevCnt, hash)
	go func() {
		for i := 1; i < prevCnt; i++ {
			height := hash.Height - i
			block := &TestBlock{hash: strconv.Itoa(height), height: height, preHash: strconv.Itoa(height - 1), signer: signer}
			log.Info("recv from net: %s", block)
			self.pool.addBlock(block)
		}
	}()
}

var signer = "viteshan"

func TestBcPool(t *testing.T) {

	reader := &TestChainReader{head: &TestBlock{hash: "0", height: 0, preHash: "-1", signer: signer}}

	testSyncer := &TestSyncer{}
	pool := newBlockchainPool(reader.insertChain, &TestVerifier{}, testSyncer, reader, "bcPool-1")
	testSyncer.pool = pool
	pool.init()
	pool.Start()
	pool.addBlock(&TestBlock{hash: "5", height: 5, preHash: "4", signer: signer})
	time.Sleep(time.Second)
	pool.addBlock(&TestBlock{hash: "1", height: 1, preHash: "0", signer: signer})
	time.Sleep(time.Second)
	pool.addBlock(&TestBlock{hash: "10", height: 10, preHash: "9", signer: signer})
	c := make(chan int)
	c <- 1
}

func TestInsertChain(t *testing.T) {
	reader := &TestChainReader{head: &TestBlock{hash: "1", height: 1, preHash: "0", signer: signer}}
	reader.insertChain(&TestBlock{hash: "1", height: 1, preHash: "0", signer: "viteshan"}, 1)
}
