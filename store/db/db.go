package db

type DB interface {
	Put(key []byte, val []byte)
	Get(key []byte)
	Del(key []byte)
}
