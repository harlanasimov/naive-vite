package pool

import (
	"sort"
	"sync"

	"fmt"
	"github.com/viteshan/naive-vite/common"
	"github.com/viteshan/naive-vite/common/log"
	"github.com/viteshan/naive-vite/syncer"
	"github.com/viteshan/naive-vite/verifier"
	"github.com/viteshan/naive-vite/version"
	"strconv"
	"sync/atomic"
	"time"
)

type chainReader interface {
	Head() common.Block
	getBlock(height int) common.Block
}

type heightChainReader interface {
	id() string
	getBlock(height int, refer bool) *BlockForPool
}

var pendingMu sync.Mutex

type BCPool struct {
	name        string
	blockpool   *blockPool
	chainpool   *chainPool
	syncer      syncer.Syncer
	verifier    verifier.Verifier
	chainReader chainReader
}

type blockPool struct {
	freeBlocks     map[string]*BlockForPool // free state
	compoundBlocks map[string]*BlockForPool // compound state
}
type chainPool struct {
	poolId          string
	lastestChainIdx int32
	current         *forkedChain
	snippetChains   map[string]*snippetChain // head is fixed
	chains          map[string]*forkedChain
	verifier        verifier.Verifier
	insertChainFn   insertChainForkCheck
	diskChain       *diskChain
	//reader          chainReader
}

func (self *chainPool) forkChain(forked *forkedChain, snippet *snippetChain) (*forkedChain, error) {
	new := &forkedChain{}

	new.heightBlocks = snippet.heightBlocks
	new.tailHeight = snippet.tailHeight
	new.headHeight = snippet.headHeight
	new.headHash = snippet.headHash
	new.referChain = forked

	new.chainId = self.genChainId()
	self.chains[new.chainId] = new
	return new, nil
}

type diskChain struct {
	reader chainReader
}

func (self *diskChain) getBlock(height int, refer bool) *BlockForPool {
	forkVersion := version.ForkVersion()
	block := self.reader.getBlock(height)
	if block == nil {
		return nil
	} else {
		return &BlockForPool{block: block, forkVersion: forkVersion, verifyStat: &BlockVerifySuccessStat{}}
	}

}
func (self *diskChain) id() string {
	return "diskChain"
}

type BlockVerifySuccessStat struct {
}

func (self *BlockVerifySuccessStat) Reset() {
}

func (self *BlockVerifySuccessStat) VerifyResult() verifier.VerifyResult {
	return verifier.SUCCESS
}

type chain struct {
	heightBlocks map[int]*BlockForPool
	headHeight   int //  forkedChain size is zero when headHeight==tailHeight
	tailHeight   int
	chainId      string
}

func (self *chain) id() string {
	return self.chainId
}

type snippetChain struct {
	chain
	tailHash string
	headHash string
}

func (self *snippetChain) init(w *BlockForPool) {
	self.heightBlocks = make(map[int]*BlockForPool)
	self.headHeight = w.block.Height()
	self.headHash = w.block.Hash()
	self.tailHash = w.block.PreHash()
	self.tailHeight = w.block.Height() - 1
	self.heightBlocks[w.block.Height()] = w
}

func (self *snippetChain) addTail(w *BlockForPool) {
	self.tailHash = w.block.PreHash()
	self.tailHeight = w.block.Height() - 1
	self.heightBlocks[w.block.Height()] = w
}
func (self *snippetChain) deleteTail(newtail *BlockForPool) {
	self.tailHash = newtail.block.Hash()
	self.tailHeight = newtail.block.Height()
	delete(self.heightBlocks, newtail.block.Height())
}

func (self *snippetChain) merge(snippet *snippetChain) {
	self.tailHeight = snippet.tailHeight
	self.tailHash = snippet.tailHash
	for k, v := range snippet.heightBlocks {
		self.heightBlocks[k] = v
	}
}

type forkedChain struct {
	chain
	// key: height
	headHash   string
	referChain heightChainReader
}

func (self *forkedChain) getBlock(height int, refer bool) *BlockForPool {
	block, ok := self.heightBlocks[height]
	if ok {
		return block
	}
	if refer {
		return self.referChain.getBlock(height, refer)
	}
	return nil
}

