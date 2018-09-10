package face

import "strconv"

type FetchRequest struct {
	Chain   string
	Height  int
	Hash    string
	PrevCnt int
}

func (self *FetchRequest) String() string {
	return self.Chain + "," + strconv.Itoa(self.Height) + "," + self.Hash + "," + strconv.Itoa(self.PrevCnt)
}
