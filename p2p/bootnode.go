package p2p

import (
	"encoding/json"
	"net/http"
	"sync"

	"context"
	"os"
	"os/signal"

	"strconv"

	"strings"

	"github.com/viteshan/naive-vite/common/log"
)

type bootnode struct {
	peers  map[int]*peer
	mu     sync.Mutex
	server *http.Server
}

func (self *bootnode) addPeer(peer *peer) {
	self.mu.Lock()
	defer self.mu.Unlock()
	old, ok := self.peers[peer.id]
	if ok && old != peer {
		log.Warn("peer exist, close old peer: %v", strconv.Itoa(peer.id))
		old.close()
	}
	self.peers[peer.id] = peer
	go self.loopread(peer)
}
func (self *bootnode) removePeer(peer *peer) {
	self.mu.Lock()
	defer self.mu.Unlock()
	old, ok := self.peers[peer.id]
	if ok && old == peer {
		log.Info("remove peer %v from bootnode.", strconv.Itoa(peer.id))
		old.close()
		delete(self.peers, peer.id)
	}
}

func (self *bootnode) loopread(peer *peer) {
	conn := peer.conn
	for !peer.closed {
		req := Req{}
		err := conn.ReadJSON(&req)
		if err != nil && strings.Contains(err.Error(), "use of closed network connection") {
			log.Error("read message error, peer: %v", strconv.Itoa(peer.id), err)
			break
		}
		if err != nil {
			log.Error("read message error, peer: %v", strconv.Itoa(peer.id), err)
			continue
		}
		if req.Tp == 1 {
			conn.WriteJSON(self.allPeer())
			continue
		}
	}
}

func (self *bootnode) start(addr string) {
	mux := http.NewServeMux()
	mux.HandleFunc("/ws", self.ws)
	server := &http.Server{Addr: addr, Handler: mux}

	//idleConnsClosed := make(chan struct{})
	go func() {
		sigint := make(chan os.Signal, 1)
		signal.Notify(sigint, os.Interrupt)
		<-sigint

		// We received an interrupt signal, shut down.
		if err := server.Shutdown(context.Background()); err != nil {
			// Error from closing listeners, or context timeout:
			log.Info("HTTP server Shutdown: %v", err)
		}
		//close(idleConnsClosed)
	}()

	go func() {
		if err := server.ListenAndServe(); err != http.ErrServerClosed {
			// Error starting or closing listener:
			log.Info("HTTP server ListenAndServe: %v", err)
		}
	}()
	self.server = server
	//<-idleConnsClosed
}

func (self *bootnode) ws(w http.ResponseWriter, r *http.Request) {
	c, err := upgrader.Upgrade(w, r, nil)
	if err == nil {
		req := Req{}
		err = c.ReadJSON(&req)
		if err != nil {
			log.Info("read fail.", err)
		}
		bytes, _ := json.Marshal(&req)
		log.Info("upgrade success, add new peer.", string(bytes))
		peer := newPeer(req.Id, req.Addr, c)
		closeHandler := c.CloseHandler()
		c.SetCloseHandler(func(code int, text string) error {
			self.removePeer(peer)
			closeHandler(code, text)
			return nil
		})

		self.addPeer(peer)
	} else {
		log.Error("upgrade error.", err)
	}
}

func (self *bootnode) allPeer() []*Peer {
	var results []*Peer
	for _, peer := range self.peers {
		results = append(results, &Peer{Id: peer.id, Addr: peer.addr})
	}
	return results
}
func (self bootnode) stop() {
	for _, peer := range self.peers {
		peer.close()
	}
	self.server.Shutdown(context.Background())

}

type Peer struct {
	Id   int
	Addr string
}

type Req struct {
	Tp   int
	Id   int
	Addr string
}
