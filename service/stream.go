package service

import (
	"context"
	"encoding/json"
	"fmt"
	"michaelyusak/go-quant-replay-engine.git/common"
	"michaelyusak/go-quant-replay-engine.git/entity"
	"michaelyusak/go-quant-replay-engine.git/repository"
	"net/http"
	"sync"
	"time"

	"github.com/michaelyusak/go-helper/apperror"
	"github.com/michaelyusak/go-helper/helper"
	"github.com/sirupsen/logrus"
)

type streamHandler struct {
	interval      entity.CandleInterval
	exchange      string
	symbol        string
	playbackSpeed float32
	startTime     time.Time
	endTime       time.Time
	token         string

	cleanedAt time.Time
}

type replay struct {
	candles1mRepo repository.Candles1m
	chMap         map[string]streamHandler

	chTtl time.Duration

	tokenLen int

	mu sync.Mutex
}

func NewReplay(
	candles1mRepo repository.Candles1m,
) *replay {
	s := replay{
		candles1mRepo: candles1mRepo,
		chMap:         map[string]streamHandler{},

		chTtl: 24 * time.Hour,

		tokenLen: 20,
	}

	go s.runStreamHandlerCleaner()

	return &s
}

func (s *replay) runStreamHandlerCleaner() {
	tic := time.NewTicker(time.Hour)

	for {
		<-tic.C

		s.cleanStreamHandler()
	}
}

func (s *replay) cleanStreamHandler() {
	s.mu.Lock()
	defer s.mu.Unlock()

	now := time.Now()

	newMap := map[string]streamHandler{}

	for ch, handler := range s.chMap {
		if now.Before(handler.cleanedAt) {
			newMap[ch] = handler
		}
	}

	s.chMap = newMap
}

func (s *replay) CreateStream(ctx context.Context, req entity.CreateStreamReq) (entity.CreateStreamRes, error) {
	startTime := time.UnixMilli(req.StartTimeUnixMilli)
	endTime := time.UnixMilli(req.EndTimeUnixMilli)

	var candlesCount int64
	var interval entity.CandleInterval

	switch req.Interval {
	case string(entity.CandleInterval1m):
		c, err := s.candles1mRepo.CountCandles1m(ctx, req.Exchange, req.Symbol, startTime, endTime)
		if err != nil {
			logrus.
				WithError(err).
				WithField("start", time.UnixMilli(req.StartTimeUnixMilli).String()).
				WithField("end", time.UnixMilli(req.StartTimeUnixMilli).String()).
				WithField("symbol", req.Symbol).
				WithField("interval", req.Interval).
				Error("[service][stream][CreateStream][candles1mRepo.CountCandles1m]")

			return entity.CreateStreamRes{}, apperror.InternalServerError(apperror.AppErrorOpt{
				Code:    http.StatusUnprocessableEntity,
				Message: fmt.Sprintf("[service][stream][CreateStream][candles1mRepo.CountCandles1m] error: %v", err),
			})
		}

		candlesCount = c
		interval = entity.CandleInterval1m
	default:
		logrus.
			WithField("start", time.UnixMilli(req.StartTimeUnixMilli).String()).
			WithField("end", time.UnixMilli(req.StartTimeUnixMilli).String()).
			WithField("symbol", req.Symbol).
			WithField("interval", req.Interval).
			Error("[service][stream][CreateStream] interval not implemented")

		return entity.CreateStreamRes{}, apperror.BadRequestError(apperror.AppErrorOpt{
			Code:            http.StatusUnprocessableEntity,
			ResponseMessage: fmt.Sprintf("the interval '%s' is not supported", req.Interval),
			Message:         fmt.Sprintf("[service][stream][CreateStream] the interval '%s' is not supported", req.Interval),
		})
	}

	playbackSpeed := float32(1.0)
	if req.PlaybackSpeed > 0 {
		playbackSpeed = req.PlaybackSpeed
	}

	channelHash := helper.HashSHA512(fmt.Sprintf("%s%s%s%v", req.Exchange, req.Symbol, req.Interval, time.Now().UnixMilli()))
	channel := fmt.Sprintf("ch:%s", channelHash)

	token := common.CreateRandomString(s.tokenLen)

	s.mu.Lock()
	s.chMap[channel] = streamHandler{
		interval:      interval,
		exchange:      req.Exchange,
		symbol:        req.Symbol,
		playbackSpeed: playbackSpeed,
		startTime:     startTime,
		endTime:       endTime,
		token:         token,

		cleanedAt: time.Now().Add(s.chTtl),
	}
	s.mu.Unlock()

	return entity.CreateStreamRes{
		CandlesCount: candlesCount,
		Channel:      channel,
	}, nil
}

