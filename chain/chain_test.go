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
			200, 0, 0, "", common.GENESIS, a, a, "", -1)
		genesis.SetHash(tools.CalculateAccountHash(genesis))

		fmt.Println(a, &common.HashHeight{Hash: genesis.Hash(), Height: genesis.Height()})
	}

	var genesisAcc = []*common.AccountHashH{
		{common.HashHeight{Hash: "71cef9aa8a67df1055a0801e2fc251fe14a298b8fc1098a54e27db31f7a75e20", Height: 0}, "viteshan"},
		{common.HashHeight{Hash: "c3e7ab31e4834cf18df6705b96c655c845db44f4ee315c6fba5232b4f965fcf0", Height: 0}, "jie"},
	}

	var genesisSnapshot = common.NewSnapshotBlock(0, "b5a3ee58d163c283e5c8c0f65ff5b26a5cd64cb9dce8119ac4581ebcb54626fd", "", "viteshan", time.Unix(1533550878, 0), genesisAcc)
	fmt.Println(tools.CalculateSnapshotHash(genesisSnapshot))
}
