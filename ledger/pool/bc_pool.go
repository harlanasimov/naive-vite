package pool

import (
	"sort"
	"sync"

	"github.com/viteshan/naive-vite/common"
	"github.com/viteshan/naive-vite/common/log"
	"github.com/viteshan/naive-vite/syncer"
	"github.com/viteshan/naive-vite/verifier"
	"github.com/viteshan/naive-vite/version"
	"time"
)

type chainReader interface {
	Head() common.Block
}

type BcPool struct {
	name        string
	pending     map[string]*BlockForPool
	waiting     *bcWaitting
	pendingMu   sync.Mutex
	syncer      syncer.Syncer
	verifier    verifier.Verifier
	chainReader chainReader
}

func newBlockchainPool(
	insertChainFn insertChainForkCheck,
	verifier verifier.Verifier,
	syncer syncer.Syncer,
	reader chainReader,
	name string) *BcPool {

	waiting := &bcWaitting{
		name:            name,
		waitingByHash:   make(map[string]*BlockForPool),
		waitingByHeight: make(map[int]*BlockForPool),
		insertChainFn:   insertChainFn,
		verifier:        verifier,
		reader:          reader,
	}
	return &BcPool{
		name:        name,
		pending:     make(map[string]*BlockForPool),
		waiting:     waiting,
		verifier:    verifier,
		syncer:      syncer,
		chainReader: reader,
	}
}

func (self *BcPool) init() {
	self.waiting.init()
}

type bcWaitting struct {
	name string
	// key:hash
	waitingByHash map[string]*BlockForPool
	// key:height
	waitingByHeight map[int]*BlockForPool
	headH           int
	headHash        string
	head            common.Block

	chainHeadH    int
	verifier      verifier.Verifier
	insertChainFn insertChainForkCheck
	reader        chainReader
}

type insertChainForkCheck func(block common.Block, forkVersion int) (bool, error)

type BlockForPool struct {
	block       common.Block
	verifyStat  verifier.BlockVerifyStat
	forkVersion int
}

func (self *BlockForPool) checkForkVersion() bool {
	forkVersion := version.ForkVersion()
	if self.forkVersion == -1 {
		self.forkVersion = forkVersion
		return true
	} else {
		return self.forkVersion == forkVersion
	}
}
func (self *BlockForPool) reset() {
	forkVersion := version.ForkVersion()
	self.verifyStat.Reset()
	self.forkVersion = forkVersion
}

func (self *bcWaitting) addLast(w *BlockForPool) {
	self.headHash = w.block.Hash()
	self.headH = w.block.Height()
	self.waitingByHash[w.block.Hash()] = w
	self.waitingByHeight[w.block.Height()] = w
	self.head = w.block
}
func (self *bcWaitting) insert(w *BlockForPool) (bool, error) {
	if w.block.Height() == self.headH+1 {
		if self.headHash == w.block.PreHash() {
			self.addLast(w)
			return true, nil
		} else {
			log.Warn("account chain fork, fork point height[%d],hash[%s], but next block[%s]'s preHash is [%s]",
				self.headH, self.headHash, w.block.Hash(), w.block.PreHash())
			return false, nil
		}
	} else {
		return false, nil
	}
}

func (self *bcWaitting) init() {
	head := self.reader.Head()
	self.chainHeadH = head.Height()
	self.headH = self.chainHeadH
	self.head = head
	self.headHash = head.Hash()
}

