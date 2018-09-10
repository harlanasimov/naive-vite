package main

import (
	"strconv"

	"encoding/json"

	"net"

	"net/http"

	"github.com/abiosoft/ishell"
	"github.com/google/gops/agent"
	"github.com/viteshan/naive-vite/common/config"
	"github.com/viteshan/naive-vite/common/log"
	"github.com/viteshan/naive-vite/consensus"
	"github.com/viteshan/naive-vite/monitor"
	"github.com/viteshan/naive-vite/node"
	"github.com/viteshan/naive-vite/p2p"
)
import (
	_ "net/http/pprof"
)

func main() {
	if err := agent.Listen(agent.Options{}); err != nil {
		log.Fatal("%v", err)
	}
	log.InitPath()
	// create new shell.
	// by default, new shell includes 'exit', 'help' and 'clear' commands.
	shell := ishell.New()

	// display welcome info.
	shell.Println("naive-vite Interactive Shell")

	defaultBootAddr := "localhost:8000"
	// subcommands and custom autocomplete.
	{
		var bootNode p2p.Boot
		autoCmd := &ishell.Cmd{
			Name: "boot",
			Help: "start or stop boot node.",
		}
		autoCmd.AddCmd(&ishell.Cmd{
			Name: "start",
			Help: "start boot node.",
			Func: func(c *ishell.Context) {
				if bootNode != nil {
					c.Println("boot has started.")
					return
				}
				c.ShowPrompt(false)
				defer c.ShowPrompt(true)
				c.Print("BootAddr: ")
				bootAddr := c.ReadLine()

				if bootAddr == "" {
					bootAddr = defaultBootAddr
				}
				bootNode = startBoot(bootAddr)
				c.Println("boot start for[" + bootAddr + "] successfully.")
			},
		})

		autoCmd.AddCmd(&ishell.Cmd{
			Name: "stop",
			Help: "stop boot node.",
			Func: func(c *ishell.Context) {
				if bootNode == nil {
					c.Println("boot has stopped.")
					return
				}
				bootNode.Stop()
				bootNode = nil
				c.Println("boot stop successfully.")
			},
		})

		autoCmd.AddCmd(&ishell.Cmd{
			Name: "list",
			Help: "list linked node info.",
			Func: func(c *ishell.Context) {
				if bootNode == nil {
					c.Println("boot should be started.")
					return
				}
				all := bootNode.All()
				c.Printf("-----boot links -----\n")
				c.Println("Id\tAddr")

				for _, p := range all {
					c.Printf("%s\t%s\n", p.Id, p.Addr)
				}
			},
		})

		shell.AddCmd(autoCmd)
	}

	var node node.Node
	{
		autoCmd := &ishell.Cmd{
			Name: "node",
			Help: "start or stop node.",
		}
		autoCmd.AddCmd(&ishell.Cmd{
			Name: "start",
			Help: "start node.",
			Func: func(c *ishell.Context) {
				if node != nil {
					c.Println("node has started.")
					return
				}
				c.ShowPrompt(false)
				defer c.ShowPrompt(true)

				c.Print("NodeId: ")
				id := c.ReadLine()

				c.Print("Port: ")
				port, _ := strconv.Atoi(c.ReadLine())

				c.Print("BootAddr: ")
				bootAddr := c.ReadLine()

				addr, e := net.ResolveTCPAddr("tcp4", bootAddr)

				if bootAddr == "" || e != nil {
					c.Printf("input boot address error, use default bootAddr[%s].\n", defaultBootAddr)
					bootAddr = defaultBootAddr
				} else {
					bootAddr = addr.String()
				}

				node = startNode(bootAddr, port, id)
				c.Println("node start for[" + bootAddr + "] successfully.")
			},
		})

		autoCmd.AddCmd(&ishell.Cmd{
			Name: "stop",
			Help: "stop node.",
			Func: func(c *ishell.Context) {
				if node == nil {
					c.Println("node has stopped.")
					return
				}
				node.Stop()
				node = nil
				c.Println("node stop successfully.")
			},
		})

		autoCmd.AddCmd(&ishell.Cmd{
			Name: "peers",
			Help: "list peers for node.",
			Func: func(c *ishell.Context) {
				if node == nil {
					c.Println("node must be started.")
					return
				}
				net := node.P2P()
				peers, _ := net.AllPeer()
				c.Printf("-----net peers -----\n")
				c.Println("Id\tRemote\tState")

				for _, p := range peers {
					bt, _ := json.Marshal(p.GetState())
					c.Printf("%s\t%s\t%s\n", p.Id(), p.RemoteAddr(), string(bt))
				}
			},
		})

		shell.AddCmd(autoCmd)
	}

	{
		autoCmd := &ishell.Cmd{
			Name: "miner",
			Help: "start or stop node.",
		}
		autoCmd.AddCmd(&ishell.Cmd{
			Name: "start",
			Help: "start miner.",
			Func: func(c *ishell.Context) {
				if node == nil {
					c.Println("node should be started.")
					return
				}
				if node.Wallet().CoinBase() == "" {
					c.Println("please set coinBase.")
					return
				}
				node.StartMiner()
				c.Println("miner start successfully.")
			},
		})

		autoCmd.AddCmd(&ishell.Cmd{
			Name: "stop",
			Help: "stop miner.",
			Func: func(c *ishell.Context) {
				if node == nil {
					c.Println("node should be stopped.")
					return
				}
				node.StopMiner()
				node = nil
				c.Println("miner stop successfully.")
			},
		})
		shell.AddCmd(autoCmd)
	}

	{
		autoCmd := &ishell.Cmd{
			Name: "account",
			Help: "start or stop node.",
		}
		//[list,create,balance,send,receive]
		//autoCmd.AddCmd(&ishell.Cmd{
		//	Name: "list",
		//	Help: "list accounts.",
		//	Func: func(c *ishell.Context) {
		//		if node == nil {
		//			c.Println("node should be started.")
		//			return
		//		}
		//		node.Leger().ListRequest()
		//		node.StartMiner()
		//		c.Println("miner start successfully.")
		//	},
		//})

		autoCmd.AddCmd(&ishell.Cmd{
			Name: "set",
			Help: "set address.",
			Func: func(c *ishell.Context) {
				if node == nil {
					c.Println("node should be started.")
					return
				}
				address := ""
				if len(c.Args) == 1 {
					address = c.Args[0]
				} else {
					c.ShowPrompt(false)
					defer c.ShowPrompt(true)
					c.Print("Address: ")
					address = c.ReadLine()
				}
				if address == "" {
					c.Println("address is empty.")
					return
				}
				node.Wallet().SetCoinBase(address)
				c.Println("set address successfully.")
			},
		})

		autoCmd.AddCmd(&ishell.Cmd{
			Name: "create",
			Help: "create account.",
			Func: func(c *ishell.Context) {
				if node == nil {
					c.Println("node should be stopped.")
					return
				}
				c.ShowPrompt(false)
				defer c.ShowPrompt(true)
				c.Print("Address: ")
				address := c.ReadLine()

				if address == "" {
					c.Println("address is empty.")
					return
				}
				node.Wallet().CreateAccount(address)
				c.Println("create address successfully.")
			},
		})

		autoCmd.AddCmd(&ishell.Cmd{
			Name: "balance",
			Help: "start miner.",
			Func: func(c *ishell.Context) {
				if node == nil {
					c.Println("node should be started.")
					return
				}
				base := node.Wallet().CoinBase()
				if base == "" {
					c.Println("please set current address.")
					return
				}
				balance := node.Leger().GetAccountBalance(base)
				c.Println("balance is ", balance)
			},
		})

		autoCmd.AddCmd(&ishell.Cmd{
			Name: "send",
			Help: "send tx.",
			Func: func(c *ishell.Context) {
				if node == nil {
					c.Println("node should be stopped.")
					return
				}
				if node.Wallet().CoinBase() == "" {
					c.Println("please set coinBase.")
					return
				}
				c.ShowPrompt(false)
				defer c.ShowPrompt(true)
				c.Print("to Address: ")
				toAddress := c.ReadLine()

				if toAddress == "" {
					c.Println("to address is empty.")
					return
				}
				c.Print("to Amount: ")
				amount, _ := strconv.Atoi(c.ReadLine())

				err := node.Leger().RequestAccountBlock(node.Wallet().CoinBase(), toAddress, -amount)
				if err != nil {
					c.Println("send tx fail.", err)
				} else {
					c.Println("send tx success.")
				}
			},
		})
		autoCmd.AddCmd(&ishell.Cmd{
			Name: "receive",
			Help: "receive tx.",
			Func: func(c *ishell.Context) {
				if node == nil {
					c.Println("node should be started.")
					return
				}
				if node.Wallet().CoinBase() == "" {
					c.Println("please set coinBase.")
					return
				}

				c.ShowPrompt(false)
				defer c.ShowPrompt(true)
				c.Print("from Address: ")
				fromAddress := c.ReadLine()

				if fromAddress == "" {
					c.Println("from address is empty.")
					return
				}
				c.Print("from block hash: ")
				reqHash := c.ReadLine()
				if reqHash == "" {
					c.Println("from hash is empty.")
					return
				}

				err := node.Leger().ResponseAccountBlock(fromAddress, node.Wallet().CoinBase(), reqHash)
				if err != nil {
					c.Println("receive tx fail.", err)
				} else {
					c.Println("receive tx success.")
				}
			},
		})

		shell.AddCmd(autoCmd)
	}

	{
		autoCmd := &ishell.Cmd{
			Name: "ablock",
			Help: "get account block info.",
		}
		autoCmd.AddCmd(&ishell.Cmd{
			Name: "list",
			Help: "list account block.",
			Func: func(c *ishell.Context) {
				if node == nil {
					c.Println("node should be started.")
					return
				}
				addr := node.Wallet().CoinBase()
				if len(c.Args) == 1 {
					addr = c.Args[0]
				}
				c.Printf("-----address[%s] blocks-----\n", addr)
				c.Println("Height\tHash\tPrevHash")
				blocks := node.Leger().ListAccountBlock(addr)
				for _, b := range blocks {
					c.Printf("%d\t%s\t%s\n", b.Height(), b.Hash(), b.PreHash())
				}
			},
		})

		autoCmd.AddCmd(&ishell.Cmd{
			Name: "head",
			Help: "head account block.",
			Func: func(c *ishell.Context) {
				if node == nil {
					c.Println("node should be stopped.")
					return
				}
				addr := node.Wallet().CoinBase()
				if len(c.Args) == 1 {
					addr = c.Args[0]
				}

				head, _ := node.Leger().Chain().HeadAccount(addr)

				if head != nil {
					c.Printf("head info, height:%d, hash:%s, prev:%s\n", head.Height(), head.Hash(), head.PreHash())
				} else {
					c.Println("head is empty.")
				}

			},
		})

		autoCmd.AddCmd(&ishell.Cmd{
			Name: "reqs",
			Help: "account requests.",
			Func: func(c *ishell.Context) {
				if node == nil {
					c.Println("node should be stopped.")
					return
				}
				addr := node.Wallet().CoinBase()
				if len(c.Args) == 1 {
					addr = c.Args[0]
				}
				c.Printf("-----address[%s] request blocks-----\n", addr)
				c.Println("From\tAmount\tReqHash")
				blocks := node.Leger().ListRequest(addr)
				for _, b := range blocks {
					c.Printf("%s\t%d\t%s\n", b.From, b.Amount, b.ReqHash)
				}
			},
		})

		autoCmd.AddCmd(&ishell.Cmd{
			Name: "detail",
			Help: "detail for account block.",
			Func: func(c *ishell.Context) {
				if node == nil {
					c.Println("node should be stopped.")
					return
				}
				height := -1
				addr := node.Wallet().CoinBase()
				if len(c.Args) == 2 {
					addr = c.Args[0]
					height, _ = strconv.Atoi(c.Args[1])
				}
				block := node.Leger().Chain().GetAccountByHeight(addr, height)
				bytes, _ := json.Marshal(block)
				c.Printf("detail info, block:%s\n", string(bytes))
			},
		})

		shell.AddCmd(autoCmd)
	}

	{
		autoCmd := &ishell.Cmd{
			Name: "sblock",
			Help: "get snapshot block info.",
		}
		autoCmd.AddCmd(&ishell.Cmd{
			Name: "list",
			Help: "list snapshot blocks.",
			Func: func(c *ishell.Context) {

				c.Printf("-----snapshot blocks-----\n")
				c.Println("Height\tHash\tPrevHash\tAccountLen\tTime")
				blocks := node.Leger().ListSnapshotBlock()
				for _, b := range blocks {
					c.Printf("%d\t%s\t%s\t%d\t%s\n", b.Height(), b.Hash(), b.PreHash(), len(b.Accounts), b.Timestamp().Format("15:04:05"))
				}
			},
		})

		autoCmd.AddCmd(&ishell.Cmd{
			Name: "head",
			Help: "head snapshot block.",
			Func: func(c *ishell.Context) {
				if node == nil {
					c.Println("node should be stopped.")
					return
				}
				head, _ := node.Leger().Chain().HeadSnapshot()

				c.Printf("head info, height:%d, hash:%s, prev:%s\n", head.Height(), head.Hash(), head.PreHash())
			},
		})

		autoCmd.AddCmd(&ishell.Cmd{
			Name: "detail",
			Help: "detail for snapshot block.",
			Func: func(c *ishell.Context) {
				if node == nil {
					c.Println("node should be stopped.")
					return
				}
				height := -1
				if len(c.Args) == 1 {
					height, _ = strconv.Atoi(c.Args[0])
				}
				block := node.Leger().Chain().GetSnapshotByHeight(height)
				bytes, _ := json.Marshal(block)
				c.Printf("detail info, block:%s\n", string(bytes))
			},
		})

		shell.AddCmd(autoCmd)
	}

	{
		autoCmd := &ishell.Cmd{
			Name: "pool",
			Help: "print pool info info.",
		}
		autoCmd.AddCmd(&ishell.Cmd{
			Name: "sprint",
			Help: "print snapshot pool info.",
			Func: func(c *ishell.Context) {
				if node == nil {
					c.Println("node should be started.")
					return
				}
				info := node.Leger().Pool().Info("")
				c.Println(info)
			},
		})

		autoCmd.AddCmd(&ishell.Cmd{
			Name: "aprint",
			Help: "print account pool info.",
			Func: func(c *ishell.Context) {
				if node == nil {
					c.Println("node should be stopped.")
					return
				}
				if len(c.Args) == 1 {
					addr := c.Args[0]
					info := node.Leger().Pool().Info(addr)
					c.Println(info)
				} else {
					c.Println("aprint [addr]")
				}
			},
		})

		shell.AddCmd(autoCmd)
	}

	{
		autoCmd := &ishell.Cmd{
			Name: "monitor",
			Help: "print stat info.",
		}
		autoCmd.AddCmd(&ishell.Cmd{
			Name: "stat",
			Help: "print monitor stat info.",
			Func: func(c *ishell.Context) {
				stat := monitor.Stat()
				for _, v := range stat {
					if v.Cnt > 0 {
						c.Printf("%s-%s\t %d, %f", v.Type, v.Name, v.Cnt, float64(float64(v.Sum)/float64(v.Cnt)))
						c.Println()
					}
				}
			},
		})

		shell.AddCmd(autoCmd)
	}

	{
		autoCmd := &ishell.Cmd{
			Name: "profile",
			Help: "profile.",
		}
		autoCmd.AddCmd(&ishell.Cmd{
			Name: "start",
			Help: "print monitor stat info.",
			Func: func(c *ishell.Context) {
				port := "6060"
				if len(c.Args) == 1 {
					port = c.Args[0]
				}
				go func() {
					http.ListenAndServe("localhost:"+port, nil)
				}()
			},
		})

		shell.AddCmd(autoCmd)
	}

	// run shell
	shell.Run()
}
func startNode(bootAddr string, port int, nodeId string) node.Node {
	cfg := config.Node{
		P2pCfg:       config.P2P{NodeId: nodeId, Port: port, LinkBootAddr: bootAddr, NetId: 0},
		ConsensusCfg: config.Consensus{Interval: 1, MemCnt: len(consensus.DefaultMembers)},
	}
	n := node.NewNode(cfg)
	n.Init()
	n.Start()
	return n
}

func checkArgs(args []string) (bool, string) {
	if len(args) != 1 {
		return false, ""
	}
	return true, args[0]
}

func startBoot(bootAddr string) p2p.Boot {
	cfg := config.Boot{BootAddr: bootAddr}
	boot := p2p.NewBoot(cfg)
	boot.Start()
	return boot
}

/**

- boot[start,stop,list]
- node[start,stop,peers]
- miner[start,stop]


- account[list,create,balance,send,receive]
- ablock[list,head,reqs,detail]
- sblock[list,head,detail]
- pool[sprint,aprint]
- monitor[stat]
- profile[start]
*/
