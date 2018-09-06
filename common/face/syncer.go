package face

type FetchRequest struct {
	Chain   string
	Height  int
	Hash    string
	PrevCnt int
}
