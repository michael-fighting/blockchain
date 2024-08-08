package app

type Transaction struct {
	TxID      string `json:"id"`
	From      string `json:"from"`
	To        string `json:"to"`
	Amount    int    `json:"amount"`
	Signature string `json:"signature"`
}
