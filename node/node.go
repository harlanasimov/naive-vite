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
	"github.com/viteshan/naive-vite/wallet"
)

type Node interface {
	Init()
	Start()
	Stop()
	StartMiner()
	StopMiner()
	Leger() ledger.Ledger
	Wallet() wallet.Wallet
}

func NewNode(cfg config.Node) Node {
	self := &node{}
	self.cfg = cfg
	self.p2p = p2p.NewP2P(self.cfg.P2pCfg)
	self.syncer = syncer.NewSyncer(self.p2p)
	self.ledger = ledger.NewLedger()
	self.consensus = consensus.NewConsensus(ledger.GetGenesisSnapshot().Timestamp(), self.cfg.ConsensusCfg)
	self.bus = EventBus.New()

	if self.cfg.MinerCfg.Enabled {
		self.miner = miner.NewMiner(self.ledger, self.bus, self.cfg.MinerCfg.CoinBase(), self.consensus)
	}
	return self
}

type node struct {
	p2p       p2p.P2P
	syncer    syncer.Syncer
	ledger    ledger.Ledger
	consensus consensus.Consensus
	miner     miner.Miner
	wallet    wallet.Wallet
	bus       EventBus.Bus

	cfg    config.Node
	closed chan struct{}
	wg     sync.WaitGroup
}

func (self *node) Init() {
	self.syncer.Init(self.ledger)
	self.ledger.Init(self.syncer)
	self.consensus.Init()

	if self.miner != nil {
		self.miner.Init()
	}
	self.wallet = wallet.NewWallet()
}

func (self *node) Start() {
	self.p2p.Start()
	self.ledger.Start()
	self.consensus.Start()

	if self.miner != nil {
		self.miner.Start()
	}

	select {
	case <-time.After(2 * time.Second):
		self.bus.Publish(common.DwlDone)
	}

	log.Info("node started...")
}

func (self *node) Stop() {
	close(self.closed)

	if self.miner != nil {
		self.miner.Stop()
	}
	self.consensus.Stop()
	self.ledger.Stop()
	self.p2p.Stop()
	self.wg.Wait()
	log.Info("node stopped...")
}

func (self *node) StartMiner() {
	if self.miner == nil {
		self.miner = miner.NewMiner(self.ledger, self.bus, self.cfg.MinerCfg.CoinBase(), self.consensus)
		self.miner.Init()
	}
	self.miner.Start()
	log.Info("miner started...")
}

func (self *node) StopMiner() {
	self.miner.Stop()
	log.Info("miner stopped...")
}

func (self *node) Leger() ledger.Ledger {
	return self.ledger
}

func (self *node) Wallet() wallet.Wallet {
	return self.wallet
}
