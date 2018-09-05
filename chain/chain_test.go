package chain

import (
	"testing"

	"fmt"

	"time"

	"github.com/viteshan/naive-vite/common"
	"github.com/viteshan/naive-vite/tools"
)

func TestGenesis(t *testing.T) {

	var genesisAccounts = []string{"viteshan", "jie"}
	for _, a := range genesisAccounts {
		genesis := common.NewAccountBlock(0, "", "", a, time.Unix(0, 0),
			200, 0, 0, "", common.GENESIS, a, a, "")
		genesis.SetHash(tools.CalculateAccountHash(genesis))

		fmt.Println(a, &common.HashHeight{Hash: genesis.Hash(), Height: genesis.Height()})
	}

	var genesisAcc = []*common.AccountHashH{
		{common.HashHeight{Hash: "9f4e832965c767166ca700c703ed91dc742958ad164ab0b63c875f22753a8d11", Height: 0}, "viteshan"},
		{common.HashHeight{Hash: "f3e4cf54cc629262e2ab6845544ba338f50095dac941eb2c0c2ed2d611b9b498", Height: 0}, "jie"},
	}

	var genesisSnapshot = common.NewSnapshotBlock(0, "1ad542792093c08518832fa644a4f3f2f1e54dcf6111879d8c6f2862e6ba1179", "", "viteshan", time.Unix(1533550878, 0), genesisAcc)
	fmt.Println(tools.CalculateSnapshotHash(genesisSnapshot))
}
