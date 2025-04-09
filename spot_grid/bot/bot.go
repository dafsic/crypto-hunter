package bot

import (
	"encoding/json"
	"math"
	"sync"
	"sync/atomic"
	"time"
	"unsafe"

	"github.com/dafsic/crypto-hunter/kraken"
	"github.com/dafsic/crypto-hunter/utils"
	"github.com/dafsic/crypto-hunter/websocket"
	"go.uber.org/zap"
)

type Boter interface {
	// Start starts the bot
	Start() error
	// Stop stops the bot
	Stop() error
}

type bot struct {
	switcher utils.Switch
	config   *Config
	logger   *zap.Logger
	// account info
	buyOrders  map[string]*Order
	sellOrders map[string]*Order
	orderMux   *sync.Mutex
	// websocket
	publicWS  *websocket.Socket
	privateWS *websocket.Socket
	// kraken
	krakenAPI kraken.Kraken
	token     string
	// strategy
	basePrice    float64
	currentPrice float64
	ticker       time.Time
}

var _ Boter = (*bot)(nil)

// NewBot creates a new bot
func NewBot(logger *zap.Logger, krakenAPI kraken.Kraken, config *Config) *bot {
	return &bot{
		config:     config,
		logger:     logger,
		krakenAPI:  krakenAPI,
		buyOrders:  make(map[string]*Order),
		sellOrders: make(map[string]*Order),
		orderMux:   new(sync.Mutex),
		ticker:     time.Now(),
	}
}

// Start starts the bot
func (b *bot) Start() error {
	b.logger.Info("Starting bot...", zap.String("name", b.config.Name))
	b.switcher.On()
	go b.mainloop()
	return nil
}

// Stop stops the bot
func (b *bot) Stop() error {
	if b.switcher.State() == utils.On {
		b.switcher.Off()
		b.logger.Info("Stopping bot...", zap.String("name", b.config.Name))
		b.privateWS.Close()
		b.publicWS.Close()
	}
	return nil
}

