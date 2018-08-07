package ledger

import (
	"fmt"
	"github.com/viteshan/naive-vite/common"
	"github.com/viteshan/naive-vite/common/log"
	"github.com/viteshan/naive-vite/syncer"
	"github.com/viteshan/naive-vite/tools"
	"strconv"
	"testing"
	"time"
)

func TestTime(t *testing.T) {
	now := time.Now()
	fmt.Printf("%d\n", now.Unix())
	block := common.NewSnapshotBlock(0, "460780b73084275422b520a42ebb9d4f8a8326e1522c79817a19b41ba69dca5b", "", "viteshan", time.Unix(1533550878, 0), nil)
	hash := tools.CalculateSnapshotHash(block)
	fmt.Printf("hash:%s\n", hash) //460780b73084275422b520a42ebb9d4f8a8326e1522c79817a19b41ba69dca5b
}

type TestSyncer struct {
	blocks map[string]*TestBlock
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

func (self *TestSyncer) Fetch(hash syncer.BlockHash, prevCnt int) {
	log.Info("fetch request,cnt:%d, hash:%v", prevCnt, hash)
}

func TestLedger(t *testing.T) {
	testSyncer := &TestSyncer{blocks: make(map[string]*TestBlock)}
	ledger := NewLedger(testSyncer)
	ledger.Start()

	ledger.AddSnapshotBlock(genSnapshotBlock(ledger))
	time.Sleep(10 * time.Second)
}
func genSnapshotBlock(ledger *ledger) *common.SnapshotBlock {
	block := ledger.sc.head

	snapshot := common.NewSnapshotBlock(block.Height()+1, "", block.Hash(), "viteshan", time.Now(), nil)
	snapshot.SetHash(tools.CalculateSnapshotHash(snapshot))
	return snapshot
}
