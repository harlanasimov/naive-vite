package common

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/viteshan/naive-vite/common/log"
)

func TestBlock(t *testing.T) {
	viteshan := "viteshan"
	block := NewAccountBlock(1, "Thash...", "TpreHash...", viteshan, time.Unix(1533550878, 0),
		0, -105, 10, "snapshotHash...", SEND, viteshan, viteshan, "")
	bytes, _ := json.Marshal(block)

	log.Info(string(bytes))
	stateBlock := &Tblock{}
	json.Unmarshal(bytes, stateBlock)
	log.Info("%v", stateBlock)
}

// {"Amount":0,"ModifiedAmount":-105,"SnapshotHeight":10,"SnapshotHash":"snapshotHash...","BlockType":0,"From":"viteshan","To":"viteshan","SourceHash":""}
// {"Theight":1,"Thash":"Thash...","TpreHash":"TpreHash...","Tsigner":"viteshan","Ttimestamp":"2018-08-06T18:21:18+08:00","Amount":0,"ModifiedAmount":-105,"SnapshotHeight":10,"SnapshotHash":"snapshotHash...","BlockType":0,"From":"viteshan","To":"viteshan","SourceHash":""}
