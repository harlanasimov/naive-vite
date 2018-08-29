package config

type P2P struct {
	NodeId       string
	Port         int
	NetId        int
	LinkBootAddr string
}

type Boot struct {
	BootAddr string
}
