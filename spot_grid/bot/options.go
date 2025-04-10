package bot

import (
	"strconv"
	"strings"
)

//type BotName string

type Config struct {
	Name        string
	BaseCoin    string  // Base coin for the grid
	QuoteCoin   string  // Quote coin for the grid
	Step        float64 // Step size for the grid
	Amount      float64 // Amount of BaseCoin per grid order
	BasePrice   float64 // Base price for the grid
	Multipliers []int   // Multipliers for the grid orders
	// If the current price stays above the base price plus step size for the full interval duration,
	// the base price will be adjusted and orders will be rebalanced.
	Interval int64
}

type Option func(*Config)

func WithBotName(name string) Option {
	return func(c *Config) {
		c.Name = name
	}
}

func WithSetp(step float64) Option {
	return func(c *Config) {
		c.Step = step
	}
}

func WithGridAmount(amount float64) Option {
	return func(c *Config) {
		c.Amount = amount
	}
}

func WithBasePrice(price float64) Option {
	return func(c *Config) {
		c.BasePrice = price
	}
}

func WithBaseCoin(coin string) Option {
	return func(c *Config) {
		c.BaseCoin = coin
	}
}

func WithQuoteCoin(coin string) Option {
	return func(c *Config) {
		c.QuoteCoin = coin
	}
}

func WithMultipliers(multipliers string) Option {
	return func(c *Config) {
		multipliersList := strings.Split(multipliers, ",")
		for _, multiplier := range multipliersList {
			m, err := strconv.Atoi(multiplier)
			if err != nil {
				continue
			}
			c.Multipliers = append(c.Multipliers, m)
		}
	}
}

func WithInterval(interval int64) Option {
	return func(c *Config) {
		c.Interval = interval
	}
}

func NewConfig(opts ...Option) *Config {
	cfg := &Config{
		Name:        "unkonwn",
		Step:        0.0001,
		Amount:      1,
		BaseCoin:    "XMR",
		QuoteCoin:   "BTC",
		Multipliers: []int{1, 3},
		Interval:    60,
	}
	for _, opt := range opts {
		opt(cfg)
	}
	return cfg
}
