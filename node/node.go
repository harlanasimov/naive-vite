package node

import (
	"sync"

	"time"

	"github.com/asaskevich/EventBus"
	"github.com/viteshan/naive-vite/common"
	"github.com/viteshan/naive-vite/common/config"
	"github.com/viteshan/naive-vite/common/log"
	"github.com/viteshan/naive-vite/consensus"
	"github.com/viteshan/naive-vite/ledger"
	"github.com/viteshan/naive-vite/miner"
	"github.com/viteshan/naive-vite/p2p"
	"github.com/viteshan/naive-vite/syncer"
)

type node struct {
	p2p       p2p.P2P
	syncer    syncer.Syncer
	ledger    ledger.Ledger
	consensus consensus.Consensus
	miner     miner.Miner

	cfg    config.Node
	closed chan struct{}
	wg     sync.WaitGroup
}

func (self *node) Init(cfg config.Node) {
	self.cfg = cfg
}

func (self *node) Start() {
	bus := EventBus.New()
	self.wg.Add(1)
	self.p2p = p2p.NewP2P(self.cfg.P2pCfg)
	self.p2p.Start()

	self.syncer = syncer.NewSyncer(self.p2p)

	self.ledger = ledger.NewLedger()
	self.syncer.Init(self.ledger)
	self.ledger.Init(self.syncer)
	self.ledger.Start()

	self.consensus = consensus.NewConsensus(ledger.GetGenesisSnapshot().Timestamp(), self.cfg.ConsensusCfg)
	self.consensus.Init()
	self.consensus.Start()

	if self.cfg.MinerCfg.Enabled {
		self.miner = miner.NewMiner(self.ledger, bus, self.cfg.MinerCfg.CoinBase(), self.consensus)
		self.miner.Init()
		self.miner.Start()
	}

	select {
	case <-time.After(2 * time.Second):
		bus.Publish(common.DwlDone)
	}

	log.Info("node started...")
	<-self.closed
	self.wg.Done()
}

func (self *node) Stop() {
	close(self.closed)
	self.wg.Wait()
	log.Info("node stopped...")
}
