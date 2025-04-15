package model

type Order struct {
	ID         int64   `json:"id"`
	UUID       string  `json:"uuid"`
	OrderID    string  `json:"order_id"`
	Bot        string  `json:"bot"`
	Exchange   string  `json:"exchange"`
	Pair       string  `json:"pair"`
	Price      float64 `json:"price"`
	Amount     float64 `json:"amount"`
	Status     string  `json:"order_status"`
	Side       string  `json:"side"`
	Multiplier int     `json:"multiplier"`
}
