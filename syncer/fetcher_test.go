package syncer

import (
	"fmt"
	"github.com/viteshan/naive-vite/common"
	"github.com/viteshan/naive-vite/test"
	"strconv"
	"sync/atomic"
	"testing"
	"time"
)

type senderTest struct {
	receiver *receiver
	store    map[string]common.Block
	times    int32
}

func (self *senderTest) sendA(tasks []hashTask) {
	go func() {
		var blocks []common.Block

		for _, task := range tasks {
			block, ok := self.store[task.hash]
			if ok {
				blocks = append(blocks, block)
			}
		}
		if len(blocks) > 0 {
			self.receiver.handleBlock(blocks)
		}
	}()
}

func (self *senderTest) sendB(task hashTask, prevCnt int) {
	go func() {
		var hashes []hashTask
		height := task.height
		for i := 1; i < prevCnt+1; i++ {
			tmpH := height - i
			hashes = append(hashes, hashTask{height: tmpH, hash: genHashByHeight(tmpH)})
		}
		if len(hashes) > 0 {
			self.receiver.handleHash(hashes)
		}
	}()
}

func (self *senderTest) handle(blocks []common.Block) {
	for _, block := range blocks {
		fmt.Printf("receive block: %v\n", block)
		atomic.CompareAndSwapInt32(&self.times, self.times, self.times+1)
	}
}

func TestFetcher(t *testing.T) {
	N := 10
	receiver := &receiver{}
	sender := &senderTest{receiver: receiver, store: genBlockStore(N)}
	receiver.blockHandle = sender

	fetcher := &fetcher{sender: sender, retryPolicy: &defaultRetryPolicy{fetchedHashs: make(map[string]*RetryStatus)}}
	receiver.fetcher = fetcher

	fetcher.fetchBlockByHash(genFetchHash(N))
	fetcher.fetchBlockByHash(genFetchHash(N + 10))
	fetcher.fetchBlockByHash(genFetchHash(N + 5))
	fetcher.fetchBlockByHash(genFetchHash(N * 2))
	fetcher.fetchHash(hashTask{height: N, hash: genHashByHeight(N)}, N)
	fetcher.fetchHash(hashTask{height: N + 5, hash: genHashByHeight(N + 5)}, N)
	fetcher.fetchHash(hashTask{height: N * 2, hash: genHashByHeight(N * 2)}, N)
	fetcher.fetchHash(hashTask{height: N * 3, hash: genHashByHeight(N * 3)}, N)

	fetcher.fetchHash(hashTask{height: N, hash: genHashByHeight(N)}, N)
	fetcher.fetchHash(hashTask{height: N + 5, hash: genHashByHeight(N + 5)}, N)
	fetcher.fetchHash(hashTask{height: N * 2, hash: genHashByHeight(N * 2)}, N)
	fetcher.fetchHash(hashTask{height: N * 3, hash: genHashByHeight(N * 3)}, N)

	fetcher.fetchHash(hashTask{height: N, hash: genHashByHeight(N)}, N)
	fetcher.fetchHash(hashTask{height: N + 5, hash: genHashByHeight(N + 5)}, N)
	fetcher.fetchHash(hashTask{height: N * 2, hash: genHashByHeight(N * 2)}, N)
	fetcher.fetchHash(hashTask{height: N * 3, hash: genHashByHeight(N * 3)}, N)

	fetcher.fetchHash(hashTask{height: N, hash: genHashByHeight(N)}, N)
	fetcher.fetchHash(hashTask{height: N + 5, hash: genHashByHeight(N + 5)}, N)
	fetcher.fetchHash(hashTask{height: N * 2, hash: genHashByHeight(N * 2)}, N)
	fetcher.fetchHash(hashTask{height: N * 3, hash: genHashByHeight(N * 3)}, N)

	time.Sleep(2 * time.Second)
	if N != int(sender.times) {
		t.Errorf("error result. expect:%d, actual:%d", N, sender.times)
	}
}
func genFetchHash(N int) []hashTask {
	var hashes []hashTask
	for i := 0; i < N; i++ {
		hashes = append(hashes, hashTask{N, genHashByHeight(N)})
	}
	return hashes
}

func genBlockStore(N int) map[string]common.Block {
	hashes := make(map[string]common.Block)
	for i := 0; i < N; i++ {
		height := N + i
		hashes[genHashByHeight(height)] = &test.TestBlock{Theight: height, Thash: genHashByHeight(height), TpreHash: genPrevHashByHeight(height)}
	}
	return hashes
}

func genHashByHeight(height int) string {
	return strconv.Itoa(height - 10)
}

func genPrevHashByHeight(height int) string {
	return strconv.Itoa(height - 11)
}
