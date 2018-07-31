package p2p

import (
	"sort"
	"strconv"
	"testing"
	"time"

	"github.com/viteshan/naive-vite/common/log"
)

func TestServerStart(t *testing.T) {
	bootAddr := "localhost:8000"

	startBootNode(bootAddr)

	N := 10
	var list []*p2p
	for i := 0; i < N; i++ {
		addr := "localhost:808" + strconv.Itoa(i)
		log.Info(addr)
		p2p := p2p{id: i, addr: addr, closed: make(chan struct{})}
		p2p.start(bootAddr)
		list = append(list, &p2p)
	}

	time.Sleep(time.Second * time.Duration(20))
	for _, v := range list {
		allPeers := v.allPeers()
		if !full(v.id, allPeers, N) {
			t.Errorf("error for p2p conn. id:%d, peers:%s.", v.id, allPeers)
		}
	}
	for _, v := range list {
		v.stop()
	}

}
func startBootNode(s string) {
	b := bootnode{peers: make(map[int]*peer)}
	b.start(s)
	time.Sleep(time.Second * time.Duration(2))
}
func full(self int, peers map[int]*peer, N int) bool {
	var keys []int
	for k := range peers {
		keys = append(keys, k)
	}
	keys = append(keys, self)

	if len(keys) != N {
		return false
	}
	sort.Ints(keys)

	for k, v := range keys {
		if k != v {
			return false
		}
	}
	return true
}

func TestFull(t *testing.T) {
	N := 10
	var m = make(map[int]*peer)
	for i := 0; i < N; i++ {
		addr := "localhost:808" + strconv.Itoa(i)
		log.Info(addr)
		peer := peer{peerId: i, selfId: -1}
		m[i] = &peer
	}
	if full(2, m, N) {
		t.Errorf("error not full.")
	}
	delete(m, 2)

	if !full(2, m, N) {
		t.Errorf("error full.")
	}
	if full(3, m, N) {
		t.Errorf("error not full")
	}
}
