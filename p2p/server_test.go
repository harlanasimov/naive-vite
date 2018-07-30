package p2p

import (
	"testing"

	"strconv"

	"github.com/vitelabs/go-vite/log"
)

func TestServerStart(t *testing.T) {
	bootAddr := "localhost:8000"

	for i := 0; i < 10; i++ {
		addr := "localhost:808" + strconv.Itoa(i)
		log.Info(addr)
		p2p := p2p{id: i, addr: addr}
		p2p.start(bootAddr)
	}
	ints := make(chan int)
	ints <- 9
}
