package tools

import "sync"

type ForkWall interface {
	Wait()
	Add(delta int)
	Done()
}

type ViteForkWall struct {
	wg sync.WaitGroup
	fw ForkWall
}

func (self *ViteForkWall) Add(delta int) {
	self.wg.Add(delta)
}

func (self *ViteForkWall) Done() {
	self.wg.Done()
}
