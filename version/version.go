package version

import (
	"github.com/viteshan/naive-vite/common/log"
	"sync/atomic"
)

var forkVersion int32

func ForkVersion() int {
	return int(forkVersion)
}

func IncForkVersion() {
	for {
		i := forkVersion
		if atomic.CompareAndSwapInt32(&forkVersion, i, i+1) {
			return
		} else {
			log.Info("fork version concurrent for %d.", i)
		}
	}
}
