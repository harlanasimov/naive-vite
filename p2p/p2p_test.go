package p2p

import (
	"sort"
	"strconv"
	"testing"
	"time"

	"github.com/viteshan/naive-vite/common/config"
	"github.com/viteshan/naive-vite/common/log"
)

func TestP2P(t *testing.T) {
	bootAddr := "localhost:8000"

	startBootNode(bootAddr)

	N := 10
	var list []P2P
	for i := 0; i < N; i++ {
		addr := "localhost:808" + strconv.Itoa(i)
		log.Info(addr)
		config := config.P2P{NodeId: strconv.Itoa(i), Port: 8080 + i, LinkBootAddr: bootAddr, NetId: 0}
		p2p := NewP2P(config)
		p2p.Start()
		list = append(list, p2p)
	}

	time.Sleep(time.Second * time.Duration(20))
	for _, v := range list {
		allPeers, _ := v.AllPeer()
		if !peerFull(v.Id(), allPeers, N) {
			result := ""
			for _, v := range allPeers {
				result += v.Id() + ":"
			}
			t.Errorf("error for p2p conn. id:%s, peers:%s.", v.Id(), result)
		}
	}
	for _, v := range list {
		v.Stop()
	}
}

func peerFull(self string, peers []Peer, N int) bool {
	var keys []string
	for _, v := range peers {
		keys = append(keys, v.Id())
	}
	keys = append(keys, self)

	if len(keys) != N {
		return false
	}
	sort.Strings(keys)

	for k, v := range keys {
		if strconv.Itoa(k) != v {
			return false
		}
	}
	return true
}
