package ledger

import "github.com/viteshan/naive-vite/ledger/pool"

type snapshotFork struct {
	pool pool.PoolReader
}

func (self *snapshotFork) loop() {
	for {
		self.checkFork()
	}
}
func (self *snapshotFork) checkFork() {
	chains := self.pool.Chains()
	current := self.pool.Current()
	longest := self.longestChain(chains, current)
	if longest.ChainId() == current.ChainId() {
		return
	}

	//modifiedAccounts :=
}
func (self *snapshotFork) longestChain(readers []pool.Chain, current pool.Chain) pool.Chain {

	longest := current
	for _, reader := range readers {
		height := reader.HeadHeight()
		if height > longest.HeadHeight() {
			longest = reader
		}
	}
	return longest
}
