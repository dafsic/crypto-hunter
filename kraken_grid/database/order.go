package database

import (
	"errors"

	"github.com/dafsic/crypto-hunter/kraken_grid/model"
	"go.uber.org/zap"
)

var (
	ErrNotFound      = errors.New("not found")
	ErrAlreadyExists = errors.New("already exists")
)

func (db *DBImpl) CreateOrder(order *model.Order) error {
	if _, exists := db.orders[order.OrderID]; exists {
		return ErrAlreadyExists
	}
	db.logger.Info("Creating order",
		zap.String("uuid", order.UUID),
		zap.String("bot", order.Bot),
		zap.String("order_id", order.OrderID),
		zap.String("symbol", order.Pair),
		zap.String("side", order.Side),
		zap.Float64("price", order.Price),
		zap.String("status", order.Status),
	)
	db.mux.Lock()
	defer db.mux.Unlock()
	db.orders[order.UUID] = order
	return nil
}

func (db *DBImpl) GetOrder(uuid string) (*model.Order, error) {
	db.mux.Lock()
	defer db.mux.Unlock()
	order, ok := db.orders[uuid]
	if !ok {
		return nil, ErrNotFound
	}
	return order, nil
}

func (db *DBImpl) GetOrdersByBot(bot string) ([]*model.Order, error) {
	db.mux.Lock()
	defer db.mux.Unlock()
	var orders []*model.Order
	for _, order := range db.orders {
		if order.Bot == bot {
			orders = append(orders, order)
		}
	}
	if len(orders) == 0 {
		return nil, ErrNotFound
	}
	return orders, nil
}

func (db *DBImpl) GetOpenOrders(bot string) ([]*model.Order, error) {
	db.mux.Lock()
	defer db.mux.Unlock()

	var orders []*model.Order
	for _, order := range db.orders {
		if order.Bot == bot && order.Status == "new" {
			orders = append(orders, order)
		}
	}
	if len(orders) == 0 {
		return nil, ErrNotFound
	}
	return orders, nil
}

func (db *DBImpl) UpdateOrder(order *model.Order) error {
	db.mux.Lock()
	defer db.mux.Unlock()
	if _, exists := db.orders[order.UUID]; !exists {
		return ErrNotFound
	}
	db.orders[order.UUID] = order
	return nil
}

func (db *DBImpl) DeleteOrder(uuid string) error {
	db.mux.Lock()
	defer db.mux.Unlock()
	if _, exists := db.orders[uuid]; !exists {
		return ErrNotFound
	}
	delete(db.orders, uuid)
	return nil
}

func (db *DBImpl) DeleteOrdersByBot(bot string) error {
	db.mux.Lock()
	defer db.mux.Unlock()
	for uuid, order := range db.orders {
		if order.Bot == bot {
			delete(db.orders, uuid)
		}
	}
	return nil
}
