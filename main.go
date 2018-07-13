package main

import (
	"bufio"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"io"
	"log"
	"net"
	"strconv"
	"sync"
	"time"
)

type Tx struct {
	TxType       int // 1: send  2:received
	AccountNonce int
	Amount       int
	From         string
	To           string
	Hash         string
	Source       string
	Signer       string
}

type AccountStateBlock struct {
	Nonce          int
	Timestamp      string
	Amount         int // the balance
	Hash           string
	PreHash        string
	Signer         string
	SnapshotHeight int
	SnapshotHash   string
	Tx             *Tx
}

type SnapshotBlock struct {
	Height       int
	Timestamp    string
	PreHash      string
	Signer       string
	AccountsHash string
	Hash         string
}

type Node struct {
	address        string
	receivedTxChan chan Tx
}

//k:address v:block
var accountStateBlockChain = make(map[string]AccountStateBlock)
var snapshotBlockChain SnapshotBlock

var chainmutex = &sync.Mutex{}

// var pendingTxs = make(map[string]Tx)
var pendingAccountStatusBlocks = make(map[string][]AccountStateBlock)

//var pendingSnapshotBlocks
var broadcastAccountBlock = make(chan AccountStateBlock)

var broadcastSnapshotBlock = make(chan SnapshotBlock)

// k:hash v:Tx
var txDB = make(map[string]Tx)

// k:hash v:block
var stateBlockDB = make(map[string]AccountStateBlock)

var nodes = make(map[string]Node)

// k1:hash   k2:node   v:accountHash
var snapshotAccountMap = make(map[string]map[string]string)

// k:hash  v:snapshotBlock
var snapshotDB = make(map[string]SnapshotBlock)

// whether it has been snapshotted
var stateSnapshotMap = make(map[string]bool)

func main() {
	httpPort := strconv.Itoa(9000)

	// start TCP and serve TCP server
	server, err := net.Listen("tcp", ":"+httpPort)
	if err != nil {
		log.Fatal(err)
	}
	log.Println("Tcp Server Listening on port :", httpPort)
	defer server.Close()

	go func() {
		for {
			snapshotBlock := <-broadcastSnapshotBlock
			if !validateSnapshotBlock(&snapshotBlock) {
				log.Printf("snapshot block invalid!")
			}
		}
	}()

	initSnapshotChain()

	go func() {
		for {
			accountBlock := <-broadcastAccountBlock
			if !validateAccountStateBlock(&accountBlock) {
				log.Printf("account block invalid!")
				continue
			}
			for nodeName, node := range nodes {
				if accountBlock.Tx != nil && accountBlock.Tx.TxType == 1 && accountBlock.Tx.To == nodeName {
					tx := accountBlock.Tx
					node.receivedTxChan <- *tx
				}
			}
		}

	}()

	go func() {
		for {
			time.Sleep(10 * time.Second)
			output := printAccountBlockChain(accountStateBlockChain)
			log.Printf("%v", output)
			snapshot := printSnapshotBlockChain(snapshotBlockChain)
			log.Printf("%v", snapshot)
		}
	}()

	for {
		conn, err := server.Accept()
		if err != nil {
			log.Fatal(err)
		}
		go handleConn(conn)
	}
}
func initSnapshotChain() {
	chainmutex.Lock()
	defer chainmutex.Unlock()

	block, accounts := generateGenesisSnapshotBlock()
	block = signSnapshotBlock("God", block)
	appendSnapshotBlockChain(block, accounts)
}

func generateSendTx(from string, to string, amount int) Tx {
	fromBlock := myAccountBlockChain(from)
	sendTx := Tx{1, fromBlock.Nonce + 1, amount, from, to, "", "", ""}
	return sendTx
}
func generateReceivedTx(send Tx) Tx {
	toBlock := myAccountBlockChain(send.To)
	sendTx := Tx{2, toBlock.Nonce + 1, send.Amount, send.From, send.To, "", send.Hash, ""}
	return sendTx
}

func signTx(tx Tx, from string) Tx {
	tx.Signer = from
	tx.Hash = calculateTxHash(tx)
	return tx
}

