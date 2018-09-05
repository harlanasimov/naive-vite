package chain

import (
	"errors"
	"strconv"

	"github.com/golang-collections/collections/stack"
	"github.com/viteshan/naive-vite/common"
	"github.com/viteshan/naive-vite/common/face"
	"github.com/viteshan/naive-vite/common/log"
	"github.com/viteshan/naive-vite/store"
)

// account block chain
type accountChain struct {
	address       string
	head          *common.AccountStateBlock
	store         store.BlockStore
	listener      face.ChainListener
	snapshotPoint *stack.Stack
}

func newAccountChain(address string, listener face.ChainListener, store store.BlockStore) *accountChain {
	self := &accountChain{}
	self.address = address
	self.store = store
	head := self.store.GetAccountHead(self.address)
	if head != nil {
		self.head = self.store.GetAccountByHeight(self.address, head.Height)
	}

	self.listener = listener
	self.snapshotPoint = stack.New()
	return self
}

func (self *accountChain) Head() *common.AccountStateBlock {
	return self.head
}

func (self *accountChain) GetBlockByHeight(height int) *common.AccountStateBlock {
	if height < 0 {
		log.Error("can't request height 0 block. account:%s", self.address)
		return nil
	}
	block := self.store.GetAccountByHeight(self.address, height)

	return block
}

func (self *accountChain) GetBlockByHashH(hashH common.HashHeight) *common.AccountStateBlock {
	if hashH.Height < 0 {
		log.Error("can't request height 0 block. account:%s", self.address)
		return nil
	}
	block := self.store.GetAccountByHeight(self.address, hashH.Height)
	if block != nil && block.Hash() == hashH.Hash {
		return block
	}
	return nil
}
func (self *accountChain) GetBlockByHash(hash string) *common.AccountStateBlock {
	block := self.store.GetAccountByHash(hash)
	return block
}

func (self *accountChain) insertChain(block *common.AccountStateBlock) error {
	log.Info("insert to account Chain: %v", block)
	self.store.PutAccount(self.address, block)
	self.head = block
	self.listener.AccountInsertCallback(self.address, block)
	self.store.SetAccountHead(self.address, &common.HashHeight{Hash: block.Hash(), Height: block.Height()})
	return nil
}
func (self *accountChain) removeChain(block *common.AccountStateBlock) error {
	log.Info("remove from account Chain: %v", block)
	has := self.hasSnapshotPoint(block.Height(), block.Hash())
	if has {
		return errors.New("has snapshot.")
	}

	head := self.store.GetAccountByHash(block.Hash())
	self.store.DeleteAccount(self.address, common.HashHeight{Hash: block.Hash(), Height: block.Height()})
	self.listener.AccountRemoveCallback(self.address, block)
	self.head = head
	if head == nil {
		self.store.SetAccountHead(self.address, nil)
	} else {
		self.store.SetAccountHead(self.address, &common.HashHeight{Hash: head.Hash(), Height: head.Height()})
	}
	return nil
}

func (self *accountChain) findAccountAboveSnapshotHeight(snapshotHeight int) *common.AccountStateBlock {
	if self.head == nil {
		return nil
	}
	for i := self.head.Height(); i >= 0; i-- {
		block := self.store.GetAccountByHeight(self.address, i)
		if block.SnapshotHeight <= snapshotHeight {
			return block
		}
	}
	return nil
}
func (self *accountChain) getBySourceBlock(sourceHash string) *common.AccountStateBlock {
	if self.head == nil {
		return nil
	}

	height := self.head.Height()
	for i := height; i > 0; i-- {
		// first block(i==0) is create block
		v := self.store.GetAccountByHeight(self.address, i)
		if v.BlockType == common.RECEIVED && v.SourceHash == sourceHash {
			return v
		}
	}
	return nil
}

func (self *accountChain) NextSnapshotPoint() (int, string) {
	var lastPoint *common.SnapshotPoint
	p := self.snapshotPoint.Peek()
	if p != nil {
		lastPoint = p.(*common.SnapshotPoint)
	}

	if lastPoint == nil {
		if self.head != nil {
			return self.head.Height(), self.head.Hash()
		}
	} else {
		if lastPoint.AccountHeight < self.head.Height() {
			return self.head.Height(), self.head.Hash()
		}
	}
	return -1, ""
}

func (self *accountChain) SnapshotPoint(snapshotHeight int, snapshotHash string, h *common.AccountHashH) error {
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

//SnapshotPoint
func (self *accountChain) RollbackSnapshotPoint(start *common.SnapshotPoint, end *common.SnapshotPoint) error {
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

//func (self *accountChain) rollbackSnapshotPoint(start *common.SnapshotPoint) error {
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
func (self *accountChain) hasSnapshotPoint(accountHeight int, accountHash string) bool {
	point := self.peek()
	if point == nil {
		return false
	}

	if point.AccountHeight >= accountHeight {
		return true
	}
	return false

}

func (self *accountChain) peek() *common.SnapshotPoint {
	var lastPoint *common.SnapshotPoint
	p := self.snapshotPoint.Peek()
	if p != nil {
		lastPoint = p.(*common.SnapshotPoint)
	}
	return lastPoint
}
