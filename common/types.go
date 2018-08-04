package common


type Block interface {
	Height() int
	Hash() string
	PreHash() string
	Signer() string
}

type AccountHashH struct {
	Addr   string
	Hash   string
	Height int
}
