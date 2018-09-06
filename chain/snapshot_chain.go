package chain

import (
	"time"

	"strconv"

	"encoding/json"

	"github.com/viteshan/naive-vite/common"
	"github.com/viteshan/naive-vite/common/log"
	"github.com/viteshan/naive-vite/store"
)

// snapshot block chain
type snapshotChain struct {
	head  *common.SnapshotBlock
	store store.BlockStore
}

func GetGenesisSnapshot() *common.SnapshotBlock {
	return genesisSnapshot
}

var genesisAccounts = []*common.AccountHashH{
	{common.HashHeight{Hash: "71cef9aa8a67df1055a0801e2fc251fe14a298b8fc1098a54e27db31f7a75e20", Height: 0}, "viteshan"},
	{common.HashHeight{Hash: "c3e7ab31e4834cf18df6705b96c655c845db44f4ee315c6fba5232b4f965fcf0", Height: 0}, "jie"},
}

var genesisSnapshot = common.NewSnapshotBlock(0, "b5a3ee58d163c283e5c8c0f65ff5b26a5cd64cb9dce8119ac4581ebcb54626fd", "", "viteshan", time.Unix(1533550878, 0), genesisAccounts)

func newSnapshotChain(store store.BlockStore) *snapshotChain {
	chain := &snapshotChain{}
	chain.store = store
	// init genesis block
	head := store.GetSnapshotHead()
	if head != nil {
		storeGenesis := store.GetSnapshotByHeight(genesisSnapshot.Height())
		if storeGenesis.Hash() != genesisSnapshot.Hash() {
			panic("error store snapshot hash. code:" + genesisSnapshot.Hash() + ", store:" + storeGenesis.Hash())
		} else {
			chain.head = chain.store.GetSnapshotByHeight(head.Height)
		}
	} else {
		chain.head = genesisSnapshot
		chain.store.PutSnapshot(genesisSnapshot)
		chain.store.SetSnapshotHead(&common.HashHeight{Hash: genesisSnapshot.Hash(), Height: genesisSnapshot.Height()})
	}
	return chain
}

func (self *snapshotChain) Head() *common.SnapshotBlock {
	return self.head
}

func (self *snapshotChain) GetBlockHeight(height int) *common.SnapshotBlock {
	if height < 0 {
		panic("height:" + strconv.Itoa(height))
		log.Error("can't request height 0 block.[snapshotChain]", height)
		return nil
	}
	block := self.store.GetSnapshotByHeight(height)
	return block
}

func (self *snapshotChain) GetBlockByHashH(hashH common.HashHeight) *common.SnapshotBlock {
	if hashH.Height < 0 {
		log.Error("can't request height 0 block.[snapshotChain]", hashH.Height)
		return nil
	}
	block := self.store.GetSnapshotByHeight(hashH.Height)
	if block != nil && hashH.Hash == block.Hash() {
		return block
	}
	return nil
}
func (self *snapshotChain) getBlockByHash(hash string) *common.SnapshotBlock {
	block := self.store.GetSnapshotByHash(hash)
	return block
}

func j(v interface{}) string {
	bytes, _ := json.Marshal(v)
	return string(bytes)
}

func (self *snapshotChain) insertChain(block *common.SnapshotBlock) error {
	log.Info("insert to snapshot Chain: %s", j(block))
	self.store.PutSnapshot(block)
	self.head = block
	self.store.SetSnapshotHead(&common.HashHeight{Hash: block.Hash(), Height: block.Height()})
	return nil
}
func (self *snapshotChain) removeChain(block *common.SnapshotBlock) error {
	log.Info("remove from snapshot Chain: %s", block)

	head := self.store.GetSnapshotByHash(block.PreHash())
	self.store.DeleteSnapshot(common.HashHeight{Hash: block.Hash(), Height: block.Height()})
	self.head = head
	if head == nil {
		self.store.SetSnapshotHead(nil)
	} else {
		self.store.SetSnapshotHead(&common.HashHeight{Hash: head.Hash(), Height: head.Height()})
	}

	return nil
}
