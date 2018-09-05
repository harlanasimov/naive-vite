package tools

import (
	"crypto/sha256"
	"encoding/hex"
	"strconv"

	"github.com/viteshan/naive-vite/common"
)

// SHA256 hasing
// calculateHash is a simple SHA256 hashing function
func calculateHash(s string) string {
	h := sha256.New()
	h.Write([]byte(s))
	hashed := h.Sum(nil)
	return hex.EncodeToString(hashed)
}

func CalculateAccountHash(block *common.AccountStateBlock) string {
	return calculateHash(blockStr(block) + strconv.Itoa(block.Amount) +
		strconv.Itoa(block.ModifiedAmount) +
		strconv.Itoa(block.SnapshotHeight) +
		block.SnapshotHash +
		block.BlockType.String() +
		block.From +
		block.To +
		block.SourceHash)
}

func blockStr(block common.Block) string {
	return strconv.FormatInt(block.Timestamp().Unix(), 10) + string(block.Signer()) + string(block.PreHash()) + strconv.Itoa(block.Height())
}

func CalculateSnapshotHash(block *common.SnapshotBlock) string {
	accStr := ""
	if block.Accounts != nil {
		for _, account := range block.Accounts {
			accStr = accStr + strconv.Itoa(account.Height) + account.Hash + account.Addr
		}
	}
	record := blockStr(block) + accStr
	return calculateHash(record)
}