// Check insertion
func (self *bcWaitting) check() {
	minH := self.chainHeadH + 1
	headH := self.headH
L:
	for i := minH; i <= headH; i++ {
		wrapper := self.waitingByHeight[i]
		block := wrapper.block
		stat := wrapper.verifyStat
		if !wrapper.checkForkVersion() {
			wrapper.reset()
		}
		self.verifier.VerifyReferred(block, stat)
		if !wrapper.checkForkVersion() {
			wrapper.reset()
			continue
		}
		result := stat.VerifyResult()
		switch result {
		case verifier.PENDING:
		case verifier.FAIL:
			log.Error("Account chain forked. verify result is %d. block info:account[%s],hash[%s],height[%d]",
				result, block.Signer(), block.Hash(), block.Height())
			break L
		case verifier.SUCCESS:
			if block.Height() == self.chainHeadH+1 {
				self.writeToChain(wrapper)
			}
		default:
			log.Error("Unexpected things happened. verify result is %d. block info:account[%s],hash[%s],height[%d]",
				result, block.Signer(), block.Hash(), block.Height())
		}
	}
}
func (self *bcWaitting) writeToChain(wrapper *BlockForPool) {
	block := wrapper.block
	height := block.Height()
	hash := block.Hash()
	forkVersion := wrapper.forkVersion
	result, err := self.insertChainFn(block, forkVersion)
	if err == nil && result {
		delete(self.waitingByHeight, height)
		delete(self.waitingByHash, hash)
		self.chainHeadH = height
	} else {
		log.Error("waiting pool insert chain fail. height:[%d], hash:[%s]", height, hash)
	}
}
func (self *bcWaitting) contain(hash string, height int) bool {
	_, ok := self.waitingByHash[hash]
	return ok
}

func (self *BcPool) addBlock(block common.Block) {
	stat := self.verifier.NewVerifyStat(verifier.VerifyReferred)
	wrapper := &BlockForPool{block: block, verifyStat: stat, forkVersion: -1}
	self.pendingMu.Lock()
	defer self.pendingMu.Unlock()
	hash := block.Hash()
	height := block.Height()
	if !self.contain(hash, height) {
		self.pending[hash] = wrapper
	} else {
		log.Warn("block exists in BcPool. hash:[%s], height:[%d].", hash, height)
	}
}

type ByHeight []*BlockForPool

func (a ByHeight) Len() int           { return len(a) }
func (a ByHeight) Swap(i, j int)      { a[i], a[j] = a[j], a[i] }
func (a ByHeight) Less(i, j int) bool { return a[i].block.Height() < a[j].block.Height() }

func (self *BcPool) loop() {
	for {
		sortPending := copyValuesFrom(self.pending, &self.pendingMu)
		sort.Sort(ByHeight(sortPending))

		tryWait := true
		prev := self.waiting.head
		for _, w := range sortPending {
			block := w.block
			if block.Height() <= self.waiting.headH {
				continue
			}
			if tryWait {
				// try to append to waiting pool and remove from pending
				result, _ := self.inertToWaiting(w)
				tryWait = result
			}

			if !tryWait && prev != nil {
				// sync missing data
				diff := block.Height() - prev.Height()
				if diff > 1 {
					hash := syncer.BlockHash{Hash: block.Hash(), Height: block.Height()}
					self.syncer.Fetch(hash, diff)
				}
			}
			prev = block
		}
		self.waiting.check()
		time.Sleep(time.Second)
	}
}
func (self *BcPool) Start() {
	go self.loop()
}
func (self *BcPool) inertToWaiting(pool *BlockForPool) (bool, error) {
	result, err := self.waiting.insert(pool)
	if result {
		delete(self.pending, pool.block.Hash())
	}
	return result, err
}
func (self *BcPool) contain(hash string, height int) bool {
	_, ok := self.pending[hash]
	return self.waiting.contain(hash, height) || ok
}

func copyMap(m map[string]*BlockForPool, mutex *sync.Mutex) map[string]*BlockForPool {
	if mutex != nil {
		mutex.Lock()
		defer mutex.Unlock()
	}
	r := make(map[string]*BlockForPool)

	for k, v := range m {
		r[k] = v
	}
	return r
}
func copyValuesFrom(m map[string]*BlockForPool, mutex *sync.Mutex) []*BlockForPool {
	if mutex != nil {
		mutex.Lock()
		defer mutex.Unlock()
	}
	var r []*BlockForPool

	for _, v := range m {
		r = append(r, v)
	}
	return r
}
