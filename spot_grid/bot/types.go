package bot

import "time"

type Order struct {
	ID      string  `json:"id"`
	Pair    string  `json:"pair"`
	Price   float64 `json:"price"`
	Amount  float64 `json:"amount"`
	Userref int     `json:"order_userref"`
	Status  string  `json:"order_status"`
	Side    string  `json:"side"`
}

type Timer struct {
	interval time.Duration
	from     time.Time
	flag     bool
}

func NewTimer(interval int64) *Timer {
	return &Timer{
		interval: time.Duration(interval) * time.Second,
	}
}
func (t *Timer) Start() {
	if t.flag {
		return
	}
	t.flag = true
	t.from = time.Now()
}

func (t *Timer) Reset() {
	t.flag = false
}

func (t *Timer) isExpired() bool {
	if !t.flag {
		return false
	}
	return time.Since(t.from) > t.interval
}
