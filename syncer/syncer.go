package syncer

type BlockHash struct {
	Height int
	Hash   string
}

type Syncer interface {
	Fetch(hash BlockHash, prevCnt int)
}
