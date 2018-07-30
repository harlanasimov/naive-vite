package p2p

import (
	"strconv"

	"github.com/gorilla/websocket"
	"github.com/vitelabs/go-vite/log"
)

type peer struct {
	id     int
	conn   *websocket.Conn
	addr   string
	closed bool
}

func (self *peer) close() error {
	if !self.closed {
		err := self.conn.Close()
		if err != nil {
			log.Error("close peer error. peer id&addr: %v", strconv.Itoa(self.id)+"&"+self.addr)
		}
		self.closed = true
		return err
	}
	return nil
}

func newPeer(id int, addr string, conn *websocket.Conn) *peer {
	return &peer{id: id, addr: addr, conn: conn, closed: false}
}
