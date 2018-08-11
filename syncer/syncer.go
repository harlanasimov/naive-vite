package syncer

import "github.com/viteshan/naive-vite/common"

type BlockHash struct {
	Height int
	Hash   string
}

type Syncer interface {
	Fetch(hash common.HashHeight, prevCnt int)
}
