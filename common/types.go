package common

import (
	"strconv"
	"time"
)

type Block interface {
	Height() int
	Hash() string
	PreHash() string
	Signer() string
	Timestamp() time.Time
}

type HashHeight struct {
	Hash   string
	Height int
}

type AccountHashH struct {
	HashHeight
	Addr string
}

func NewAccountHashH(address string, hash string, height int) *AccountHashH {
	self := &AccountHashH{}
	self.Addr = address
	self.Hash = hash
	self.Height = height
	return self
}

type SnapshotPoint struct {
	SnapshotHeight int
	SnapshotHash   string
	AccountHeight  int
	AccountHash    string
}

func (self *SnapshotPoint) Equals(peer *SnapshotPoint) bool {
	if peer == nil {
		return false
	}
	if self.SnapshotHash == peer.SnapshotHash &&
		self.SnapshotHeight == peer.SnapshotHeight &&
		self.AccountHeight == peer.AccountHeight &&
		self.AccountHash == peer.AccountHash {
		return true
	}
	return false
}

func (self *SnapshotPoint) String() string {
	return "[" + strconv.Itoa(self.SnapshotHeight) + "][" + self.SnapshotHash + "][" + strconv.Itoa(self.AccountHeight) + "][" + self.AccountHash + "]"
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

type Address []byte

func HexToAddress(hexStr string) Address {
	return []byte(hexStr)
}

func (self *Address) String() string {
	return string((*self)[:])
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
