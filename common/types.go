package common


type Block interface {
	Height() int
	Hash() string
	PreHash() string
	Signer() string
}
