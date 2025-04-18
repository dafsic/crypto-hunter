package bot

import (
	"context"
	"encoding/json"
	"math"
	"sync/atomic"
	"unsafe"

	"github.com/dafsic/crypto-hunter/kraken"
	"github.com/dafsic/crypto-hunter/kraken_grid/dao"
	"github.com/dafsic/crypto-hunter/kraken_grid/model"
	"github.com/dafsic/crypto-hunter/utils"
	"github.com/dafsic/crypto-hunter/websocket"
	"go.uber.org/zap"
)

type Bot interface {
	Name() string
	Run() error
	Stop() error
	Done() <-chan struct{}
	Err() error
}

type Grid struct {
	status *atomic.Int32
	config *Config
	logger *zap.Logger
	// orders
	dao dao.Dao
	// websockets
	publicWS  *websocket.Socket
	privateWS *websocket.Socket
	// kraken
	krakenAPI kraken.Kraken
	token     string
	// context
	ctx    context.Context
	cancel context.CancelFunc
}

var _ Bot = (*Grid)(nil)

// NewBot creates a new bot
func NewBot(logger *zap.Logger, config *Config, krakenAPI kraken.Kraken, dao dao.Dao) *Grid {
	return &Grid{
		status:    new(atomic.Int32),
		config:    config,
		logger:    logger,
		krakenAPI: krakenAPI,
		dao:       dao,
	}
}

func (b *Grid) Name() string {
	return b.config.name
}

func (b *Grid) Done() <-chan struct{} {
	return b.ctx.Done()
}

func (b *Grid) Err() error {
	return b.ctx.Err()
}

func (b *Grid) Run() error {
	b.logger.Info("Starting bot...", zap.String("name", b.config.name))
	b.ctx, b.cancel = context.WithCancel(context.Background())
	utils.TurnOn(b.status)
	go b.mainloop()
	return nil
}

func (b *Grid) Stop() error {
	if utils.TurnOff(b.status) {
		b.logger.Info("Stopping bot...", zap.String("name", b.config.name))
		b.privateWS.Close()
		b.publicWS.Close()
	}
	return nil
}

func (b *Grid) mainloop() {
	b.logger.Info("Starting main loop...")

	// Get websocket token
	token, err := b.krakenAPI.GetWebsocketToken()
	if err != nil {
		b.logger.Error(err.Error())
		b.Stop()
		return
	}
	b.token = token.Token

	// Initialize websockets
	b.publicWS = b.newSocket(kraken.PublicWSURL)
	b.privateWS = b.newSocket(kraken.PrivateWSURL)

	// Subscribe to necessary channels
	if err := b.krakenAPI.SubscribeTickers(b.publicWS, b.config.baseCoin+"/"+b.config.quoteCoin); err != nil {
		b.logger.Error(err.Error())
		b.Stop()
		return
	}
	if err := b.krakenAPI.SubscribeExecutions(b.privateWS, b.token); err != nil {
		b.logger.Error(err.Error())
		b.Stop()
		return
	}
}

func (b *Grid) newSocket(url string) *websocket.Socket {
	socket := websocket.New(url, b.logger)
	socket.OnPingReceived = func(appData string, s *websocket.Socket) {
		b.logger.Info("WebSocket ping received", zap.String("url", s.Url), zap.String("data", appData))
	}
	socket.OnPongReceived = func(appData string, s *websocket.Socket) {
		b.logger.Info("WebSocket pong received", zap.String("url", s.Url), zap.String("data", appData))
	}
	socket.OnConnected = func(s *websocket.Socket) {
		b.logger.Info("WebSocket connected", zap.String("url", s.Url))
	}
	socket.OnConnectError = func(err error, s *websocket.Socket) {
		b.logger.Error("WebSocket connection error", zap.String("url", s.Url), zap.Error(err))
		b.Stop()
	}

	socket.OnDisconnected = func(err error, s *websocket.Socket) {
		s.Conn.Close()
		b.Stop()
	}

	socket.OnBinaryMessage = b.OnBinaryMessage
	socket.OnTextMessage = b.OnTextMessage

	socket.Connect()
	return socket
}

func (b *Grid) OnBinaryMessage(data []byte, socket *websocket.Socket) {
	b.logger.Info("WebSocket binary message received", zap.ByteString("message", data))
}

func (b *Grid) OnTextMessage(data string, socket *websocket.Socket) {
	// b.logger.Info("WebSocket text message received", zap.String("message", data), zap.String("url", socket.Url))
	var message any
	err := json.Unmarshal(utils.StringToBytes(data), &message)
	if err != nil {
		b.logger.Error("WebSocket binary message error", zap.Error(err))
		b.Stop()
		return
	}

	switch message := message.(type) {
	case map[string]any:
		b.handleMapMessage(message)
	case []any:
		for _, msg := range message {
			if msgMap, ok := msg.(map[string]any); ok {
				b.handleMapMessage(msgMap)
			} else {
				b.logger.Error("WebSocket text message error", zap.String("error", "message is not a map"))
				b.Stop()
				return
			}
		}
	default:
		b.logger.Error("WebSocket text message error", zap.String("error", "message is not a map or slice"))
		b.Stop()
		return
	}
}

func (b *Grid) handleMapMessage(message map[string]any) {
	if message["method"] != nil {
		switch message["method"] {
		case "add_order", "cannel_order", "subscribe":
			b.handleMethodResponse(message)
		default:
			b.logger.Info("WebSocket message ignored", zap.Any("method", message["method"]))
		}
		return
	}

	if message["channel"] != nil {
		switch message["channel"] {
		case "status", "heartbeat":
			// b.logger.Info("WebSocket admin message", zap.Any("channel", message["channel"]))
		case "ticker":
			b.handleTickerChannel(message)
		case "executions":
			b.handleExecutionsChannel(message)
		default:
			b.logger.Info("WebSocket message ignored", zap.Any("channel", message["channel"]))
		}
	}
}