func newBlockChainPool(
	insertChainFn insertChainForkCheck,
	verifier verifier.Verifier,
	syncer syncer.Syncer,
	reader chainReader,
	name string) *BCPool {

	//waiting := &bcWaitting{
	//	name:            name,
	//	waitingByHash:   make(map[string]*BlockForPool),
	//	waitingByHeight: make(map[int]*BlockForPool),
	//	insertChainFn:   insertChainFn,
	//	verifier:        verifier,
	//	reader:          reader,
	//}

	diskChain := &diskChain{reader: reader}
	chainpool := &chainPool{
		poolId:        "chainPool",
		insertChainFn: insertChainFn,
		verifier:      verifier,
		diskChain:     diskChain,
	}
	chainpool.current = &forkedChain{}

	chainpool.current.chainId = chainpool.genChainId()
	blockpool := &blockPool{
		compoundBlocks: make(map[string]*BlockForPool),
		freeBlocks:     make(map[string]*BlockForPool),
	}
	return &BCPool{
		name: name,

		chainpool: chainpool,
		blockpool: blockpool,

		verifier:    verifier,
		syncer:      syncer,
		chainReader: reader,
	}
}

func (self *chainPool) genChainId() string {
	return self.poolId + "-" + strconv.Itoa(self.incChainIdx())
}

func (self *chainPool) incChainIdx() int {
	for {
		old := self.lastestChainIdx
		new := old + 1
		if atomic.CompareAndSwapInt32(&self.lastestChainIdx, old, new) {
			return int(new)
		} else {
			log.Info("lastest forkedChain idx concurrent for %d.", old)
		}
	}
}
func (self *chainPool) init(initBlock common.Block) {
	self.current.init(initBlock)
	self.current.referChain = self.diskChain
	self.chains = make(map[string]*forkedChain)
	self.snippetChains = make(map[string]*snippetChain)
	self.chains[self.current.chainId] = self.current
}

// fork, insert, forkedChain
//func (self *chainPool) forky(wrapper *BlockForPool, chains []*forkedChain) (bool, bool, *forkedChain) {
//	block := wrapper.block
//	bHeight := block.Height()
//	bPreHash := block.PreHash()
//	bHash := block.Hash()
//	for _, c := range chains {
//		if bHeight == c.headHeight+1 && bPreHash == c.headHash {
//			return false, true, c
//		}
//		//bHeight <= c.tailHeight
//		if bHeight > c.headHeight {
//			continue
//		}
//		pre := c.getBlock(bHeight - 1)
//		uncle := c.getBlock(bHeight)
//		if pre != nil &&
//			uncle != nil &&
//			pre.block.Hash() == bPreHash &&
//			uncle.block.Hash() != bHash {
//			return true, false, c
//		}
//	}
//	return false, false, nil
//}

func (self *chainPool) forky(snippet *snippetChain, chains []*forkedChain) (bool, bool, *forkedChain) {
	for _, c := range chains {
		tailHeight := snippet.tailHeight
		tailHash := snippet.tailHash
		if tailHeight == c.headHeight && tailHash == c.headHash {
			return false, true, c
		}
		//bHeight <= c.tailHeight
		if tailHeight > c.headHeight {
			continue
		}
		point := findForkPoint(snippet, c)
		if point != nil {
			return true, false, c
		}
		if snippet.headHeight == snippet.tailHeight {
			delete(self.snippetChains, snippet.id())
			return false, false, nil
		}
	}
	point := findForkPoint(snippet, self.diskChain)
	if point != nil {
		return true, false, self.current
	}
	if snippet.headHeight == snippet.tailHeight {
		delete(self.snippetChains, snippet.id())
		return false, false, nil
	}
	return false, false, nil
}
func findForkPoint(snippet *snippetChain, chain heightChainReader) *BlockForPool {
	tailHeight := snippet.tailHeight
	headHeight := snippet.headHeight

	forkpoint := chain.getBlock(tailHeight, false)
	if forkpoint == nil {
		return nil
	}
	if forkpoint.block.Hash() != snippet.tailHash {
		return nil
	}

	for i := tailHeight + 1; i <= headHeight; i++ {
		uncle := chain.getBlock(i, false)
		if uncle == nil {
			log.Error("chain error. chain:%s", chain)
			return nil
		}
		point := snippet.heightBlocks[i]
		if point.block.Hash() != uncle.block.Hash() {
			return forkpoint
		} else {
			snippet.deleteTail(point)
			forkpoint = point
			continue
		}
	}
	return nil
}

