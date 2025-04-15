package database

type Config struct {
	// Driver is the database driver
	// 	"mysql", "postgres", "sqlite3"
	// 	"postgres" is the default driver
	Driver string
	// 	"host=localhost port=5432 user=postgres password=postgres dbname=postgres sslmode=disable" is the default DSN
	DSN string // Data Source Name
}

type Option func(*Config)

func WithDriver(driver string) Option {
	return func(c *Config) {
		c.Driver = driver
	}
}

func WithDSN(dsn string) Option {
	return func(c *Config) {
		c.DSN = dsn
	}
}

func NewConfig(opts ...Option) *Config {
	cfg := new(Config)
	for _, opt := range opts {
		opt(cfg)
	}
	return cfg
}
