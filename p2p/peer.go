package p2p

import (
	"net"
	"sync"

	"strconv"

	"encoding/json"

	"github.com/gorilla/websocket"
	"github.com/viteshan/naive-vite/common/log"
)

type closeOnce struct {
	closed chan struct{}
	once   sync.Once
}

type peer struct {
	closeOnce
	peerId      int
	selfId      int
	peerSrvAddr string
	conn        *websocket.Conn
	remoteAddr  net.Addr
	loopWg      sync.WaitGroup
}

func (self *peer) Write(msg *Msg) error {
	byt, err := json.Marshal(msg)
	if err != nil {
		log.Error("serialize msg fail. err:%v, msg:%v", err, msg)
		return err
	}
	self.conn.WriteMessage(websocket.BinaryMessage, byt)
	return nil
}

func (self *peer) Id() string {
	return string(self.peerId)
}

func (self *peer) close() {
	self.once.Do(self.realClose)
}
func (self *peer) realClose() {
	close(self.closed)
	self.conn.Close()
}

//func (self *peer) loop() {
//	conn := self.conn
//	defer self.close()
//	self.loopWg.Add(1)
//	defer self.loopWg.Done()
//	for {
//		select {
//		case <-self.closed:
//			log.Info("peer[%s] closed.", self.info())
//			return
//		default:
//			messageType, p, err := conn.ReadMessage()
//			if messageType == websocket.CloseMessage {
//				log.Warn("read closed message, peer: %s", self.info())
//				return
//			}
//			if err != nil {
//				log.Error("read message error, peer: %s, err:%v", self.info(), err)
//				return
//			}
//			log.Info("read message: %s", string(p))
//		}
//	}
//}
func (self *peer) stop() {
	self.close()
	self.loopWg.Wait()
}

func newPeer(fromId int, toId int, peerSrvAddr string, conn *websocket.Conn) *peer {
	c := conn.CloseHandler()
	remoteAddr := conn.RemoteAddr()
	peer := &peer{peerId: fromId, selfId: toId, peerSrvAddr: peerSrvAddr, conn: conn, remoteAddr: remoteAddr}
	peer.closed = make(chan struct{})
	conn.SetCloseHandler(func(code int, text string) error {
		log.Info("peer received closed msg. %s, %v", peer.info(), remoteAddr)
		return c(code, text)
	})
	return peer
}

func (self *peer) info() string {
	return "[" + strconv.Itoa(self.selfId) + "]-[" + strconv.Itoa(self.peerId) + "]"
}
