package ws

import (
	"github.com/go-logr/logr"
	"github.com/gorilla/websocket"
	"github.com/kalmhq/kalm/api/client"
	"github.com/kalmhq/kalm/api/log"
	"github.com/labstack/echo/v4"
	"net/http"
)

var (
	upgrader = websocket.Upgrader{
		CheckOrigin: func(r *http.Request) bool {
			return true
		},
	}
)

type WsHandler struct {
	clientManager client.ClientManager
	clientPool    *ClientPool
	logger        logr.Logger
}

func NewWsHandler(clientManager client.ClientManager) *WsHandler {
	clientPool := NewClientPool()
	go clientPool.run()

	return &WsHandler{
		clientManager: clientManager,
		clientPool:    clientPool,
		logger:        log.DefaultLogger(),
	}
}

func (h *WsHandler) Serve(c echo.Context) error {
	clt := &Client{
		clientPool:    h.clientPool,
		send:          make(chan []byte, 256),
		done:          make(chan struct{}),
		stopWatcher:   make(chan struct{}),
		clientManager: h.clientManager,
		logger:        h.logger,
	}

	conn, err := upgrader.Upgrade(c.Response(), c.Request(), nil)

	if err != nil {
		log.Error(err, "ws upgrade error")
		return err
	}

	clt.conn = conn

	clientInfo, err := h.clientManager.GetClientInfoFromContext(c)

	if err == nil && clientInfo != nil {
		clt.clientInfo = clientInfo
	}

	clt.clientPool.register <- clt

	go clt.write()
	clt.read()

	return nil
}
