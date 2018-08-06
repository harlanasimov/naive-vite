package common

import "time"

type Block interface {
	Height() int
	Hash() string
	PreHash() string
	Signer() string
	Timestamp() time.Time
}

type AccountHashH struct {
	Addr   string
	Hash   string
	Height int
}

//BlockType is the type of Tx described by int.
type BlockType int

// BlockType types.
const (
	SEND BlockType = iota
	RECEIVED
	CREATE
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
	Block
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
func (self *block) SetHash(hash string) {
	self.hash = hash
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
	Accounts []*AccountHashH
}

func NewSnapshotBlock(
	height int,
	hash string,
	preHash string,
	signer string,
	timestamp time.Time,
	accounts []*AccountHashH,
) *SnapshotBlock {

	block := &SnapshotBlock{}
	block.height = height
	block.hash = hash
	block.preHash = preHash
	block.signer = signer
	block.timestamp = timestamp
	block.Accounts = accounts
	return block
}

func NewAccountBlock(
	height int,
	hash string,
	preHash string,
	signer string,
	timestamp time.Time,

	amount int,
	modifiedAmount int,
	snapshotHeight int,
	snapshotHash string,
	blockType BlockType,
	from string,
	to string,
	sourceHash string,
) *AccountStateBlock {

	block := &AccountStateBlock{}
	block.height = height
	block.hash = hash
	block.preHash = preHash
	block.signer = signer
	block.timestamp = timestamp
	block.Amount = amount
	block.ModifiedAmount = modifiedAmount
	block.SnapshotHash = snapshotHash
	block.SnapshotHeight = snapshotHeight
	block.BlockType = blockType
	block.From = from
	block.To = to
	block.SourceHash = sourceHash
	return block
}