func (self *chainPool) insertSnippet(c *forkedChain, snippet *snippetChain) error {
	for i := snippet.tailHeight + 1; i <= snippet.headHeight; i++ {
		w := snippet.heightBlocks[i]
		err := self.insert(c, w)
		if err != nil {
			return err
		} else {
			delete(snippet.heightBlocks, i)
			snippet.tailHeight = w.block.Height()
			snippet.tailHash = w.block.Hash()
		}
	}
	if snippet.tailHeight == snippet.headHeight {
		delete(self.snippetChains, snippet.chainId)
	}
	return nil
}

type ForkChainError struct {
	What string
}

func (e ForkChainError) Error() string {
	return fmt.Sprintf("%s", e.What)
}
func (self *chainPool) insert(c *forkedChain, wrapper *BlockForPool) error {
	if wrapper.block.Height() == c.headHeight+1 {
		if c.headHash == wrapper.block.PreHash() {
			c.addHead(wrapper)
			return nil
		} else {
			log.Warn("account forkedChain fork, fork point height[%d],hash[%s], but next block[%s]'s preHash is [%s]",
				c.headHeight, c.headHash, wrapper.block.Hash(), wrapper.block.PreHash())
			return &ForkChainError{What: "fork chain."}
		}
	} else {
		log.Warn("account forkedChain fork, fork point height[%d],hash[%s], but next block[%s]'s preHash is [%s]",
			c.headHeight, c.headHash, wrapper.block.Hash(), wrapper.block.PreHash())
		return &ForkChainError{What: "fork chain."}
	}
}
func (self *chainPool) check() {

	self.printChains()

	current := self.current
	minH := current.tailHeight + 1
	headH := current.headHeight
L:
	for i := minH; i <= headH; i++ {
		wrapper := current.getBlock(i, false)
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
			log.Error("Account forkedChain forked. verify result is %d. block info:account[%s],hash[%s],height[%d]",
				result, block.Signer(), block.Hash(), block.Height())
			break L
		case verifier.SUCCESS:
			if block.Height() == current.tailHeight+1 {
				self.writeToChain(current, wrapper)
			}
		default:
			log.Error("Unexpected things happened. verify result is %d. block info:account[%s],hash[%s],height[%d]",
				result, block.Signer(), block.Hash(), block.Height())
		}
	}
}
func (self *chainPool) writeToChain(chain *forkedChain, wrapper *BlockForPool) {
	block := wrapper.block
	height := block.Height()
	hash := block.Hash()
	forkVersion := wrapper.forkVersion
	result, err := self.insertChainFn(block, forkVersion)
	if err == nil && result {
		delete(chain.heightBlocks, height)
		chain.tailHeight = height
		self.fixReferChange(chain, height)
	} else {
		log.Error("waiting pool insert forkedChain fail. height:[%d], hash:[%s]", height, hash)
	}
}
func (self *chainPool) printChains() {
	log.Info("----------------------------------")
	chains := copyChains(self.chains)
	for _, c := range chains {
		if c.chainId == self.current.chainId {
			log.Info("%s [current]", c)
		} else {
			log.Info("%s ", c)
		}

	}
}
func (self *chainPool) fixReferChange(target *forkedChain, fixHeight int) {
	targetId := target.id()
	for id, chain := range self.chains {
		if chain.referChain.id() == targetId && chain.tailHeight <= fixHeight {
			chain.referChain = self.diskChain
			log.Info("forkedChain[%s] reset refer from %s because of %d, refer to disk.", id, targetId, fixHeight)
		}
	}
}

func (self *forkedChain) init(initBlock common.Block) {
	self.heightBlocks = make(map[int]*BlockForPool)
	self.tailHeight = initBlock.Height()
	self.headHeight = initBlock.Height()
	self.headHash = initBlock.Hash()
}

//func (self *forkedChain) copy(maxHeight int, maxHash string) *forkedChain {
//	copyHeightBlocks := make(map[int]*BlockForPool)
//
//	tail := self.tailHeight
//	for i := tail + 1; i <= maxHeight; i++ {
//		tmp := self.getBlock(i)
//		copyHeightBlocks[i] = tmp
//	}
//	chain := &forkedChain{}
//	chain.heightBlocks = copyHeightBlocks
//	chain.tailHeight = tail
//	chain.headHeight = maxHeight
//	chain.headHash = maxHash
//	chain.referChain = self
//	return chain
//
//}
func (self *forkedChain) addHead(w *BlockForPool) {
	self.headHash = w.block.Hash()
	self.headHeight = w.block.Height()
	self.heightBlocks[w.block.Height()] = w
}