func generateFromAccountStateBlock(oldBlock AccountStateBlock, tx Tx, from string, snapshotHeight int, snapshotHash string) AccountStateBlock {
	var newBlock AccountStateBlock

	t := time.Now()

	newBlock.Tx = &tx
	newBlock.Nonce = oldBlock.Nonce + 1
	newBlock.Timestamp = t.String()
	newBlock.Amount = oldBlock.Amount - tx.Amount
	newBlock.PreHash = oldBlock.Hash
	newBlock.SnapshotHeight = snapshotHeight
	newBlock.SnapshotHash = snapshotHash

	return newBlock
}

// generateBlock creates a new block using previous block's hash
func generateToAccountStateBlock(oldBlock AccountStateBlock, tx Tx, to string, snapshotHeight int, snapshotHash string) AccountStateBlock {
	var newBlock AccountStateBlock

	t := time.Now()

	newBlock.Tx = &tx
	newBlock.Nonce = oldBlock.Nonce + 1
	newBlock.Timestamp = t.String()
	newBlock.Amount = oldBlock.Amount + tx.Amount
	newBlock.PreHash = oldBlock.Hash
	newBlock.SnapshotHeight = snapshotHeight
	newBlock.SnapshotHash = snapshotHash

	return newBlock
}

func generateGenesisAccountStateBlock(initBalance int, address string, snapshotHeight int, snapshotHash string) AccountStateBlock {
	t := time.Now()
	genesisBlock := AccountStateBlock{0, t.String(), initBalance, "", "", address, snapshotHeight, snapshotHash, nil}
	return genesisBlock
}

func generateGenesisSnapshotBlock() (SnapshotBlock, map[string]string) {
	var accounts = make(map[string]string)
	accountsByt, _ := json.Marshal(accounts)

	genesisBlock := SnapshotBlock{0, "today", "", "", string(accountsByt), ""}
	return genesisBlock, accounts
}

func appendAccountStateBlockChain(node string, block AccountStateBlock) bool {
	chainmutex.Lock()
	defer chainmutex.Unlock()

	if !validateAccountStateBlock(&block) {
		return false
	}
	current := accountStateBlockChain[node]
	if current.Hash == block.PreHash {
		accountStateBlockChain[node] = block
		stateBlockDB[block.Hash] = block
		txDB[block.Tx.Hash] = *block.Tx
		return true
	} else {
		return false
	}
}

func signAccountStateBlock(block AccountStateBlock, address string) AccountStateBlock {
	block.Signer = address
	block.Hash = calculateAccountBlockHash(block)
	return block
}

func generateSnapshotBlock(current SnapshotBlock) (SnapshotBlock, map[string]string) {
	currentAccountChain := accountStateBlockChain

	var accounts = make(map[string]string)

	for address, account := range currentAccountChain {
		exists := stateSnapshotMap[account.Hash]
		if !exists {
			accounts[address] = account.Hash
		}
	}

	accountHashByt, _ := json.Marshal(accounts)

	accountHash := string(accountHashByt)

	now := time.Now()
	block := SnapshotBlock{current.Height + 1, now.String(), current.Hash, "", accountHash, ""}
	return block, accounts
}

func appendSnapshotBlockChain(block SnapshotBlock, accounts map[string]string) bool {
	if !validateSnapshotBlock(&block) {
		return false
	}

	if block.Height != 0 && snapshotBlockChain.Hash != block.PreHash {
		return false
	}
	snapshotBlockChain = block
	snapshotDB[block.Hash] = block
	snapshotAccountMap[block.AccountsHash] = accounts
	for _, accountHash := range accounts {
		stateSnapshotMap[accountHash] = true
	}
	broadcastSnapshotBlock <- block
	return true
}

func signSnapshotBlock(signer string, block SnapshotBlock) SnapshotBlock {
	block.Signer = signer
	block.Hash = calculateSnapshotHash(block)
	return block
}

func validateTx(tx *Tx) bool {
	hash := calculateTxHash(*tx)
	return tx.Hash == hash
}

