package database

import (
	"sync"

	"github.com/dafsic/crypto-hunter/kraken_grid/model"
	"go.uber.org/zap"
)

type Database interface {
	// Connect establishes a connection to the database
	Connect() error
	// Close closes the connection to the database
	Close() error
	// CreateOrder creates a new order in the database
	CreateOrder(order *model.Order) error
	// GetOrder retrieves an order from the database by its OrderID
	GetOrder(uuid string) (*model.Order, error)
	// GetOrdersBybot retrieves all orders from the database for a specific bot
	GetOrdersByBot(bot string) ([]*model.Order, error)
	// GetOpenOrders retrieves all open orders from the database
	GetOpenOrders(bot string) ([]*model.Order, error)
	// UpdateOrder updates an existing order in the database
	UpdateOrder(order *model.Order) error
	// DeleteOrder deletes an order from the database by its ID
	DeleteOrder(uuid string) error
	// DeleteOrders deletes all orders for a specific bot
	DeleteOrdersByBot(bot string) error
}

type DBImpl struct {
	config *Config
	logger *zap.Logger
	orders map[string]*model.Order
	mux    *sync.Mutex
}

var _ Database = (*DBImpl)(nil)

func NewDB(logger *zap.Logger, config *Config) *DBImpl {
	return &DBImpl{
		config: config,
		logger: logger,
		orders: make(map[string]*model.Order),
		mux:    new(sync.Mutex),
	}
}

func (db *DBImpl) Connect() error {
	// Implement connection logic if needed
	return nil
}

func (db *DBImpl) Close() error {
	// Implement close logic if needed
	return nil
}