func (self *forkedChain) String() string {
	return self.chainId + "\t" + strconv.Itoa(self.headHeight) + "[" + self.headHash + "]\t" + strconv.Itoa(self.tailHeight)
}

func (self *BCPool) init() {
	head := self.chainReader.Head()
	self.chainpool.init(head)
}

//
//type bcWaitting struct {
//	name string
//	// key:hash
//	waitingByHash map[string]*BlockForPool
//	// key:height
//	waitingByHeight map[int]*BlockForPool
//	headH           int
//	headHash        string
//	head            common.Block
//
//	chainHeadH    int
//	verifier      verifier.Verifier
//	insertChainFn insertChainForkCheck
//	reader        chainReader
//}

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

//func (self *bcWaitting) addHead(w *BlockForPool) {
//	self.headHash = w.block.Hash()
//	self.headH = w.block.Height()
//	self.waitingByHash[w.block.Hash()] = w
//	self.waitingByHeight[w.block.Height()] = w
//	self.head = w.block
//}
//func (self *bcWaitting) insert(w *BlockForPool) (bool, error) {
//	if w.block.Height() == self.headH+1 {
//		if self.headHash == w.block.PreHash() {
//			self.addHead(w)
//			return true, nil
//		} else {
//			log.Warn("account forkedChain fork, fork point height[%d],hash[%s], but next block[%s]'s preHash is [%s]",
//				self.headH, self.headHash, w.block.Hash(), w.block.PreHash())
//			return false, nil
//		}
//	} else {
//		return false, nil
//	}
//}
//
//func (self *bcWaitting) init() {
//	head := self.reader.Head()
//	self.chainHeadH = head.Height()
//	self.headH = self.chainHeadH
//	self.head = head
//	self.headHash = head.Hash()
//}
//
//// Check insertion
//func (self *bcWaitting) check() {
//	minH := self.chainHeadH + 1
//	headH := self.headH
//L:
//	for i := minH; i <= headH; i++ {
//		wrapper := self.waitingByHeight[i]
//		block := wrapper.block
//		stat := wrapper.verifyStat
//		if !wrapper.checkForkVersion() {
//			wrapper.reset()
//		}
//		self.verifier.VerifyReferred(block, stat)
//		if !wrapper.checkForkVersion() {
//			wrapper.reset()
//			continue
//		}
//		result := stat.VerifyResult()
//		switch result {
//		case verifier.PENDING:
//		case verifier.FAIL:
//			log.Error("Account forkedChain forked. verify result is %d. block info:account[%s],hash[%s],height[%d]",
//				result, block.Signer(), block.Hash(), block.Height())
//			break L
//		case verifier.SUCCESS:
//			if block.Height() == self.chainHeadH+1 {
//				self.writeToChain(wrapper)
//			}
//		default:
//			log.Error("Unexpected things happened. verify result is %d. block info:account[%s],hash[%s],height[%d]",
//				result, block.Signer(), block.Hash(), block.Height())
//		}
//	}
//}
//func (self *bcWaitting) writeToChain(wrapper *BlockForPool) {
//	block := wrapper.block
//	height := block.Height()
//	hash := block.Hash()
//	forkVersion := wrapper.forkVersion
//	result, err := self.insertChainFn(block, forkVersion)
//	if err == nil && result {
//		delete(self.waitingByHeight, height)
//		delete(self.waitingByHash, hash)
//		self.chainHeadH = height
//	} else {
//		log.Error("waiting pool insert forkedChain fail. height:[%d], hash:[%s]", height, hash)
//	}
//}
func (self *blockPool) contains(hash string, height int) bool {
	_, free := self.freeBlocks[hash]
	_, compound := self.compoundBlocks[hash]
	return free || compound
}
func (self *blockPool) putBlock(hash string, pool *BlockForPool) {
	self.freeBlocks[hash] = pool
}
func (self *blockPool) compound(w *BlockForPool) {
	pendingMu.Lock()
	defer pendingMu.Unlock()
	self.compoundBlocks[w.block.Hash()] = w
	delete(self.freeBlocks, w.block.Hash())
}

func (self *BCPool) addBlock(block common.Block) {
	stat := self.verifier.NewVerifyStat(verifier.VerifyReferred)
	wrapper := &BlockForPool{block: block, verifyStat: stat, forkVersion: -1}
	pendingMu.Lock()
	defer pendingMu.Unlock()
	hash := block.Hash()
	height := block.Height()
	if !self.blockpool.contains(hash, height) {
		self.blockpool.putBlock(hash, wrapper)
	} else {
		log.Warn("block exists in BCPool. hash:[%s], height:[%d].", hash, height)
	}
}

