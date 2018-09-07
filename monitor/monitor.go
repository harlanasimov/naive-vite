package monitor

import (
	"time"

	"os"
	"strconv"

	"github.com/vitelabs/go-vite/log15"
)

func init() {
	m = &monitor{}
	//设置日志文件地址
	dir := "/Users/jie/log/test.log"
	log15.Info(dir)
	log15.Root().SetHandler(
		log15.LvlFilterHandler(log15.LvlInfo, log15.Must.FileHandler(dir, log15.JsonFormat())),
	)
	logger = log15.New("logtype", "1", "appkey", "govite", "PID", strconv.Itoa(os.Getpid()))

}

var m *monitor

var logger log15.Logger

type monitor struct {
}

func key(t string, name string) string {
	return t + "-" + name
}
func LogEvent(t string, name string) {
	//c := m.cntR.GetOrRegister(key(t, name), metrics.NewMeter()).(metrics.Meter)
	//c.Mark(1)
	logger.Info("", "group", t, "name", name, "metric", 1)
}

func LogTime(t string, name string, tm time.Time) {
	logger.Info("", "group", t, "name", name, "metric", time.Now().Sub(tm).Nanoseconds()/time.Millisecond.Nanoseconds())
}

func Stat() string {
	return "nothing"
}

// 250