func (b *bot) mainloop() {
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
	if err := b.krakenAPI.SubscribeTickers(b.publicWS, b.config.BaseCoin+"/"+b.config.QuoteCoin); err != nil {
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

func (b *bot) newSocket(url string) *websocket.Socket {
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
		b.logger.Error("WebSocket disconnected", zap.String("url", s.Url), zap.Error(err))
		b.Stop()
	}

	socket.OnBinaryMessage = b.OnBinaryMessage
	socket.OnTextMessage = b.OnTextMessage

	socket.Connect()
	return socket
}

func (b *bot) OnBinaryMessage(data []byte, socket *websocket.Socket) {
	b.logger.Info("WebSocket binary message received", zap.ByteString("message", data))
}

func (b *bot) OnTextMessage(data string, socket *websocket.Socket) {
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

func (b *bot) handleMapMessage(message map[string]any) {
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

func (b *bot) handleMethodResponse(message map[string]any) {
	if success, ok := message["success"].(bool); !ok || !success {
		b.logger.Error("WebSocket method response error", zap.Any("method", message["method"]), zap.Any("error", message["error"]))
		b.Stop()
	}
}

func (b *bot) handleTickerChannel(message map[string]any) {
	data, ok := message["data"].([]any)
	if !ok || len(data) == 0 {
		return
	}

	tickerData, ok := data[0].(map[string]any)
	if !ok {
		return
	}

	if price, ok := tickerData["last"].(float64); ok {
		b.currentPrice = price
		b.logger.Info("Price updated", zap.Float64("price", b.currentPrice))
	}

	if math.Abs(b.currentPrice-b.getBasePrice()) > b.config.Step {
		b.logger.Info("Price exceeded step", zap.Float64("current_price", b.currentPrice), zap.Float64("base_price", b.basePrice))
		now := time.Now()
		if now.Sub(b.ticker) > time.Minute {
			b.ticker = now
			b.setBasePrice(b.currentPrice)
			b.rebaseOrders()
		}
	}

}

func (b *bot) handleExecutionsChannel(message map[string]any) {
	data, ok := message["data"].([]any)
	if !ok {
		return
	}

	b.orderMux.Lock()
	defer b.orderMux.Unlock()
	for _, execution := range data {
		exec, ok := execution.(map[string]any)
		if !ok {
			continue
		}

		orderID := exec["order_id"].(string)
		status := exec["exec_type"].(string)
		userref := exec["order_userref"].(float64)
		symbol, _ := exec["symbol"].(string)
		side, _ := exec["side"].(string)
		price, _ := exec["limit_price"].(float64)
		order := &Order{
			ID:      orderID,
			Pair:    symbol,
			Price:   price,
			Amount:  b.config.GridAmount,
			Userref: int(userref),
			Status:  status,
			Side:    side,
		}
		b.logger.Info("New order execution",
			zap.String("order_id", order.ID),
			zap.String("symbol", order.Pair),
			zap.String("side", order.Side),
			zap.Float64("price", order.Price),
			zap.String("status", order.Status),
			zap.Int("userref", order.Userref),
		)
		switch status {
		case "new":

			if side == string(kraken.Buy) {
				b.buyOrders[orderID] = order
			} else {
				b.sellOrders[orderID] = order
			}
		case "filled":
			b.handleOrderFilled(orderID, kraken.Side(side))
		case "canceled":
			b.handleOrderCanceled(orderID, kraken.Side(side))
		}
	}
}

func (b *bot) rebaseOrders() {
	b.orderMux.Lock()
	defer b.orderMux.Unlock()

	// cancel all orders
	var orderIDs []string
	for orderID := range b.buyOrders {
		orderIDs = append(orderIDs, orderID)
	}
	for orderID := range b.sellOrders {
		orderIDs = append(orderIDs, orderID)
	}

	if len(orderIDs) > 0 {
		err := b.krakenAPI.CancelOrderWithWebsocket(b.privateWS, b.token, orderIDs)
		if err != nil {
			b.logger.Error("Failed to cancel orders", zap.Error(err))
			b.Stop()
		}
	}

	// place new orders
	b.addOrder(kraken.Buy, 1)
	b.addOrder(kraken.Buy, 3)
	b.addOrder(kraken.Sell, 1)
	b.addOrder(kraken.Sell, 3)
}

func (b *bot) addOrder(side kraken.Side, multiplier int) {
	basePrice := b.getBasePrice()
	price := basePrice - (b.config.Step * float64(multiplier))
	if side == kraken.Sell {
		price = basePrice + (b.config.Step * float64(multiplier))
	}
	price = utils.FormatFloat(price, 6)
	err := b.krakenAPI.AddOrderWithWebsocket(
		b.privateWS,
		b.config.BaseCoin+"/"+b.config.QuoteCoin,
		b.token,
		side,
		b.config.GridAmount,
		price,
		multiplier,
	)
	if err != nil {
		b.logger.Error("Failed to place new order", zap.Error(err))
		b.Stop()
	}
}

func (b *bot) handleOrderFilled(orderID string, side kraken.Side) {
	var multiplier int
	if side == kraken.Buy {
		multiplier = b.buyOrders[orderID].Userref
	} else {
		multiplier = b.sellOrders[orderID].Userref
	}

	b.removeOrder(orderID, side)

	b.addOrder(side.Opposite(), multiplier)
}

func (b *bot) handleOrderCanceled(orderID string, side kraken.Side) {
	b.removeOrder(orderID, side)
}

func (b *bot) removeOrder(orderID string, side kraken.Side) {
	if side == kraken.Buy {
		delete(b.buyOrders, orderID)
	} else {
		delete(b.sellOrders, orderID)
	}
}

func (b *bot) getBasePrice() float64 {
	return math.Float64frombits(atomic.LoadUint64((*uint64)(unsafe.Pointer(&b.basePrice))))
}

func (b *bot) setBasePrice(new float64) {
	atomic.StoreUint64((*uint64)(unsafe.Pointer(&b.basePrice)), math.Float64bits(new))
}