func validateAccountStateBlock(block *AccountStateBlock) bool {
	hash := calculateAccountBlockHash(*block)

	return block.Hash == hash && validateTx(block.Tx)
}

func validateSnapshotBlock(block *SnapshotBlock) bool {
	hash := calculateSnapshotHash(*block)
	return block.Hash == hash
}

func handleConn(conn net.Conn) {
	defer conn.Close()

	var closedChan = make(chan bool)

	// validator address
	var address string

	// allow user to allocate number of tokens to stake
	// the greater the number of tokens, the greater chance to forging a new block
	io.WriteString(conn, "Enter node address:")

	scanAddress := bufio.NewScanner(conn)
	if scanAddress.Scan() {
		address = scanAddress.Text()
	}

	node := initNode(address)
	defer destoryNode(node)

	io.WriteString(conn, address+", Enter role(1:tx node, 2:snapshot node):")

	scanRole := bufio.NewScanner(conn)
	if scanRole.Scan() {
		role, err := strconv.Atoi(scanRole.Text())
		if err != nil {
			log.Printf("%v not a number: %v", scanRole.Text(), err)
			return
		}

		if role == 1 {
			// account node
			go receiveTx(node)
			sendTx(conn, address)
		} else if role == 2 {
			// dpos node
			snapshot(conn, node)
		} else if role == 3 {

		}
	}

	closedChan <- true

}
func snapshot(conn net.Conn, node Node) {
	for {
		snapshotChain(node.address)
		time.Sleep(10 * time.Second)
	}
}
func snapshotChain(address string) {
	chainmutex.Lock()
	defer chainmutex.Unlock()

	current := snapshotBlockChain
	block, accounts := generateSnapshotBlock(current)
	block = signSnapshotBlock(address, block)
	appendSnapshotBlockChain(block, accounts)
}
func receiveTx(node Node) {
	for {
		sendTx := <-node.receivedTxChan

		if lookupOtherReceiveTx(node, sendTx) {
			continue
		}
		receiveTx := generateReceivedTx(sendTx)
		receiveTx = signTx(receiveTx, node.address)

		chain := myAccountBlockChain(node.address)
		var block = generateToAccountStateBlock(chain, receiveTx, node.address, snapshotBlockChain.Height, snapshotBlockChain.Hash)
		block = signAccountStateBlock(block, node.address)

		if appendAccountStateBlockChain(node.address, block) {
			log.Printf("submit received Tx success [" + node.address + "].\n")
			broadcastAccountBlock <- block
		} else {
			log.Printf("submit received Tx failed  [" + node.address + "].\n")
		}
	}

}

// if repeat : true
func lookupOtherReceiveTx(node Node, sendTx Tx) bool {
	chain := myAccountBlockChain(node.address)
	sourceHash := sendTx.Hash
	var block = chain
	for {
		if block.Tx != nil && block.Tx.TxType == 2 {
			if sourceHash == block.Tx.Source {
				return true
			}
		}
		if block.PreHash == "" {
			break
		}
		block = getAccountState(block.PreHash)
	}
	return false
}
func destoryNode(node Node) {
	chainmutex.Lock()
	defer chainmutex.Unlock()

	delete(nodes, node.address)
	close(node.receivedTxChan)
	delete(accountStateBlockChain, node.address)
}
func initNode(node string) Node {
	chainmutex.Lock()
	defer chainmutex.Unlock()

	n := Node{address: node, receivedTxChan: make(chan Tx)}
	nodes[node] = n
	snapshotHeight := snapshotBlockChain.Height
	snapshotHash := snapshotBlockChain.Hash
	block := generateGenesisAccountStateBlock(100, node, snapshotHeight, snapshotHash)
	block = signAccountStateBlock(block, node)
	accountStateBlockChain[node] = block
	stateBlockDB[block.Hash] = block
	return n
}

func myAccountBlockChain(node string) AccountStateBlock {
	return accountStateBlockChain[node]
}
func existsAccountBlockChain(node string) bool {
	_, exists := accountStateBlockChain[node]
	return exists
}

