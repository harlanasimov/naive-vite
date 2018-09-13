package config

import (
	"github.com/pkg/errors"
	"github.com/viteshan/naive-vite/common"
)

type Miner struct {
	Enabled     bool   `yaml:"enabled"`
	HexCoinbase string `yaml:"coinbase"`
}

func (self *Miner) CoinBase() common.Address {
	coinbase := common.HexToAddress(self.HexCoinbase)
	return coinbase
}
func (self *Miner) Check(cfg *Base) error {
	if !self.Enabled {
		return nil
	}
	if len(self.CoinBase().String()) == 0 {
		return errors.New("miner coinbase must be set when miner enabled")
	}
	return nil
}
