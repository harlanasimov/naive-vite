package p2p

import (
	"net/url"
	"strconv"
	"sync"
	"time"

	"encoding/json"

	"github.com/gorilla/websocket"
	"github.com/pkg/errors"
	"github.com/viteshan/naive-vite/common"
	"github.com/viteshan/naive-vite/common/log"
)

type Peer interface {
	Write(msg *Msg) error
	Id() string
	SetState(interface{})
	GetState() interface{}
}

type Msg struct {
	t    common.NetMsgType // type: 2~100 basic msg  101~200:biz msg
	data []byte
}

func (self *Msg) Type() common.NetMsgType {
	return self.t
}
func (self *Msg) Data() []byte {
	return self.data
}

func NewMsg(t common.NetMsgType, data []byte) *Msg {
	return &Msg{t: t, data: data}
}

type MsgHandle func(common.NetMsgType, []byte, Peer)

type P2P interface {
	BestPeer() (Peer, error)
	AllPeer() ([]Peer, error)
	SetHandlerFn(MsgHandle)
}

type p2p struct {
	//peers  []*peer
	mu           sync.Mutex
	server       *server
	dial         *dial
	linker       *linker
	peers        map[int]*peer
	pendingDials map[int]string
	id           int
	addr         string
	closed       chan struct{}
	loopWg       sync.WaitGroup
	msgHandleFn  MsgHandle
}

func (self *p2p) BestPeer() (Peer, error) {
	if len(self.peers) > 0 {
		for _, v := range self.peers {
			return v, nil
		}
	}
	return nil, errors.New("can't find best peer.")
}

func (self *p2p) AllPeer() ([]Peer, error) {
	var result []Peer
	for _, v := range self.peers {
		result = append(result, v)
	}
	if len(result) > 0 {
		return result, nil
	}
	return nil, errors.New("can't find best peer.")
}

func (self *p2p) SetHandlerFn(MsgHandle) {
	panic("implement me")
}

func (self *p2p) addPeer(peer *peer) {
	self.mu.Lock()
	defer self.mu.Unlock()
	old, ok := self.peers[peer.peerId]
	if ok && old != peer {
		log.Warn("peer exist, close old peer: %v", peer.info())
		old.close()
	}
	self.peers[peer.peerId] = peer
	go self.loopPeer(peer)
}
func (self *p2p) loopPeer(peer *peer) {
	self.loopWg.Add(1)
	defer self.loopWg.Done()
	conn := peer.conn
	defer peer.close()
	defer delete(self.peers, peer.peerId)
	for {
		select {
		case <-self.closed:
			log.Info("peer[%s] closed.", peer.info())
			return
		default:
			messageType, p, err := conn.ReadMessage()
			if messageType == websocket.CloseMessage {
				log.Warn("read closed message, peer: %s", peer.info())
				return
			}
			if err != nil {
				log.Error("read message error, peer: %s, err:%v", peer.info(), err)
				return
			}
			if messageType == websocket.BinaryMessage {
				msg := &Msg{}
				err := json.Unmarshal(p, msg)
				if err != nil {
					log.Error("serialize msg fail. messageType:%d, msg:%v", messageType, p)
					continue
				}
				if self.msgHandleFn != nil {
					self.msgHandleFn(msg.t, msg.data, peer)
				}
			}
			log.Info("read message: %s", string(p))
		}
	}
}

func (self *p2p) allPeers() map[int]*peer {
	self.mu.Lock()
	defer self.mu.Unlock()
	result := make(map[int]*peer, len(self.peers))
	for k, v := range self.peers {
		result[k] = v
	}
	return result
}

func (self *p2p) start(bootAddr string) {
	self.pendingDials = make(map[int]string)
	self.peers = make(map[int]*peer)
	self.dial = &dial{p2p: self}
	self.server = &server{id: self.id, addr: self.addr, bootAddr: bootAddr, p2p: self}
	self.linker = newLinker(self, url.URL{Scheme: "ws", Host: bootAddr, Path: "/ws"})
	self.server.start()
	self.linker.start()
	go self.loop()
}

func (self *p2p) loop() {
	self.loopWg.Add(1)
	defer self.loopWg.Done()
	for {
		ticker := time.NewTicker(3 * time.Second)

		select {
		case <-ticker.C:
			for i, v := range self.pendingDials {
				_, ok := self.peers[i]
				if !ok {
					log.Info("node " + strconv.Itoa(self.server.id) + " try to connect to " + strconv.Itoa(i))
					connectted := self.dial.connect(v)
					if connectted {
						log.Info("connect success." + strconv.Itoa(self.server.id) + ":" + strconv.Itoa(i))
						delete(self.pendingDials, i)
					}
				} else {
					log.Info("has connected for " + strconv.Itoa(self.server.id) + ":" + strconv.Itoa(i))
				}
			}
		case <-self.closed:
			log.Info("p2p[%d] closed.", self.id)
			return
		}
	}
}

func (self *p2p) stop() {
	self.linker.stop()
	for _, v := range self.peers {
		v.stop()
	}
	self.server.stop()
	close(self.closed)
	self.loopWg.Wait()

}

func (self *p2p) addDial(id int, addr string) bool {
	self.mu.Lock()
	defer self.mu.Unlock()
	if id == self.id {
		return false
	}
	_, pok := self.peers[id]
	_, dok := self.pendingDials[id]
	if !pok && !dok {
		self.pendingDials[id] = addr
		return true
	}
	return false
}
