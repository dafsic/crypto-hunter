package websocket

import (
	"crypto/tls"
	"errors"
	"net/http"
	"net/url"
	"sync"

	"github.com/gorilla/websocket"
	"go.uber.org/zap"
)

type Socket struct {
	Conn              *websocket.Conn
	WebsocketDialer   *websocket.Dialer
	Url               string
	ConnectionOptions ConnectionOptions
	RequestHeader     http.Header
	OnConnected       func(socket *Socket)
	OnTextMessage     func(message string, socket *Socket)
	OnBinaryMessage   func(data []byte, socket *Socket)
	OnConnectError    func(err error, socket *Socket)
	OnDisconnected    func(err error, socket *Socket)
	OnPingReceived    func(data string, socket *Socket)
	OnPongReceived    func(data string, socket *Socket)
	sendMu            *sync.Mutex // Prevent "concurrent write to websocket connection"
	logger            *zap.Logger
}

type ConnectionOptions struct {
	UseCompression bool
	UseSSL         bool
	Proxy          func(*http.Request) (*url.URL, error)
	Subprotocols   []string
}

func New(url string, l *zap.Logger) *Socket {
	return &Socket{
		Url:           url,
		RequestHeader: http.Header{},
		ConnectionOptions: ConnectionOptions{
			UseCompression: false,
			UseSSL:         true,
		},
		WebsocketDialer: &websocket.Dialer{},
		sendMu:          &sync.Mutex{},
		logger:          l,
	}
}

func (socket *Socket) setConnectionOptions() {
	socket.WebsocketDialer.EnableCompression = socket.ConnectionOptions.UseCompression
	socket.WebsocketDialer.TLSClientConfig = &tls.Config{InsecureSkipVerify: socket.ConnectionOptions.UseSSL}
	socket.WebsocketDialer.Proxy = socket.ConnectionOptions.Proxy
	socket.WebsocketDialer.Subprotocols = socket.ConnectionOptions.Subprotocols
}

func (socket *Socket) Connect() {
	var err error
	var resp *http.Response
	socket.setConnectionOptions()

	socket.Conn, resp, err = socket.WebsocketDialer.Dial(socket.Url, socket.RequestHeader)

	if err != nil {
		socket.logger.Error("Error while connecting to server ", zap.Error(err))
		if resp != nil {
			socket.logger.Error("HTTP Response ", zap.Int("code", resp.StatusCode), zap.String("status", resp.Status))
		}
		if socket.OnConnectError != nil {
			socket.OnConnectError(err, socket)
		}
		return
	}

	socket.logger.Info("Connected to server", zap.String("url", socket.Url))

	if socket.OnConnected != nil {
		socket.OnConnected(socket)
	}

	defaultPingHandler := socket.Conn.PingHandler()
	socket.Conn.SetPingHandler(func(appData string) error {
		if socket.OnPingReceived != nil {
			socket.OnPingReceived(appData, socket)
		}
		return defaultPingHandler(appData)
	})

	defaultPongHandler := socket.Conn.PongHandler()
	socket.Conn.SetPongHandler(func(appData string) error {
		if socket.OnPongReceived != nil {
			socket.OnPongReceived(appData, socket)
		}
		return defaultPongHandler(appData)
	})

	defaultCloseHandler := socket.Conn.CloseHandler()
	socket.Conn.SetCloseHandler(func(code int, text string) error {
		result := defaultCloseHandler(code, text)
		if socket.OnDisconnected != nil {
			socket.OnDisconnected(errors.New(text), socket)
		}
		return result
	})

	go func() {
		for {
			messageType, message, err := socket.Conn.ReadMessage()
			if err != nil {
				socket.logger.Error("socket read error", zap.Error(err))
				if socket.OnDisconnected != nil {
					socket.OnDisconnected(err, socket)
				}
				return
			}
			//socket.logger.Info("socket recv", zap.ByteString("message", message))

			switch messageType {
			case websocket.TextMessage:
				if socket.OnTextMessage != nil {
					socket.OnTextMessage(string(message), socket)
				}
			case websocket.BinaryMessage:
				if socket.OnBinaryMessage != nil {
					socket.OnBinaryMessage(message, socket)
				}
			}
		}
	}()
}

func (socket *Socket) SendText(message string) error {
	return socket.send(websocket.TextMessage, []byte(message))

}

func (socket *Socket) SendBinary(data []byte) error {
	return socket.send(websocket.BinaryMessage, data)
}

func (socket *Socket) send(messageType int, data []byte) error {
	socket.sendMu.Lock()
	err := socket.Conn.WriteMessage(messageType, data)
	socket.sendMu.Unlock()
	return err
}

func (socket *Socket) Close() {
	err := socket.send(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseNormalClosure, ""))
	if err != nil {
		socket.logger.Error("socket write close error", zap.Error(err))
	}
	socket.Conn.Close()
}
