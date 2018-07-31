package p2p

import (
	"net/http"
	"net/url"
	"strconv"
	"sync"
	"time"

	"context"

	"github.com/gorilla/websocket"
	"github.com/viteshan/naive-vite/common/log"
)

type server struct {
	id       int
	addr     string
	p2p      *p2p
	bootAddr string
	srv      *http.Server
}

var upgrader = websocket.Upgrader{} // use default options

func (self *server) ws(w http.ResponseWriter, r *http.Request) {
	c, err := upgrader.Upgrade(w, r, nil)
	if err == nil {
		c.WriteJSON(Req{Id: self.p2p.id, Addr: self.p2p.addr})
		req := Req{}
		c.ReadJSON(&req)
		log.Info("upgrade success, add new peer.%v", req)
		self.p2p.addPeer(newPeer(req.Id, self.p2p.id, req.Addr, c))
	} else {
		log.Error("upgrade error.", err)
	}
}

func (self *server) start() {
	//http.HandleFunc("/ws", self.ws)
	mux := http.NewServeMux()
	mux.HandleFunc("/ws", self.ws)
	srv := &http.Server{Addr: self.addr, Handler: mux}
	self.srv = srv
	go srv.ListenAndServe()
}
func (self *server) loop() {
	self.srv.ListenAndServe()
}

func (self *server) stop() {
	self.srv.Shutdown(context.Background())
}

type dial struct {
	p2p *p2p
}

func (self *dial) connect(addr string) bool {
	u := url.URL{Scheme: "ws", Host: addr, Path: "/ws"}
	c, _, err := websocket.DefaultDialer.Dial(u.String(), nil)
	if err == nil {
		c.WriteJSON(Req{Id: self.p2p.id, Addr: self.p2p.addr})
		req := Req{}
		c.ReadJSON(&req)
		log.Info("client connect success, add new peer.%v", req)
		self.p2p.addPeer(newPeer(req.Id, self.p2p.id, req.Addr, c))
		return true
	} else {
		log.Error("dial error.", err, addr)
		return false
	}
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
	go peer.loop()
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
