package config

import "github.com/viteshan/naive-vite/common"

type Miner struct {
	Enabled     bool
	HexCoinbase string
}

func (self Miner) CoinBase() common.Address {
	coinbase := common.HexToAddress(self.HexCoinbase)
	return coinbase
}
