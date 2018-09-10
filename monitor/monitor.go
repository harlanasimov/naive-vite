package monitor

import (
	"encoding/json"
	"os"
	"sort"
	"strconv"
	"sync"
	"sync/atomic"
	"time"

	"github.com/vitelabs/go-vite/log15"
)

func init() {
	m = &monitor{r: newRing(60)}
	//设置日志文件地址
	dir := "/Users/jie/log/test.log"
	log15.Info(dir)
	log15.Root().SetHandler(
		log15.LvlFilterHandler(log15.LvlInfo, log15.Must.FileHandler(dir, log15.JsonFormat())),
	)
	logger = log15.New("logtype", "1", "appkey", "govite", "PID", strconv.Itoa(os.Getpid()))
	go loop()
}

var m *monitor

var logger log15.Logger

type monitor struct {
	ms sync.Map
	r  *ring
}

type Msg struct {
	Type string
	Name string
	Cnt  int64
	Sum  int64
}

func (self *Msg) add(i int64) *Msg {
	atomic.AddInt64(&self.Cnt, 1)
	atomic.AddInt64(&self.Sum, i)
	return self
}
func (self *Msg) merge(ms *Msg) *Msg {
	atomic.AddInt64(&self.Cnt, ms.Cnt)
	atomic.AddInt64(&self.Sum, ms.Sum)
	return self
}

func (self *Msg) String() string {
	return "{\"Cnt\":" + strconv.FormatInt(self.Cnt, 10) + ",\"Sum\":" + strconv.FormatInt(self.Sum, 10) + "}"
}

func (self *Msg) reset() *Msg {
	atomic.StoreInt64(&self.Sum, 0)
	atomic.StoreInt64(&self.Cnt, 0)
	return self
}
func (self *Msg) snapshot() Msg {
	return *self
}

func newMsg(t string, name string) *Msg {
	return &Msg{Type: t, Name: name}
}

func key(t string, name string) string {
	return t + "-" + name
}
func LogEvent(t string, name string) {
	log(t, name, 1)
}

func LogTime(t string, name string, tm time.Time) {
	log(t, name, time.Now().Sub(tm).Nanoseconds())
}

func LogDuration(t string, name string, duration int64) {
	log(t, name, duration)
}

func log(t string, name string, i int64) {
	k := key(t, name)
	value, ok := m.ms.Load(k)
	if ok {
		value.(*Msg).add(i)
	} else {
		m.ms.Store(k, newMsg(t, name).add(i))
	}
}

type stat struct {
	Cnt int64
	Avg float64
}

func Stat() []*Msg {
	all := m.r.all()
	msgs := make(map[string]*Msg)
	for _, v := range all {
		msgM := v.(map[string]*Msg)
		for k2, v2 := range msgM {
			tmpM, ok := msgs[k2]
			if ok {
				tmpM.merge(v2)
			} else {
				msgs[k2] = v2
			}
		}
	}

	var r []*Msg
	for _, v := range msgs {
		r = append(r, v)
	}

	sort.Sort(byStr(r))
	return r
}

type byStr []*Msg

func (a byStr) Len() int      { return len(a) }
func (a byStr) Swap(i, j int) { a[i], a[j] = a[j], a[i] }
func (a byStr) Less(i, j int) bool {
	if a[i].Type == a[j].Type {
		return a[i].Name < a[j].Name
	} else {
		return a[i].Type < a[j].Type
	}
}

func StatJson() string {
	all := m.r.all()
	msgs := make(map[string]*Msg)
	for _, v := range all {
		msgM := v.(map[string]*Msg)
		for k2, v2 := range msgM {
			tmpM, ok := msgs[k2]
			if ok {
				tmpM.merge(v2)
			} else {
				s := v2.snapshot()
				msgs[k2] = &s
			}
		}
	}
	r := make(map[string]*stat)
	for k, v := range msgs {
		if v.Cnt != 0 {
			r[k] = &stat{Cnt: v.Cnt, Avg: float64(v.Sum / v.Cnt)}
		}
	}
	b, _ := json.Marshal(r)
	return string(b)
}

func loop() {
	t := time.NewTicker(time.Second * 1)
	for {

		select {
		case <-t.C:
			snapshot := make(map[string]*Msg)

			m.ms.Range(func(k, v interface{}) bool {
				tmpM := v.(*Msg)
				c := tmpM.Cnt
				s := tmpM.Sum
				key := k.(string)
				logger.Info("", "group", "monitor", "interval", 1, "name", key,
					"metric-cnt", c,
					"metric-sum", s,
				)
				sm := tmpM.snapshot()
				snapshot[key] = &sm
				tmpM.reset()
				return true
			})
			m.r.add(snapshot)
		}
	}

}
