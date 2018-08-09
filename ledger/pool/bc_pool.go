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

type PoolReader interface {
	Current() Chain
	Chains() []Chain
}
type Chain interface {
	HeadHeight() int
	ChainId() string
	Head() common.Block
	GetBlock(height int) common.Block
}

type ChainReader interface {
	Head() common.Block
	GetBlock(height int) common.Block
}

type heightChainReader interface {
	id() string
	getBlock(height int, refer bool) *BlockForPool
	contains(height int) bool
}

var pendingMu sync.Mutex

type BCPool struct {
	Id        string
	blockpool *blockPool
	chainpool *chainPool
	syncer    syncer.Syncer
	verifier  verifier.Verifier
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
	removeChainFn   removeChainForkCheck
	diskChain       *diskChain
	//reader          Chain
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
	reader  ChainReader
	chainId string
}

func (self *diskChain) getBlock(height int, refer bool) *BlockForPool {
	forkVersion := version.ForkVersion()
	block := self.reader.GetBlock(height)
	if block == nil {
		return nil
	} else {
		return &BlockForPool{block: block, forkVersion: forkVersion, verifyStat: &BlockVerifySuccessStat{}}
	}
}
func (self *diskChain) getBlockBetween(tail int, head int, refer bool) *BlockForPool {
	//forkVersion := version.ForkVersion()
	//block := self.reader.GetBlock(height)
	//if block == nil {
	//	return nil
	//} else {
	//	return &BlockForPool{block: block, forkVersion: forkVersion, verifyStat: &BlockVerifySuccessStat{}}
	//}
	return nil
}

func (self *diskChain) contains(height int) bool {
	return self.reader.Head().Height() >= height
}

func (self *diskChain) id() string {
	return self.chainId
}

