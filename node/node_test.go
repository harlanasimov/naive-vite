package node

import (
	"strconv"
	"testing"

	"time"

	"github.com/viteshan/naive-vite/common/config"
	"github.com/viteshan/naive-vite/consensus"
	"github.com/viteshan/naive-vite/p2p"
)

func TestNode(t *testing.T) {
	bootAddr := "localhost:8000"
	startBoot(bootAddr)
	//N := 2
	//for j := 0; j < N; j++ {
	//	go func(i int) {
	//		cfg := config.Node{
	//			P2pCfg:       config.P2P{NodeId: strconv.Itoa(i), Port: 8080 + i, LinkBootAddr: bootAddr, NetId: 0},
	//			ConsensusCfg: config.Consensus{Interval: 1, MemCnt: len(consensus.DefaultMembers)},
	//		}
	//		n := node{}
	//		n.Init(cfg)
	//		n.Start()
	//	}(j)
	//}
	cfg := config.Node{
		P2pCfg:       config.P2P{NodeId: strconv.Itoa(3), Port: 8080 + 3, LinkBootAddr: bootAddr, NetId: 0},
		ConsensusCfg: config.Consensus{Interval: 1, MemCnt: len(consensus.DefaultMembers)},
		MinerCfg:     config.Miner{Enabled: true, HexCoinbase: "vite_2ad1b8f936f015fc80a2a5857dffb84b39f7675ab69ae31fc8"},
	}
	n := NewNode(cfg)
	n.Init()
	n.Start()

	time.Sleep(200 * time.Second)
	n.Stop()
}

func startBoot(bootAddr string) p2p.Boot {
	cfg := config.Boot{BootAddr: bootAddr}
	boot := p2p.NewBoot(cfg)
	boot.Start()
	return boot
}

func TestStruct(t *testing.T) {
	node := config.Node{}
	if node.MinerCfg.Enabled == true {
		t.Error("error except")
	}
}

func TestBootNode(t *testing.T) {
	bootAddr := "localhost:8000"
	startBoot(bootAddr)
	time.Sleep(200 * time.Second)
}

func TestNode_Start(t *testing.T) {
	bootAddr := "localhost:8000"
	i := 0
	cfg := config.Node{
		P2pCfg:       config.P2P{NodeId: strconv.Itoa(i), Port: 8080 + i, LinkBootAddr: bootAddr, NetId: 0},
		ConsensusCfg: config.Consensus{Interval: 1, MemCnt: len(consensus.DefaultMembers)},
	}
	n := NewNode(cfg)
	n.Init()
	n.Start()
	time.Sleep(200 * time.Second)
}