// print sth
func printAccountBlockChain(blocks map[string]AccountStateBlock) string {
	var result string
	for k, v := range blocks {
		var tmp = v
		for {
			result = strconv.Itoa(tmp.Amount) + result
			if tmp.PreHash == "" {
				break
			}
			result = "->" + result
			tmp = getAccountState(tmp.PreHash)
		}
		result = k + ":" + result
		result = "\n" + result
	}
	return result
}

// print sth
func printSnapshotBlockChain(block SnapshotBlock) string {
	var result string
	var tmp = block
	for {
		result = printSnapshotBlock(tmp) + result
		if tmp.PreHash == "" {
			break
		}
		tmp = getSnapshotBlock(tmp.PreHash)
	}
	return result
}

// print sth
func printSnapshotBlock(block SnapshotBlock) string {
	var result string
	hash := block.AccountsHash
	accounts := snapshotAccountMap[hash]
	for k, v := range accounts {
		state := getAccountState(v)
		result = k + "->" + strconv.Itoa(state.Amount) + "," + result
	}
	result = "\n" + strconv.Itoa(block.Height) + ":" + result
	return result
}
func getAccountState(hash string) AccountStateBlock {
	return stateBlockDB[hash]
}
func getSnapshotBlock(hash string) SnapshotBlock {
	return snapshotDB[hash]
}

func sendTx(conn net.Conn, address string) {
	for {
		chain := myAccountBlockChain(address)
		currentBalance := chain.Amount
		io.WriteString(conn, "current balance is :"+strconv.Itoa(currentBalance)+"\n")
		io.WriteString(conn, "Enter to address:")
		scanTx := bufio.NewScanner(conn)
		var toAddress string
		if scanTx.Scan() {
			toAddress = scanTx.Text()
			exists := existsAccountBlockChain(address)
			if !exists {
				io.WriteString(conn, "address:"+toAddress+" not exists")
				continue
			}
		}
		io.WriteString(conn, address+", Enter to amount:")

		if scanTx.Scan() {
			toAmount, err := strconv.Atoi(scanTx.Text())
			if err != nil {
				log.Printf("%v not a number: %v", scanTx.Text(), err)
				continue
			}
			submitTx(address, toAddress, toAmount)
		}
	}

}

func submitTx(from string, to string, amount int) {
	chain := myAccountBlockChain(from)
	tx := generateSendTx(from, to, amount)
	tx = signTx(tx, from)
	var block = generateFromAccountStateBlock(chain, tx, from, snapshotBlockChain.Height, snapshotBlockChain.Hash)
	block = signAccountStateBlock(block, from)

	if appendAccountStateBlockChain(from, block) {
		log.Printf("submit send Tx success[" + from + "].\n")
		broadcastAccountBlock <- block
	} else {
		log.Printf("submit send Tx failed[" + from + "].\n")
	}
}

// SHA256 hasing
// calculateHash is a simple SHA256 hashing function
func calculateHash(s string) string {
	h := sha256.New()
	h.Write([]byte(s))
	hashed := h.Sum(nil)
	return hex.EncodeToString(hashed)
}

func calculateTxHash(tx Tx) string {
	record := string(tx.Amount) + string(tx.AccountNonce) + string(tx.From) + string(tx.To) + string(tx.Source) + string(tx.Signer)
	return calculateHash(record)
}

func calculateAccountBlockHash(block AccountStateBlock) string {
	if block.Tx == nil {
		record := string(block.Nonce) + block.Timestamp + string(block.Amount) + block.PreHash + block.Signer + block.SnapshotHash + string(block.SnapshotHeight)
		return calculateHash(record)
	} else {
		record := string(block.Nonce) + block.Timestamp + string(block.Amount) + block.PreHash + block.Signer + block.SnapshotHash + string(block.SnapshotHeight) + block.Tx.Hash
		return calculateHash(record)
	}
}

func calculateSnapshotHash(block SnapshotBlock) string {
	record := string(block.Timestamp) + string(block.Signer) + string(block.PreHash) + string(block.AccountsHash)
	return calculateHash(record)
}