func (b *Grid) handleMethodResponse(message map[string]any) {
	b.logger.Info("WebSocket method response", zap.Any("method", message["method"]), zap.Any("result", message["result"]), zap.Bool("success", message["success"].(bool)))
	if success, ok := message["success"].(bool); !ok || !success {
		b.logger.Error("WebSocket method response error", zap.Any("method", message["method"]), zap.Any("error", message["error"]))
		b.Stop()
	}
}

func (b *Grid) handleTickerChannel(message map[string]any) {
	data, ok := message["data"].([]any)
	if !ok || len(data) == 0 {
		return
	}

	tickerData, ok := data[0].(map[string]any)
	if !ok {
		return
	}

	if price, ok := tickerData["last"].(float64); ok {
		b.config.currentPrice = price
	}
	b.logger.Info("WebSocket ticker message",
		zap.Float64("current price", b.config.currentPrice),
		zap.Float64("center price", b.getCenterPrice()),
	)

	centerPrice := b.getCenterPrice()
	if math.Abs(b.config.currentPrice-centerPrice) > float64(b.config.multipliers[len(b.config.multipliers)-2])*b.config.step {
		b.logger.Info("Price exceeded threshold",
			zap.Float64("current price", b.config.currentPrice),
			zap.Float64("center price", centerPrice),
		)
		b.config.timer.Start()
		if b.config.timer.IsExpired() {
			b.config.timer.Reset()
			b.setCenterPrice(b.config.currentPrice)
			b.rebaseOrders()
		}
	} else {
		b.config.timer.Reset()
	}

}

func (b *Grid) handleExecutionsChannel(message map[string]any) {
	data, ok := message["data"].([]any)
	if !ok {
		return
	}

	for _, execution := range data {
		exec := execution.(map[string]any) // assume execution is a map, panic if not
		orderID := exec["order_id"].(string)
		userref := exec["order_userref"].(float64)
		order, err := b.dao.GetOrder(b.ctx, int(userref))
		if err != nil {
			b.logger.Error("Failed to get order from database", zap.String("order_id", orderID), zap.Error(err))
			b.Stop()
			return
		}

		order.Status = exec["exec_type"].(string)
		err = b.dao.UpdateOrder(b.ctx, order.ID, map[string]any{"status": order.Status}) // Update order in database
		if err != nil {
			b.logger.Error("Failed to update order in database", zap.String("order_id", orderID), zap.Error(err))
			b.Stop()
			return
		}
		b.logger.Info("Order update",
			zap.String("order_id", order.OrderID),
			zap.String("status", order.Status),
			zap.String("side", order.Side),
			zap.Float64("price", order.Price),
			zap.String("pair", order.Pair),
			zap.Int("multiplier", order.Multiplier),
		)

		switch order.Status {
		case "filled":
			b.handleOrderFilled(order)
		default: // "new", "cancelled", "pending"
		}
	}
}

func (b *Grid) rebaseOrders() {
	// cancel all orders
	orders, err := b.dao.GetOpenOrders(b.ctx, b.config.name)
	if err != nil && err != dao.ErrNotFound {
		b.logger.Error("Failed to get open orders from database", zap.Error(err))
		b.Stop()
		return
	}

	orderIDs := make([]string, len(orders))
	for i, order := range orders {
		orderIDs[i] = order.OrderID
	}

	if len(orders) > 0 {
		err := b.krakenAPI.CancelOrderWithWebsocket(b.privateWS, b.token, orderIDs)
		if err != nil {
			b.logger.Error("Failed to cancel orders", zap.Error(err))
			b.Stop()
			return
		}
	}

	// place new orders
	for _, v := range b.config.multipliers {
		b.addOrder(kraken.Buy, b.config.centerPrice, v)
		b.addOrder(kraken.Sell, b.config.centerPrice, v)
	}
}

func (b *Grid) addOrder(side kraken.Side, basePrice float64, multiplier int) {
	price := basePrice - (b.config.step * float64(multiplier))
	if side == kraken.Sell {
		price = basePrice + (b.config.step * float64(multiplier))
	}

	order := b.newOrder()
	order.Pair = b.config.baseCoin + "/" + b.config.quoteCoin
	order.Price = utils.FormatFloat(price, 6)
	order.Amount = b.config.amount
	order.Side = side.String()
	order.Multiplier = multiplier
	order.Status = "pending"

	// Save order to database
	err := b.dao.CreateOrder(b.ctx, order)
	if err != nil {
		b.logger.Error("Failed to create order in database", zap.Error(err))
		b.Stop()
		return
	}

	err = b.krakenAPI.AddOrderWithWebsocket(
		b.privateWS,
		order.Pair,
		b.token,
		side,
		order.Amount,
		order.Price,
		order.ID,
	)
	if err != nil {
		b.logger.Error("Failed to place new order", zap.Error(err))
		b.Stop()
	}
}

func (b *Grid) handleOrderFilled(order *model.Order) {
	b.addOrder(kraken.NewSide(order.Side).Opposite(), order.Price, order.Multiplier)
}

func (b *Grid) newOrder() *model.Order {
	return &model.Order{
		Bot:      b.config.name,
		Exchange: "kraken",
	}
}

func (b *Grid) getCenterPrice() float64 {
	return math.Float64frombits(atomic.LoadUint64((*uint64)(unsafe.Pointer(&b.config.centerPrice))))
}

func (b *Grid) setCenterPrice(new float64) {
	atomic.StoreUint64((*uint64)(unsafe.Pointer(&b.config.centerPrice)), math.Float64bits(new))
}
