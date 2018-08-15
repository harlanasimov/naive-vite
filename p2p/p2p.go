package p2p

type Peer interface {
	Write(msg *Msg) error
	Read() (*Msg, error)
	Id() string
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

type Msg struct {
	t    NetMsgType // type: 2~100 basic msg  101~200:biz msg
	data []byte
}

func NewMsg(t NetMsgType, data []byte) *Msg {
	return &Msg{t: t, data: data}
}

type P2P interface {
	BestPeer() (Peer, error)
	AllPeer() ([]Peer, error)
}
