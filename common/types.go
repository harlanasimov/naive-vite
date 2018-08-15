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

type Tblock struct {
	Theight    int
	Thash      string
	TpreHash   string
	Tsigner    string
	Ttimestamp time.Time
}

func (self *Tblock) Height() int {
	return self.Theight
}

func (self *Tblock) Hash() string {
	return self.Thash
}

func (self *Tblock) PreHash() string {
	return self.TpreHash
}

func (self *Tblock) Signer() string {
	return self.Tsigner
}
func (self *Tblock) Timestamp() time.Time {
	return self.Ttimestamp
}
func (self *Tblock) SetHash(hash string) {
	self.Thash = hash
}

type AccountStateBlock struct {
	Tblock
	Amount         int // the balance
	ModifiedAmount int
	SnapshotHeight int
	SnapshotHash   string
	BlockType      BlockType // 1: send  2:received
	From           string
	To             string
	SourceHash     string // source Block Thash
}

type SnapshotBlock struct {
	Tblock
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
	block.Theight = height
	block.Thash = hash
	block.TpreHash = preHash
	block.Tsigner = signer
	block.Ttimestamp = timestamp
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
	block.Theight = height
	block.Thash = hash
	block.TpreHash = preHash
	block.Tsigner = signer
	block.Ttimestamp = timestamp
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

type NetMsgType int

const (
	RequestAccountHash    NetMsgType = 102
	RequestSnapshotHash   NetMsgType = 103
	RequestAccountBlocks  NetMsgType = 104
	RequestSnapshotBlocks NetMsgType = 105
	AccountHashes         NetMsgType = 121
	SnapshotHashes        NetMsgType = 122
	AccountBlocks         NetMsgType = 123
	SnapshotBlocks        NetMsgType = 124
)