type ByHeight []*BlockForPool

func (a ByHeight) Len() int           { return len(a) }
func (a ByHeight) Swap(i, j int)      { a[i], a[j] = a[j], a[i] }
func (a ByHeight) Less(i, j int) bool { return a[i].block.Height() < a[j].block.Height() }

type ByTailHeight []*snippetChain

func (a ByTailHeight) Len() int           { return len(a) }
func (a ByTailHeight) Swap(i, j int)      { a[i], a[j] = a[j], a[i] }
func (a ByTailHeight) Less(i, j int) bool { return a[i].tailHeight < a[j].tailHeight }

func (self *BCPool) loop() {
	for {
		self.loopForsnippetChains()

		sortSnippets := copyMap(self.chainpool.snippetChains)
		sort.Sort(ByTailHeight(sortSnippets))

		tmpChains := copyChains(self.chainpool.chains)

		for _, w := range sortSnippets {
			forky, insertable, c := self.chainpool.forky(w, tmpChains)
			if forky {
				newChain, err := self.chainpool.forkChain(c, w)
				if err == nil {
					//err = self.chainpool.insertSnippet(newChain, w)
					tmpChains = append(tmpChains, newChain)
					delete(self.chainpool.snippetChains, w.id())
				}
				continue
			}
			if insertable {
				err := self.chainpool.insertSnippet(c, w)
				if err != nil {
					log.Error("insert fail. %v", err)
				}
				continue
			}
		}

		sortSnippets = copyMap(self.chainpool.snippetChains)
		sort.Sort(ByTailHeight(sortSnippets))

		prev := -1
		for _, w := range sortSnippets {
			diff := 5
			if prev > 0 {
				diff = w.tailHeight - prev
			}

			if diff < 0 {
				diff = 5
			}

			if diff > 1 {
				hash := syncer.BlockHash{Hash: w.tailHash, Height: w.tailHeight}
				self.syncer.Fetch(hash, diff)
			}
			prev = w.headHeight
		}
		self.chainpool.check()
		time.Sleep(time.Second)

	}
}
func (self *BCPool) Start() {
	go self.loop()
}
func (self *BCPool) loopForsnippetChains() {
	//  self.chainpool.snippetChains
	sortPending := copyValuesFrom(self.blockpool.freeBlocks)
	sort.Reverse(ByHeight(sortPending))

	chains := copyMap(self.chainpool.snippetChains)

	for _, v := range sortPending {
		if !tryInsert(chains, v) {
			snippet := &snippetChain{}
			snippet.chainId = self.chainpool.genChainId()
			snippet.init(v)
			chains = append(chains, snippet)
		}
		self.blockpool.compound(v)
	}

	headMap := splitToMap(chains)
	for _, chain := range chains {
		for {
			tail := chain.tailHash
			oc, ok := headMap[tail]
			if ok && chain.id() != oc.id() {
				delete(headMap, tail)
				chain.merge(oc)
			} else {
				break
			}
		}
	}
	final := make(map[string]*snippetChain)
	for _, v := range headMap {
		final[v.id()] = v
	}
	self.chainpool.snippetChains = final
}

func splitToMap(chains []*snippetChain) map[string]*snippetChain {
	headMap := make(map[string]*snippetChain)
	//tailMap := make(map[string]*snippetChain)
	for _, chain := range chains {
		head := chain.headHash
		//tail := chain.tailHash
		headMap[head] = chain
		//tailMap[tail] = chain
	}
	return headMap
}

func tryInsert(chains []*snippetChain, pool *BlockForPool) bool {
	for _, c := range chains {
		if c.tailHash == pool.block.Hash() {
			c.addTail(pool)
			return true
		}
		height := pool.block.Height()
		if c.headHeight > height && height > c.tailHeight {
			return true
		}
	}
	return false
}

func copyMap(m map[string]*snippetChain) []*snippetChain {
	var s []*snippetChain
	for _, v := range m {
		s = append(s, v)
	}
	return s
}
func copyValuesFrom(m map[string]*BlockForPool) []*BlockForPool {
	pendingMu.Lock()
	defer pendingMu.Unlock()
	var r []*BlockForPool

	for _, v := range m {
		r = append(r, v)
	}
	return r
}

func copyChains(m map[string]*forkedChain) []*forkedChain {
	var r []*forkedChain

	for _, v := range m {
		r = append(r, v)
	}
	return r
}
