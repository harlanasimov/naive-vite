package ledger

import (
	"fmt"
	"github.com/viteshan/naive-vite/common"
	"github.com/viteshan/naive-vite/tools"
	"testing"
	"time"
)

func TestTime(t *testing.T) {
	now := time.Now()
	fmt.Printf("%d\n", now.Unix())
	block := common.NewSnapshotBlock(0, "460780b73084275422b520a42ebb9d4f8a8326e1522c79817a19b41ba69dca5b", "", "viteshan", time.Unix(1533550878, 0), nil)
	hash := tools.CalculateSnapshotHash(block)
	fmt.Printf("hash:%s\n", hash)//460780b73084275422b520a42ebb9d4f8a8326e1522c79817a19b41ba69dca5b
}
