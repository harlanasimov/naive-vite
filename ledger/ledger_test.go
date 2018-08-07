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
	viteshan := "viteshan"
	reqs := ledger.reqPool.getReqs(viteshan)
	if len(reqs) > 0 {
		t.Errorf("reqs should be empty. reqs:%v", reqs)
	}

	viteshan1 := "viteshan1"
	ledger.CreateAccount(viteshan)
	time.Sleep(2 * time.Second)
	snapshotBlock, _ := ledger.HeadSnaphost()
	headAccount, _ := ledger.HeadAccount(viteshan1)

	{
		block := common.NewAccountBlock(1, "", headAccount.Hash(), viteshan1, time.Unix(1533550878, 0),
			0, -105, snapshotBlock.Height(), snapshotBlock.Hash(), common.SEND, viteshan1, viteshan, "")
		block.SetHash(tools.CalculateAccountHash(block))
		var err error
		err = ledger.MiningAccountBlock(viteshan1, block)
		if err == nil {
			t.Error("expected error.")
		} else {
			log.Info("error:%v", err)
		}
	}
	{
		block := common.NewAccountBlock(1, "", headAccount.Hash(), viteshan1, time.Unix(1533550878, 0),
			10, -90, snapshotBlock.Height(), snapshotBlock.Hash(), common.SEND, viteshan1, viteshan, "")
		block.SetHash(tools.CalculateAccountHash(block))

		err := ledger.MiningAccountBlock(viteshan1, block)
		if err != nil {
			t.Errorf("expected error.%v", err)

		}
	}
	{
		reqs = ledger.reqPool.getReqs(viteshan)
		if len(reqs) != 1 {
			t.Errorf("reqs should be empty. reqs:%v", reqs)
		}
		req := reqs[0]

		headAcc, _ := ledger.HeadAccount(viteshan)

		block := common.NewAccountBlock(1, "", headAcc.Hash(), viteshan, time.Unix(1533550878, 0),
			190, 90, snapshotBlock.Height(), snapshotBlock.Hash(), common.RECEIVED, viteshan1, viteshan, req.reqHash)

		block.SetHash(tools.CalculateAccountHash(block))

		err := ledger.MiningAccountBlock(viteshan, block)
		if err != nil {
			t.Errorf("expected error.%v", err)
		}
	}

	time.Sleep(10 * time.Second)
}

func TestSnapshotFork(t *testing.T) {
	testSyncer := &TestSyncer{blocks: make(map[string]*TestBlock)}
	ledger := NewLedger(testSyncer)
	ledger.Start()
	time.Sleep(time.Second)

	//ledger.AddSnapshotBlock(genSnapshotBlock(ledger))
	//ledger.AddSnapshotBlock(genSnapshotBlock(ledger))
	block := ledger.sc.head
	block = genSnapshotBlockBy(block)
	ledger.AddSnapshotBlock(block)
	block = genSnapshotBlockBy(block)
	ledger.AddSnapshotBlock(block)

	block2 := block
	block = genSnapshotBlockBy(block)
	ledger.AddSnapshotBlock(block)
	block = genSnapshotBlockBy(block)
	ledger.AddSnapshotBlock(block)
	time.Sleep(2 * time.Second)
	by := genSnapshotBlockBy(block2)
	ledger.AddSnapshotBlock(by)
	by = genSnapshotBlockBy(by)
	ledger.AddSnapshotBlock(by)
	time.Sleep(10 * time.Second)
	by = genSnapshotBlockBy(by)
	ledger.AddSnapshotBlock(by)

	c :=make(chan int)
	c <-1
	//time.Sleep(10 * time.Second)
}

func genSnapshotBlockBy(block *common.SnapshotBlock) *common.SnapshotBlock {
	snapshot := common.NewSnapshotBlock(block.Height()+1, "", block.Hash(), "viteshan", time.Now(), nil)
	snapshot.SetHash(tools.CalculateSnapshotHash(snapshot))
	return snapshot
}

func genSnapshotBlock(ledger *ledger) *common.SnapshotBlock {
	block := ledger.sc.head

	snapshot := common.NewSnapshotBlock(block.Height()+1, "", block.Hash(), "viteshan", time.Now(), nil)
	snapshot.SetHash(tools.CalculateSnapshotHash(snapshot))
	return snapshot
}