func (s *replay) StreamReplay(ctx context.Context, ch chan []byte, channel, token string) error {
	s.mu.Lock()
	streamHandler, ok := s.chMap[channel]
	s.mu.Unlock()

	if !ok {
		logrus.Warn("[service][replay][StreamCandles] stream not found")

		return apperror.BadRequestError(apperror.AppErrorOpt{
			Code:    http.StatusNotFound,
			Message: "[service][replay][StreamCandles] stream not found",
		})
	}

	if streamHandler.token != token {
		logrus.Warn("[service][replay][StreamCandles] invalid token")

		return apperror.UnauthorizedError(apperror.AppErrorOpt{
			Message: "[service][replay][StreamCandles] invalid token",
		})
	}

	limit := 5000
	var dbHandler func(cursor time.Time) ([]entity.Candle, error)
	var interval time.Duration

	switch streamHandler.interval {
	case entity.CandleInterval1m:
		dbHandler = func(cursor time.Time) ([]entity.Candle, error) {
			candles, err := s.candles1mRepo.GetCandles(ctx, streamHandler.exchange, streamHandler.symbol, cursor, streamHandler.endTime, limit)
			if err != nil {
				return []entity.Candle{}, err
			}

			return candles, nil
		}
		interval = time.Minute
	}

	candlesBytesCh := make(chan [][]byte)
	pullCh := make(chan bool)
	doneCh := make(chan bool)
	errCh := make(chan error)
	defer func() {
		close(candlesBytesCh)
		close(doneCh)
		close(errCh)
	}()

	latency := interval / time.Duration(streamHandler.playbackSpeed)

	var wg sync.WaitGroup

	wg.Add(1)
	go func() {
		defer wg.Done()

		pullCh <- true

	loop:
		for {
			select {
			case <-doneCh:
				break loop
			case candlesBytes := <-candlesBytesCh:
				candlesCount := len(candlesBytes)

				for i, candleBytes := range candlesBytes {
					ch <- candleBytes

					if i > candlesCount/3 {
						pullCh <- true
					}

					time.Sleep(latency)
				}
			}
		}
	}()

	go func() {
		defer wg.Done()

		cursor := streamHandler.startTime

	loop:
		for {
			<-pullCh

			candles, err := dbHandler(cursor)
			if err != nil {
				doneCh <- true
				errCh <- fmt.Errorf("[service][replay][StreamReplay][dbHandler] error: %w", err)
				break loop
			}

			lastUnix := candles[len(candles)-1].Epoch
			cursor = time.Unix(lastUnix, 0)

			candlesBytes := [][]byte{}

			for _, candle := range candles {
				candleBytes, err := json.Marshal(candle)
				if err != nil {
					doneCh <- true
					errCh <- fmt.Errorf("[service][replay][StreamReplay][json.Marshal(candle)] error: %w", err)
					break loop
				}

				candlesBytes = append(candlesBytes, candleBytes)
			}

			candlesBytesCh <- candlesBytes
		}
	}()

	wg.Wait()

	err, ok := <-errCh
	if ok && err != nil {
		logrus.WithError(err).Error("[service][replay][StreamReplay][dbHandler]")
		return apperror.InternalServerError(apperror.AppErrorOpt{
			Message: fmt.Sprintf("[service][replay][StreamReplay][dbHandler] error: %v", err),
		})
	}

	logrus.Info("[service][replay][StreamReplay] done")

	return nil
}
