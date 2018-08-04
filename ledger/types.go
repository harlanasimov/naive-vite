package ledger

import (
	"github.com/viteshan/naive-vite/common"
	"time"
)

//BlockType is the type of Tx described by int.
type BlockType int

// BlockType types.
const (
	SEND BlockType = iota
	RECEIVED
)

var txStr map[BlockType]string

func init() {
	txStr = map[BlockType]string{
		SEND:     "send tx",
		RECEIVED: "received tx",
	}
}

func (self BlockType) String() string {
	if s, ok := txStr[self]; ok {
		return s
	}
	return "Unknown"
}

type block struct {
	common.Block
	height    int
	hash      string
	preHash   string
	signer    string
	timestamp time.Time
}

func (self *block) Height() int {
	return self.height
}

func (self *block) Hash() string {
	return self.hash
}

func (self *block) PreHash() string {
	return self.preHash
}

func (self *block) Signer() string {
	return self.signer
}
func (self *block) Timestamp() time.Time {
	return self.timestamp
}

type AccountStateBlock struct {
	block
	Amount         int // the balance
	ModifiedAmount int
	SnapshotHeight int
	SnapshotHash   string
	BlockType      BlockType // 1: send  2:received
	From           string
	To             string
	SourceHash     string // source Block hash
}

type SnapshotBlock struct {
	block
	Accounts []*common.AccountHashH
}

