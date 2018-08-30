package wallet

type Wallet interface {
	Accounts() []string
	CreateAccount(address string) string
}

func NewWallet() Wallet {
	w := &wallet{}
	return w
}

type wallet struct {
	accounts []string
}

func (self *wallet) Accounts() []string {
	return self.accounts
}
func (self *wallet) CreateAccount(address string) string {
	self.accounts = append(self.accounts, address)
	return address
}

func (self *wallet) Sign(address string, data []byte) []byte {
	return data
}
