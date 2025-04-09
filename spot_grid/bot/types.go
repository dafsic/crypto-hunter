package bot

type Order struct {
	ID      string  `json:"id"`
	Pair    string  `json:"pair"`
	Price   float64 `json:"price"`
	Amount  float64 `json:"amount"`
	Userref int     `json:"order_userref"`
	Status  string  `json:"order_status"`
	Side    string  `json:"side"`
}
