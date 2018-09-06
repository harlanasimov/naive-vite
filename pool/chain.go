package pool

import (
	"time"

	"strconv"

	"github.com/pkg/errors"
	ch "github.com/viteshan/naive-vite/chain"
	"github.com/viteshan/naive-vite/common"
	"github.com/viteshan/naive-vite/version"
)

type chainRw interface {
	insertChain(block common.Block, forkVersion int) error
	removeChain(block common.Block) error

	head() common.Block
	getBlock(height int) common.Block
}

type accountCh struct {
	address string
	bc      ch.BlockChain
	version *version.Version
}

func (self *accountCh) insertChain(block common.Block, forkVersion int) error {
	if forkVersion != self.version.Val() {
		return errors.New("error fork version. current:" + self.version.String() + ", target:" + strconv.Itoa(forkVersion))
	}
	return self.bc.InsertAccountBlock(self.address, block.(*common.AccountStateBlock))
}

func (self *accountCh) removeChain(block common.Block) error {
	return self.bc.RemoveAccountHead(self.address, block.(*common.AccountStateBlock))
}

func (self *accountCh) head() common.Block {
	block, _ := self.bc.HeadAccount(self.address)
	if block == nil {
		return nil
	}
	return block
}

func (self *accountCh) getBlock(height int) common.Block {
	if height == -1 {
		return common.NewAccountBlock(-1, "", "", "", time.Unix(0, 0), 0, 0, 0, "", common.SEND, "", "", "", -1)
	}
	block := self.bc.GetAccountByHeight(self.address, height)
	if block == nil {
		return nil
	}
	return block
}
func (self *accountCh) findAboveSnapshotHeight(height int) *common.AccountStateBlock {
	return self.bc.FindAccountAboveSnapshotHeight(self.address, height)
}

type snapshotCh struct {
	bc      ch.BlockChain
	version *version.Version
}

func (self *snapshotCh) getBlock(height int) common.Block {
	head := self.bc.GetSnapshotByHeight(height)
	if head == nil {
		return nil
	}
	return head
}

func (self *snapshotCh) head() common.Block {
	block, _ := self.bc.HeadSnapshot()
	if block == nil {
		return nil
	}
	return block
}

func (self *snapshotCh) insertChain(block common.Block, forkVersion int) error {
	return self.bc.InsertSnapshotBlock(block.(*common.SnapshotBlock))
}

func (self *snapshotCh) removeChain(block common.Block) error {
	return self.bc.RemoveSnapshotHead(block.(*common.SnapshotBlock))
}
