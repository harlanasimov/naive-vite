package ledger

import (
	"errors"
	"strconv"
	"time"

	"github.com/golang-collections/collections/stack"
	"github.com/viteshan/naive-vite/common"
	"github.com/viteshan/naive-vite/common/log"
	"github.com/viteshan/naive-vite/tools"
)

// account block chain
type AccountChain struct {
	address         string
	head            *common.AccountStateBlock
	accountDB       map[string]*common.AccountStateBlock
	accountHeightDB map[int]*common.AccountStateBlock

	reqPool       *reqPool
	snapshotPoint *stack.Stack
}

var blank = common.NewAccountBlock(-1, "", "", "", time.Unix(1533550870, 0),
	0, 0, GetGenesisSnapshot().Height(), GetGenesisSnapshot().Hash(), common.CREATE, "", "", "")

func NewAccountChain(address string, reqPool *reqPool) *AccountChain {
	self := &AccountChain{}
	self.address = address
	self.head = common.NewAccountBlock(0, "", "", address, time.Now(),
		100, 0, GetGenesisSnapshot().Height(), GetGenesisSnapshot().Hash(), common.CREATE, address, address, "")
	self.head.SetHash(tools.CalculateAccountHash(self.head))
	self.accountDB = make(map[string]*common.AccountStateBlock)
	self.accountHeightDB = make(map[int]*common.AccountStateBlock)
	self.accountDB[self.head.Hash()] = self.head
	self.accountHeightDB[self.head.Height()] = self.head
	self.reqPool = reqPool

	self.snapshotPoint = stack.New()
	return self
}

func (self *AccountChain) Head() common.Block {
	return self.head
}

func (self *AccountChain) GetBlock(height int) common.Block {
	if height == -1 {
		return blank
	}
	if height < 0 {
		log.Error("can't request height 0 block. account:%s", self.address)
		return nil
	}
	block, ok := self.accountHeightDB[height]
	if ok {
		return block
	}
	return nil
}

func (self *AccountChain) GetBlockByHashH(hashH common.HashHeight) *common.AccountStateBlock {
	if hashH.Height < 0 {
		log.Error("can't request height 0 block. account:%s", self.address)
		return nil
	}
	block, ok := self.accountHeightDB[hashH.Height]
	if ok && block.Hash() == hashH.Hash {
		return block
	}
	return nil
}
func (self *AccountChain) GetBlockByHash(hash string) *common.AccountStateBlock {
	block, ok := self.accountDB[hash]
	if !ok {
		return nil
	}
	return block
}

func (self *AccountChain) insertChain(b common.Block, forkVersion int) error {
	log.Info("insert to account Chain: %s", b)
	block := b.(*common.AccountStateBlock)
	self.accountDB[block.Hash()] = block
	self.accountHeightDB[block.Height()] = block
	self.head = block
	self.reqPool.blockInsert(block)
	return nil
}
func (self *AccountChain) removeChain(b common.Block) error {
	log.Info("remove from account Chain: %s", b)
	block := b.(*common.AccountStateBlock)
	has := self.hasSnapshotPoint(block.Height(), block.Hash())
	if has {
		return errors.New("has snapshot.")
	}

	head := self.accountDB[block.PreHash()]
	delete(self.accountDB, block.Hash())
	delete(self.accountHeightDB, block.Height())
	self.reqPool.blockRollback(block)
	self.head = head
	return nil
}

func (self *AccountChain) FindBlockAboveSnapshotHeight(snapshotHeight int) *common.AccountStateBlock {
	for i := self.head.Height(); i >= 0; i-- {
		block := self.accountHeightDB[i]
		if block.SnapshotHeight <= snapshotHeight {
			return block
		}
	}
	return nil
}
func (self *AccountChain) GetBySourceBlock(sourceHash string) *common.AccountStateBlock {
	height := self.head.Height()
	for i := height; i > 0; i-- {
		// first block(i==0) is create block
		v := self.accountHeightDB[i]
		if v.BlockType == common.SEND && v.Hash() == sourceHash {
			return v
		}
	}
	return nil
}

func (self *AccountChain) NextSnapshotPoint() (int, string) {
	var lastPoint *common.SnapshotPoint
	p := self.snapshotPoint.Peek()
	if p != nil {
		lastPoint = p.(*common.SnapshotPoint)
	}

	if lastPoint == nil {
		if self.head != nil {
			return self.head.Height(), self.head.Hash()
		}
	}
	return -1, ""
}

func (self *AccountChain) SnapshotPoint(snapshotHeight int, snapshotHash string, h *common.AccountHashH) error {
	// check valid
	head := self.head
	if head == nil {
		return errors.New("account[" + self.address + "] not exist.")
	}
	if h.Hash == head.Hash() && h.Height == head.Height() {
		point := &common.SnapshotPoint{SnapshotHeight: snapshotHeight, SnapshotHash: snapshotHash, AccountHash: h.Hash, AccountHeight: h.Height}
		self.snapshotPoint.Push(point)
		return nil
	}
	errMsg := "account[] state error. accHeight: " + strconv.Itoa(h.Height) +
		"accHash:" + h.Hash +
		" expAccHeight:" + strconv.Itoa(head.Height()) +
		" expAccHash:" + head.Hash()
	return errors.New(errMsg)
}

//SnapshotPoint ddd
func (self *AccountChain) RollbackSnapshotPoint(start *common.SnapshotPoint, end *common.SnapshotPoint) error {
	point := self.peek()
	if point == nil {
		return errors.New("not exist snapshot point.")
	}
	if !point.Equals(start) {
		return errors.New("not equals for start")
	}
	for {
		point := self.peek()
		if point == nil {
			return errors.New("not exist snapshot point.")
		}
		if point.AccountHeight <= end.AccountHeight {
			self.snapshotPoint.Pop()
		} else {
			break
		}
		if point.AccountHeight == end.AccountHeight {
			break
		}
	}
	return nil
}

//func (self *AccountChain) rollbackSnapshotPoint(start *common.SnapshotPoint) error {
//	point := self.peek()
//	if point == nil {
//		return errors.New("not exist snapshot point."}
//	}
//
//	if point.SnapshotHash == start.SnapshotHash &&
//		point.SnapshotHeight == start.SnapshotHeight &&
//		point.AccountHeight == start.AccountHeight &&
//		point.AccountHash == start.AccountHash {
//		self.snapshotPoint.Pop()
//		return nil
//	}
//
//	errMsg := "account[" + self.address + "] state error. expect:" + point.String() +
//		", actual:" + start.String()
//	return errors.New( errMsg}
//}

//SnapshotPoint ddd
func (self *AccountChain) hasSnapshotPoint(accountHeight int, accountHash string) bool {
	point := self.peek()
	if point == nil {
		return false
	}

	if point.AccountHeight >= accountHeight {
		return true
	}
	return false

}

func (self *AccountChain) peek() *common.SnapshotPoint {
	var lastPoint *common.SnapshotPoint
	p := self.snapshotPoint.Peek()
	if p != nil {
		lastPoint = p.(*common.SnapshotPoint)
	}
	return lastPoint
}
