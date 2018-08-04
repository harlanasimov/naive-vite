package tools

import (
	"fmt"
	"sync"
	"testing"
	"time"
)

func TestRWLock(t *testing.T) {
	var mutex sync.RWMutex
	fmt.Println("------start-------")
	go func() {
		fmt.Println("----rlock---------")
		mutex.RLock()
		fmt.Println("----rlock done---------")
		fmt.Println("----lock---------")
		mutex.Lock()
		fmt.Println("----lock done---------")

		fmt.Println("-------------")
	}()

	time.Sleep(time.Second * 2)

	fmt.Println("----unrlock---------")
	mutex.RUnlock()
	fmt.Println("----unrlock done---------")
	fmt.Println("----unlock---------")
	mutex.Unlock()
	fmt.Println("----unlock done---------")
}
