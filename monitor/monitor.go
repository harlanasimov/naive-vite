package monitor

import (
	"strconv"
	"time"
)

func init() {
	m = &monitor{}
	go m.loopStats()
}

var m *monitor

type monitor struct {
	event int
}

func LogEvent(t string, name string) {
	m.event++
}
func Stat() string {
	return strconv.Itoa(int(m.event / 10))
}

func (self *monitor) loopStats() {
	t := time.NewTicker(time.Second * 1)
	for {
		select {
		case <-t.C:
			self.event -= self.event / 10
		}
	}
}
