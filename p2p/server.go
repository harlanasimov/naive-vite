package p2p

import (
	"context"
	"net/http"
	"net/url"

	"github.com/gorilla/websocket"
	"github.com/viteshan/naive-vite/common/log"
)

type server struct {
	id       string
	addr     string
	p2p      *p2p
	bootAddr string
	srv      *http.Server
}

var upgrader = websocket.Upgrader{} // use default options

func (self *server) ws(w http.ResponseWriter, r *http.Request) {
	c, err := upgrader.Upgrade(w, r, nil)
	if err == nil {
		c.WriteJSON(bootReq{Id: self.p2p.id, Addr: self.p2p.addr})
		req := bootReq{}
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
		c.WriteJSON(bootReq{Id: self.p2p.id, Addr: self.p2p.addr})
		req := bootReq{}
		c.ReadJSON(&req)
		log.Info("client connect success, add new peer.%v", req)
		self.p2p.addPeer(newPeer(req.Id, self.p2p.id, req.Addr, c))
		return true
	} else {
		log.Error("dial error.", err, addr)
		return false
	}
}
