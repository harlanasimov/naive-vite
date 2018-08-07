package miner

import (
	"github.com/asaskevich/EventBus"
	"github.com/vitelabs/go-vite/common/types"
	"github.com/viteshan/naive-vite/common"
	"github.com/viteshan/naive-vite/common/log"
	"github.com/viteshan/naive-vite/consensus"
	"sync/atomic"
	"time"
)

// Package miner implements vite block creation

// Backend wraps all methods required for mining.
type SnapshotChainRW interface {
	MiningSnapshotBlock(address string, timestamp int64) error
}

type DownloaderRegister func(chan<- int) // 0 represent success, not 0 represent failed.

/**

0->1->2->3->4->5->6->7->8
		 ^|_______\
*/
// 0:origin 1: initing 2:inited 3:starting 4:started 5:stopping 6:stopped 7:destroying 8:destroyed
type MinerLifecycle struct {
	types.LifecycleStatus
}

func (self *MinerLifecycle) PreDestroy() bool {
	return atomic.CompareAndSwapInt32(&self.Status, 6, 7)
}
func (self *MinerLifecycle) PostDestroy() bool {
	return atomic.CompareAndSwapInt32(&self.Status, 7, 8)
}

func (self *MinerLifecycle) PreStart() bool {
	return atomic.CompareAndSwapInt32(&self.Status, 2, 3) || atomic.CompareAndSwapInt32(&self.Status, 6, 3)
}
func (self *MinerLifecycle) PostStart() bool {
	return atomic.CompareAndSwapInt32(&self.Status, 3, 4)
}

type Miner struct {
	MinerLifecycle
	chain       SnapshotChainRW
	mining      int32
	coinbase    common.Address // address
	worker      *worker
	committee   *consensus.Committee
	mem         *consensus.SubscribeMem
	bus         EventBus.Bus
	dwlFinished bool
}

func NewMiner(chain SnapshotChainRW, bus EventBus.Bus, coinbase common.Address, committee *consensus.Committee) *Miner {
	miner := &Miner{chain: chain, coinbase: coinbase}

	miner.committee = committee
	miner.mem = &consensus.SubscribeMem{Mem: miner.coinbase, Notify: make(chan time.Time)}
	miner.worker = &worker{chain: chain, workChan: miner.mem.Notify, coinbase: coinbase}
	miner.bus = bus
	miner.dwlFinished = false
	return miner
}
func (self *Miner) Init() {
	self.PreInit()
	defer self.PostInit()
	self.worker.Init()
	dwlDownFn := func() {
		log.Info("downloader success.")
		self.dwlFinished = true
		self.committee.Subscribe(self.mem)
	}
	self.bus.SubscribeOnce(common.DwlDone, dwlDownFn)
}

func (self *Miner) Start() {
	self.PreStart()
	defer self.PostStart()

	if self.dwlFinished {
		self.committee.Subscribe(self.mem)
	}
	self.worker.Start()
}

func (self *Miner) Stop() {
	self.PreStop()
	defer self.PostStop()

	self.worker.Stop()
	self.committee.Subscribe(nil)
}

func (self *Miner) Destroy() {
	self.PreDestroy()
	defer self.PostDestroy()
}
