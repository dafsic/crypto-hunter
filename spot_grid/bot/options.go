package bot

//type BotName string

type Config struct {
	// Name is the name of the bot
	Name string
	// Grid parameters
	BaseCoin   string  // Base coin for the grid
	QuoteCoin  string  // Quote coin for the grid
	Step       float64 // Step size for the grid
	GridAmount float64 // Amount of XMR per grid order
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
		c.GridAmount = amount
	}
}

func NewConfig(opts ...Option) *Config {
	cfg := &Config{
		Name:       "unkonwn",
		Step:       0.0001,
		GridAmount: 2,
		BaseCoin:   "XMR",
		QuoteCoin:  "BTC",
	}
	for _, opt := range opts {
		opt(cfg)
	}
	return cfg
}
