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

func (self *TestVerifier) NewVerifyStat(t verifier.VerifyType, block common.Block) verifier.BlockVerifyStat {
	return &TestBlockVerifyStat{result: verifier.NONE}
}

type TestBlock struct {
	hash      string
	height    int
	preHash   string
	signer    string
	timestamp time.Time
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
func (self *TestBlock) Timestamp() time.Time {
	return self.timestamp
}
func (self *TestBlock) String() string {
	return "height:[" + strconv.Itoa(self.height) + "]\thash:[" + self.hash + "]\tpreHash:[" + self.preHash + "]\tsigner:[" + self.signer + "]"
}

type TestSyncer struct {
	pool   *BCPool
	blocks map[string]*TestBlock
}

type TestChainReader struct {
	head  common.Block
	store map[int]common.Block
}

func (self *TestChainReader) GetBlock(height int) common.Block {
	return self.store[height]
}

func (self *TestChainReader) init() {
	self.store = make(map[int]common.Block)
	self.head = genesis
	self.store[genesis.height] = genesis
}

func (self *TestChainReader) Head() common.Block {
	return self.head
}
func (self *TestChainReader) insertChain(block common.Block, forkVersion int) (bool, error) {
	log.Info("insert to forkedChain: %s", block)
	self.head = block
	self.store[block.Height()] = block
	return true, nil
}

func (self *TestChainReader) removeChain(block common.Block) (bool, error) {
	log.Info("remove from forkedChain: %s", block)
	self.head = self.store[block.Height()-1]
	delete(self.store, block.Height())
	return true, nil
}
func (self *TestSyncer) Fetch(hash syncer.BlockHash, prevCnt int) {
	log.Info("fetch request,cnt:%d, hash:%v", prevCnt, hash)
	go func() {
		prev := hash.Hash

		for i := 0; i < prevCnt; i++ {
			block, ok := self.blocks[prev]
			if ok {
				log.Info("recv from net: %s", block)
				self.pool.AddBlock(block)
			} else {
				return
			}
			prev = block.preHash
		}

	}()
}

func (self *TestSyncer) genLinkedData() {
	self.blocks = genLinkBlock("A-", 1, 100, genesis)
	block := self.blocks["A-5"]
	tmp := genLinkBlock("B-", 6, 30, block)
	for k, v := range tmp {
		self.blocks[k] = v
	}

	block = self.blocks["A-6"]
	tmp = genLinkBlock("C-", 7, 30, block)
	for k, v := range tmp {
		self.blocks[k] = v
	}
}

func genLinkBlock(mark string, start int, end int, genesis *TestBlock) map[string]*TestBlock {
	blocks := make(map[string]*TestBlock)
	last := genesis
	for i := start; i < end; i++ {
		hash := mark + strconv.Itoa(i)
		block := &TestBlock{hash: hash, height: i, preHash: last.Hash(), signer: signer}
		blocks[hash] = block
		last = block
	}
	return blocks
}

var genesis = &TestBlock{hash: "A-0", height: 0, preHash: "-1", signer: signer, timestamp: time.Now()}

var signer = "viteshan"

func TestBcPool(t *testing.T) {

	reader := &TestChainReader{head: genesis}
	reader.init()
	testSyncer := &TestSyncer{blocks: make(map[string]*TestBlock)}
	testSyncer.genLinkedData()
	pool := newBlockChainPool("bcPool-1")
	testSyncer.pool = pool
	pool.init(reader.insertChain, reader.removeChain, &TestVerifier{}, testSyncer, reader)
	go pool.loop()
	pool.AddBlock(&TestBlock{hash: "A-6", height: 6, preHash: "A-5", signer: signer, timestamp: time.Now()})
	time.Sleep(time.Second)
	pool.AddBlock(&TestBlock{hash: "C-10", height: 10, preHash: "C-9", signer: signer, timestamp: time.Now()})
	time.Sleep(time.Second)
	pool.AddBlock(&TestBlock{hash: "A-1", height: 1, preHash: "A-0", signer: signer, timestamp: time.Now()})
	time.Sleep(time.Second)

	pool.AddBlock(&TestBlock{hash: "A-20", height: 20, preHash: "A-19", signer: signer, timestamp: time.Now()})
	pool.AddBlock(&TestBlock{hash: "B-9", height: 9, preHash: "A-8", signer: signer, timestamp: time.Now()})
	c := make(chan int)
	c <- 1
}

func TestInsertChain(t *testing.T) {
	reader := &TestChainReader{head: &TestBlock{hash: "1", height: 1, preHash: "0", signer: signer, timestamp: time.Now()}}
	reader.insertChain(&TestBlock{hash: "1", height: 1, preHash: "0", signer: "viteshan", timestamp: time.Now()}, 1)
}
