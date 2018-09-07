package monitor

import (
	"testing"
	"time"
)

func TestLogTime(t *testing.T) {
	go func() {
		for {
			LogTime("a", "b", time.Now().Add(-time.Second*30))
			time.Sleep(200 * time.Millisecond)
		}
	}()

	go func() {
		t := time.NewTicker(time.Second * 2)
		for {
			select {
			case <-t.C:
				println(Stat())
			}
		}
	}()

	time.Sleep(10 * time.Second)
	go func() {
		for {
			LogTime("a", "b", time.Now().Add(-time.Second*60))
			time.Sleep(200 * time.Millisecond)
		}
	}()

	<-make(chan int)

}
