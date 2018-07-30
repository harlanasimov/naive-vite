package p2p

import (
	"net/http"

	"sync"

	"net/url"
	"time"

	"encoding/json"

	"strconv"

	"github.com/gorilla/websocket"
	"github.com/vitelabs/go-vite/log"
)

type server struct {
	id       int
	addr     string
	p2p      *p2p
	bootAddr string
}

var upgrader = websocket.Upgrader{} // use default options

func (self *server) ws(w http.ResponseWriter, r *http.Request) {
	c, err := upgrader.Upgrade(w, r, nil)
	if err == nil {
		c.WriteJSON(Req{Id: self.p2p.id, Addr: self.p2p.addr})
		req := Req{}
		c.ReadJSON(&req)
		log.Info("upgrade success, add new peer.", req)
		self.p2p.addPeer(newPeer(req.Id, req.Addr, c))
	} else {
		log.Error("upgrade error.", err)
	}
}

func (self *server) start() {
	//http.HandleFunc("/ws", self.ws)
	mux := http.NewServeMux()
	mux.HandleFunc("/ws", self.ws)
	go http.ListenAndServe(self.addr, mux)
	go self.linkbootnode(self.bootAddr)
}
func (self *server) linkbootnode(bootAddr string) {
	u := url.URL{Scheme: "ws", Host: bootAddr, Path: "/ws"}
	log.Info("boot node connecting to %s", u.String())

	c, _, err := websocket.DefaultDialer.Dial(u.String(), nil)
	if err != nil {
		log.Error("dial:", err)
	}

	defer c.Close()

	done := make(chan struct{})
	c.WriteJSON(&Req{Id: self.id, Addr: self.addr})

	go func() {
		defer close(done)
		for {
			_, message, err := c.ReadMessage()
			if err != nil {
				log.Error("read:", err)
				return
			}
			//log.Info("recv: %s", string(message))
			res := []Req{}
			json.Unmarshal(message, &res)
			for _, r := range res {
				id := r.Id
				addr := r.Addr
				yes := self.p2p.addDial(id, addr)
				if yes {
					log.Info("recv: add dial success.", strconv.Itoa(id), strconv.Itoa(self.id))
				}
			}
		}
	}()

	ticker := time.NewTicker(8 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-done:
			return
		case <-ticker.C:
			err := c.WriteJSON(&Req{Tp: 1})
			if err != nil {
				return
			}
		}
	}
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
		log.Info("upgrade success, add new peer.", req)
		self.p2p.addPeer(newPeer(req.Id, req.Addr, c))
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
	peers        map[int]*peer
	pendingDials map[int]string
	id           int
	addr         string
}

func (self *p2p) addPeer(peer *peer) {
	self.mu.Lock()
	defer self.mu.Unlock()
	self.peers[peer.id] = peer
}

func (self *p2p) start(bootAddr string) {
	self.pendingDials = make(map[int]string)
	self.peers = make(map[int]*peer)
	self.dial = &dial{p2p: self}
	self.server = &server{id: self.id, addr: self.addr, bootAddr: bootAddr, p2p: self}
	self.server.start()

	go func() {
		for {
			time.Sleep(4 * time.Second)

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
		}
	}()
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