func (self *diskChain) Head() common.Block {
	head := self.reader.Head()
	if head == nil {
		return self.reader.GetBlock(-1) // hack implement
	}
	return head
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

func (self *chain) HeadHeight() int {
	return self.headHeight
}

func (self *chain) ChainId() string {
	return self.chainId
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
	tailHash   string
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

func (self *forkedChain) Head() common.Block {
	return self.GetBlock(self.headHeight)
}

func (self *forkedChain) GetBlock(height int) common.Block {
	w := self.getBlock(height, true)
	if w == nil {
		return nil
	}
	return w.block
}

func (self *forkedChain) contains(height int) bool {
	return height > self.tailHeight && self.headHeight <= height
}

func newBlockChainPool(name string) *BCPool {
	return &BCPool{
		Id: name,
	}
}
func (self *BCPool) init(insertChainFn insertChainForkCheck,
	removeChainFn removeChainForkCheck,
	verifier verifier.Verifier,
	syncer syncer.Syncer,
	reader ChainReader) {

	diskChain := &diskChain{chainId: self.Id + "-diskchain", reader: reader}
	chainpool := &chainPool{
		poolId:        self.Id,
		insertChainFn: insertChainFn,
		removeChainFn: removeChainFn,
		verifier:      verifier,
		diskChain:     diskChain,
	}
	chainpool.current = &forkedChain{}

	chainpool.current.chainId = chainpool.genChainId()
	blockpool := &blockPool{
		compoundBlocks: make(map[string]*BlockForPool),
		freeBlocks:     make(map[string]*BlockForPool),
	}
	self.chainpool = chainpool
	self.blockpool = blockpool
	self.verifier = verifier
	self.syncer = syncer

	self.chainpool.init()

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
func (self *chainPool) init() {
	initBlock := self.diskChain.Head()
	self.current.init(initBlock)
	self.current.referChain = self.diskChain
	self.chains = make(map[string]*forkedChain)
	self.snippetChains = make(map[string]*snippetChain)
	self.chains[self.current.chainId] = self.current
}

func (self *chainPool) currentModifyToChain(chain *forkedChain) error {
	head := self.diskChain.Head()
	w := chain.getBlock(head.Height(), true)
	if w == nil ||
		w.block.Hash() != head.Hash() {
		return common.StrError{"error"}
	}
	for chain.referChain.id() != self.diskChain.id() {
		fromChain := chain.referChain.(*forkedChain)
		self.modifyRefer(fromChain, chain)
	}
	log.Warn("current modify from:%s, to:%s", self.current.id(), chain.id())
	self.current = chain
	return nil
}

func (self *chainPool) modifyRefer(from *forkedChain, to *forkedChain) {
	for i := to.tailHeight; i > from.tailHeight; i-- {
		w := from.heightBlocks[i]
		from.removeTail(w)
		to.addTail(w)
	}
	to.referChain = from.referChain
	from.referChain = to
}
func (self *chainPool) currentModify(initBlock common.Block) {
	new := &forkedChain{}
	new.chainId = self.genChainId()
	new.init(initBlock)
	new.referChain = self.diskChain
	self.current = new
	self.chains[new.chainId] = new
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
		point := findForkPoint(snippet, c, false)
		if point != nil {
			return true, false, c
		}
		if snippet.headHeight == snippet.tailHeight {
			delete(self.snippetChains, snippet.id())
			return false, false, nil
		}
	}
	if snippet.tailHeight <= self.diskChain.Head().Height() {
		point := findForkPoint(snippet, self.current, true)
		if point != nil {
			return true, false, self.current
		}
	}
	if snippet.headHeight == snippet.tailHeight {
		delete(self.snippetChains, snippet.id())
		return false, false, nil
	}
	return false, false, nil
}

// snippet.tailHeight <= chain.headHeight
func findForkPoint(snippet *snippetChain, chain heightChainReader, refer bool) *BlockForPool {
	tailHeight := snippet.tailHeight
	headHeight := snippet.headHeight

	forkpoint := chain.getBlock(tailHeight, refer)
	if forkpoint == nil {
		return nil
	}
	if forkpoint.block.Hash() != snippet.tailHash {
		return nil
	}

	for i := tailHeight + 1; i <= headHeight; i++ {
		uncle := chain.getBlock(i, refer)
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
func (self *chainPool) check(verifierFailcallback verifier.Callback, insertSuccessCallback verifier.Callback) {

	current := self.current
	minH := current.tailHeight + 1
	headH := current.headHeight
L:
	for i := minH; i <= headH; i++ {
		wrapper := current.getBlock(i, false)
		block := wrapper.block
		stat := wrapper.verifyStat
		//if !wrapper.checkForkVersion() {
		wrapper.reset()
		//}
		self.verifier.VerifyReferred(block, stat)
		if !wrapper.checkForkVersion() {
			wrapper.reset()
			continue
		}
		result := stat.VerifyResult()
		switch result {
		case verifier.PENDING:
		case verifier.FAIL:
			log.Error("forkedChain forked. verify result is %s. block info:account[%s],hash[%s],height[%d]",
				result, block.Signer(), block.Hash(), block.Height())
			if verifierFailcallback != nil {
				verifierFailcallback(block, stat)
			}
			break L
		case verifier.SUCCESS:
			if block.Height() == current.tailHeight+1 {
				err := self.writeToChain(current, wrapper)
				if err == nil && insertSuccessCallback != nil {
					insertSuccessCallback(block, stat)
				}
			}
		default:
			log.Error("Unexpected things happened. verify result is %d. block info:account[%s],hash[%s],height[%d]",
				result, block.Signer(), block.Hash(), block.Height())
		}
	}
}

//func (self *BCPool) Rollback(new common.Block) error {
//	return self.chainpool.rollback(new.Height())
//}
func (self *BCPool) Rollback(rollbackHeight int, rollbackHash string) error {
	// todo add hash
	return self.chainpool.rollback(rollbackHeight)
}
func (self *BCPool) RollbackAll() error {
	return self.chainpool.rollback(-1)
}

func (self *chainPool) rollback(newHeight int) error {
	height := self.current.tailHeight
	log.Warn("chain[%s] rollback. from:%d, to:%d", self.current.id(), height, newHeight)
	for i := height; i > newHeight; i-- {
		block := self.diskChain.getBlock(i, true)
		block.verifyStat = self.verifier.NewVerifyStat(verifier.VerifyReferred, block.block)
		e := self.removeChainFn(block.block)
		if e != nil {
			log.Error("remove from chain error. %v", e)
			return e
		} else {
			self.current.addTail(block)
		}
	}

	{ // check logic, could be deleted
		head := self.diskChain.Head()
		if self.current.tailHeight != head.Height() ||
			self.current.tailHash != head.Hash() {
			log.Error("error rollback. pool:%s, rollback:%d", self.poolId, newHeight)
			return common.StrError{"rollback fail."}
		}
	}
	return nil
}

func (self *chainPool) insertNotify(head common.Block) {
	self.currentModify(head)
}

func (self *chainPool) writeToChain(chain *forkedChain, wrapper *BlockForPool) error {
	block := wrapper.block
	height := block.Height()
	hash := block.Hash()
	forkVersion := wrapper.forkVersion
	err := self.insertChainFn(block, forkVersion)
	if err == nil {
		chain.removeTail(wrapper)
		//self.fixReferInsert(chain, self.diskChain, height)
		return nil
	} else {
		log.Error("waiting pool insert forkedChain fail. height:[%d], hash:[%s]", height, hash)
		return err
	}
}
func (self *chainPool) printChains() {
	result := "\n---------------" + self.poolId + "--start-----------------\n"
	chains := copyChains(self.chains)
	for _, c := range chains {
		hashs := ""
		for i := c.headHeight; i >= 0; i-- {
			block := c.getBlock(i, true)
			hashs = block.block.Hash() + "\n" + hashs
		}
		result = result + "hashes:\n" + hashs + "\n"
		if c.chainId == self.current.chainId {
			result = result + c.String() + " [current]\n"
		} else {
			result = result + c.String() + "\n"
		}
		result = result + "++++++++++++++++++++++++++++++++++++++++" + "\n"

	}
	result = result + "---------------" + self.poolId + "--end-----------------\n"
	log.Info(result)
}

//func (self *chainPool) fixReferInsert(origin heightChainReader, target heightChainReader, fixHeight int) {
//	originId := origin.id()
//	for id, chain := range self.chains {
//		if chain.referChain.id() == originId && chain.tailHeight <= fixHeight {
//			chain.referChain = target
//			log.Info("forkedChain[%s] reset refer from %s because of %d, refer to disk.", id, originId, fixHeight)
//		}
//	}
//}
//
//func (self *chainPool) fixReferRollback(origin heightChainReader, target heightChainReader, fixHeight int) {
//	originId := origin.id()
//	for id, chain := range self.chains {
//		if chain.referChain.id() == originId && chain.tailHeight> fixHeight {
//			chain.referChain = self.diskChain
//			log.Info("forkedChain[%s] reset refer from %s because of %d, refer to disk.", id, targetId, fixHeight)
//		}
//	}
//}

func (self *forkedChain) init(initBlock common.Block) {
	self.heightBlocks = make(map[int]*BlockForPool)
	self.tailHeight = initBlock.Height()
	self.tailHash = initBlock.Hash()
	self.headHeight = initBlock.Height()
	self.headHash = initBlock.Hash()
}

func (self *forkedChain) addHead(w *BlockForPool) {
	self.headHash = w.block.Hash()
	self.headHeight = w.block.Height()
	self.heightBlocks[w.block.Height()] = w
}

func (self *forkedChain) removeTail(w *BlockForPool) {
	self.tailHash = w.block.Hash()
	self.tailHeight = w.block.Height()
	delete(self.heightBlocks, w.block.Height())
}

func (self *forkedChain) addTail(w *BlockForPool) {
	self.tailHash = w.block.PreHash()
	self.tailHeight = w.block.Height() - 1
	self.heightBlocks[w.block.Height()] = w
}

func (self *forkedChain) String() string {
	return "chainId:\t" + self.chainId + "\n" +
		"headHeight:\t" + strconv.Itoa(self.headHeight) + "\n" +
		"headHash:\t" + "[" + self.headHash + "]\t" + "\n" +
		"tailHeight:\t" + strconv.Itoa(self.tailHeight)
}

type insertChainForkCheck func(block common.Block, forkVersion int) error
type removeChainForkCheck func(block common.Block) error

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

func (self *BCPool) AddBlock(block common.Block) {
	stat := self.verifier.NewVerifyStat(verifier.VerifyReferred, block)
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

func (self *BCPool) LoopGenSnippetChains() {
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
func (self *BCPool) AddDirectBlock(block common.Block) error {
	forkVersion := version.ForkVersion()
	stat := self.verifier.NewVerifyStat(verifier.VerifyReferred, block)
	self.verifier.VerifyReferred(block, stat)
	result := stat.VerifyResult()
	switch result {
	case verifier.PENDING:
		return common.StrError{"add pending."}
	case verifier.FAIL:
		return common.StrError{"add error."}
	case verifier.SUCCESS:
		self.chainpool.insertChainFn(block, forkVersion)
		head := self.chainpool.diskChain.Head()
		self.chainpool.insertNotify(head)
		return nil
	default:
		return common.StrError{"add unexpected."}
	}
}
func (self *BCPool) LoopAppendChains() {
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
}
func (self *BCPool) LoopFetchForSnippets() {
	sortSnippets := copyMap(self.chainpool.snippetChains)
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
}
func (self *BCPool) CheckCurrentInsert(verifierFailcallback verifier.Callback, insertSuccessCallback verifier.Callback) {
	self.chainpool.printChains()
	self.chainpool.check(verifierFailcallback, insertSuccessCallback)
}

func (self *BCPool) whichChain(height int, hash string) *forkedChain {
	var finalChain *forkedChain
	for _, chain := range self.chainpool.chains {
		block := chain.getBlock(height, false)
		if block != nil && block.block.Hash() == hash {
			finalChain = chain
			break
		}
	}
	if finalChain == nil {
		block := self.chainpool.current.getBlock(height, true)
		if block != nil && block.block.Hash() == hash {
			finalChain = self.chainpool.current
		}
	}
	if finalChain == nil {
		// todo fetch data
		head := self.chainpool.diskChain.Head()
		self.syncer.Fetch(syncer.BlockHash{Height: height, Hash: hash}, height-head.Height())
		log.Warn("block chain can't find. poolId:%s, height:%d, hash:%s", self.chainpool.poolId, height, hash)
		return nil
	}
	return finalChain
}

// fork point must be in memory
func (self *BCPool) currentModify(forkHeight int, forkHash string) error {
	var finalChain *forkedChain
	//forkHeight := target.Height
	//forkHash := target.Hash
	for _, chain := range self.chainpool.chains {
		block := chain.getBlock(forkHeight, false)
		if block != nil {
			finalChain = chain
			break
		}
	}

	if finalChain == nil {
		// todo fetch data
		head := self.chainpool.diskChain.Head()
		self.syncer.Fetch(syncer.BlockHash{Height: forkHeight, Hash: forkHash}, forkHeight-head.Height())
		log.Warn("account can't fork. poolId:%s, height:%d, hash:%s", self.chainpool.poolId, forkHeight, forkHash)
		return nil
	}
	if finalChain.id() == self.chainpool.current.id() {
		return nil
	}

	_, forkBlock, err := self.getForkPointByChains(finalChain, self.chainpool.current)
	if err != nil {
		return common.StrError{"can't find fork point."}
	}

	//self.chainpool.getSendBlock(forkBlock.Height())

	err = self.chainpool.rollback(forkBlock.Height())
	if err != nil {
		return common.StrError{"rollback fail."}
	}
	return self.chainpool.currentModifyToChain(finalChain)
}
func (self *BCPool) CurrentModifyToChain(target Chain) error {
	chain := target.(*forkedChain)
	return self.chainpool.currentModifyToChain(chain)
}

func (self *BCPool) LongestChain() Chain {
	readers := self.chainpool.chains
	current := self.chainpool.current
	longest := current
	for _, reader := range readers {
		height := reader.headHeight
		if height > longest.headHeight {
			longest = reader
		}
	}
	return longest
}
func (self *BCPool) CurrentChain() Chain {
	return self.chainpool.current
}

// keyPoint, forkPoint, err
func (self *BCPool) getForkPointByChains(chain1 Chain, chain2 Chain) (common.Block, common.Block, error) {
	if chain1.Head().Height() > chain2.Head().Height() {
		return self.getForkPoint(chain1, chain2)
	} else {
		return self.getForkPoint(chain2, chain1)
	}
}

// keyPoint, forkPoint, err
func (self *BCPool) getForkPoint(longest Chain, current Chain) (common.Block, common.Block, error) {
	curHeadHeight := current.HeadHeight()

	i := curHeadHeight
	var forkedBlock common.Block

	for {
		block := longest.GetBlock(i)
		curBlock := current.GetBlock(i)
		if block == nil {
			log.Error("longest chain is not longest. chainId:%s. height:%d", longest.ChainId(), i)
			return nil, nil, common.StrError{"longest chain error."}
		}

		if curBlock == nil {
			log.Error("current chain is wrong. chainId:%s. height:%d", current.ChainId(), i)
			return nil, nil, common.StrError{"current chain error."}
		}

		if block.Hash() == curBlock.Hash() {
			forkedBlock = block
			keyPoint := longest.GetBlock(i + 1)
			return keyPoint, forkedBlock, nil
		}
		i = i - 1
	}
	return nil, nil, common.StrError{"can't find fork point"}
}

func (self *BCPool) loop() {
	for {
		self.LoopGenSnippetChains()
		self.LoopAppendChains()
		self.LoopFetchForSnippets()
		self.CheckCurrentInsert(nil, nil)
		time.Sleep(time.Second)
	}
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
