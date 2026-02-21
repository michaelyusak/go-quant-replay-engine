package handler

import (
	"context"
	"encoding/json"
	"errors"
	"michaelyusak/go-quant-replay-engine.git/common"
	"michaelyusak/go-quant-replay-engine.git/entity"
	"michaelyusak/go-quant-replay-engine.git/service"
	"sync"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
	hHelper "github.com/michaelyusak/go-helper/helper"
	"github.com/sirupsen/logrus"
)

type Replay struct {
	replayService service.Replay
	upgrader      websocket.Upgrader
}

func NewReplay(
	replayService service.Replay,
	upgrader websocket.Upgrader,
) *Replay {
	return &Replay{
		replayService: replayService,
		upgrader:      upgrader,
	}
}

func (h *Replay) CreateStream(ctx *gin.Context) {
	ctx.Header("Content-Type", "application/json")

	var req entity.CreateStreamReq

	err := ctx.ShouldBindJSON(&req)
	if err != nil {
		ctx.Error(err)
		return
	}

	c := ctx.Request.Context()

	res, err := h.replayService.CreateStream(c, req)
	if err != nil {
		ctx.Error(err)
		return
	}

	hHelper.ResponseOK(ctx, res)
}

func (h *Replay) StreamReplay(ctx *gin.Context) {
	conn, err := h.upgrader.Upgrade(ctx.Writer, ctx.Request, nil)
	if err != nil {
		ctx.Error(err)
		return
	}
	defer conn.Close()

	c, done := context.WithCancel(ctx.Request.Context())
	defer done()

	authDataCh := make(chan json.RawMessage)
	authenticatedCh := make(chan bool)
	dataCh := make(chan []byte)
	defer close(dataCh)

	var wg sync.WaitGroup

	wg.Add(1)
	go func() {
		defer wg.Done()

		raw := <-authDataCh
		close(authDataCh)

		var data entity.WsAuthData
		err := json.Unmarshal(raw, &data)
		if err != nil {
			logrus.
				WithError(err).
				Error("[handler][Replay][StreamReplay][Auth][json.Unmarshal(raw, &data)]")

			done()
			return
		}

		authenticatedCh <- true
		close(authenticatedCh)

		err = h.replayService.StreamReplay(c, dataCh, data.Channel)
		if err != nil {
			logrus.
				WithError(err).
				Error("[handler][Replay][StreamReplay][Auth][replayService.StreamReplay]")
		}

		done()
	}()

	wg.Add(1)
	go func() {
		defer wg.Done()

	loop:
		for {
			select {
			case <-c.Done():
				logrus.Warn("[handler][Replay][StreamReplay][Write] closing loop")
				break loop
			default:
				data := <-dataCh

				err := conn.WriteMessage(websocket.TextMessage, data)
				if err != nil {
					if errors.Is(err, websocket.ErrCloseSent) {
						break
					}

					logrus.
						WithError(err).
						Warn("[handler][Replay][StreamReplay][Write][conn.WriteMessage]")
					continue
				}
			}
		}
	}()

	wg.Add(1)
	go func() {
		defer wg.Done()

	loop:
		for {
			select {
			case <-c.Done():
				logrus.Warn("[handler][Replay][StreamReplay][Read] closing loop")
				break loop
			default:
				messageType, message, err := conn.ReadMessage()
				if err != nil {
					if websocket.IsUnexpectedCloseError(err) {
						done()
						break loop
					}

					logrus.
						WithError(err).
						Warn("[handler][Replay][StreamReplay][Read][conn.ReadMessage]")

					continue
				}

				switch messageType {
				case websocket.TextMessage:
					var msg entity.WsMessage
					err = json.Unmarshal(message, &msg)
					if err != nil {
						logrus.
							WithError(err).
							Warn("[handler][Replay][StreamReplay][Read][json.Unmarshal(message, &msg)]")
						continue
					}

					if msg.Type == string(entity.WsMessageTypeAuth) {
						authDataCh <- msg.Data
					}

					<-authenticatedCh

					// Feature
				}
			}
		}
	}()

	wg.Wait()

	err = common.CloseConn(conn)
	if err != nil {
		ctx.Error(err)
	}
}
