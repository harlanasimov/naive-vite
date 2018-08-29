package miner

import (
	"strconv"
	"testing"
	"time"

	"github.com/asaskevich/EventBus"
	"github.com/viteshan/naive-vite/common"
	"github.com/viteshan/naive-vite/consensus"
	"github.com/viteshan/naive-vite/ledger"
)

type SnapshotRW struct {
	Ch chan<- int
}

func (SnapshotRW) MiningSnapshotBlock(address string, timestamp int64) error {
	println(address + ":" + time.Unix(timestamp, 0).Format(time.StampMilli) + ":" + strconv.FormatInt(timestamp, 10))
	return nil
}

func genMiner(committee *consensus.Committee) (Miner, EventBus.Bus) {
	bus := EventBus.New()
	coinbase := common.HexToAddress("vite_2ad1b8f936f015fc80a2a5857dffb84b39f7675ab69ae31fc8")
	rw := &SnapshotRW{}
	miner := NewMiner(rw, bus, coinbase, committee)
	return miner, bus
}

func genMinerAuto(committee *consensus.Committee) (Miner, EventBus.Bus) {
	bus := EventBus.New()
	coinbase := common.HexToAddress("vite_2ad1b8f936f015fc80a2a5857dffb84b39f7675ab69ae31fc8")
	rw := &SnapshotRW{}
	miner := NewMiner(rw, bus, coinbase, committee)
	return miner, bus
}

func genCommitee() *consensus.Committee {
	genesisTime := ledger.GetGenesisSnapshot().Timestamp()
	committee := consensus.NewCommittee(genesisTime, 1, int32(len(consensus.DefaultMembers)))
	return committee
}

func TestNewMiner(t *testing.T) {
	committee := genCommitee()
	miner, bus := genMiner(committee)

	committee.Init()
	miner.Init()
	committee.Start()
	miner.Start()
	var c chan int = make(chan int)
	select {
	case c <- 0:
	case <-time.After(5 * time.Second):
		println("timeout and downloader finish.")
		//miner.downloaderRegisterCh <- 0
		bus.Publish(common.DwlDone)
		println("-----------timeout")
	}
	c <- 0
}
func TestVerifier(t *testing.T) {
	committee := genCommitee()

	coinbase := common.HexToAddress("vite_2ad1b8f936f015fc80a2a5857dffb84b39f7675ab69ae31fc8")

	verify, _ := committee.Verify(SnapshotRW{}, common.NewSnapshotBlock(0, "", "", coinbase.String(), time.Unix(1532504321, 0), nil))
	println(verify)
	verify2, _ := committee.Verify(SnapshotRW{}, common.NewSnapshotBlock(0, "", "", coinbase.String(), time.Unix(1532504320, 0), nil))
	println(verify2)
}

func TestChan(t *testing.T) {
	ch1 := make(chan int)
	ch2 := make(chan int)
	ch3 := make(chan int)

	go func() {
		select {
		// Handle ChainHeadEvent
		case event := <-ch1:
			println(event)
		case e2, ok := <-ch2: // ok代表channel是否正常使用, 如果ok==false, 说明channel已经关闭
			println(e2)
			println(ok)
			println("------")
		}

		ch3 <- 99

	}()

	time.Sleep(1 * time.Second)

	//ch2 <-10
	close(ch2)

	i := <-ch3

	println(i)
}

func TestLifecycle(t *testing.T) {
	commitee := genCommitee()
	miner, bus := genMinerAuto(commitee)

	commitee.Init()
	miner.Init()

	bus.Publish(common.DwlDone)
	commitee.Start()

	miner.Start()
	var c chan int = make(chan int)

	//
	time.Sleep(30 * time.Second)
	println("miner stop.")
	miner.Stop()
	time.Sleep(1 * time.Second)

	println("miner start.")
	miner.Start()

	time.Sleep(20 * time.Second)
	println("miner stop.")
	miner.Stop()
	time.Sleep(1 * time.Second)

	c <- 0
}
