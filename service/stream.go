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

	replayConfiguration entity.ReplayConfiguration

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

func (s *replay) GetConfiguration(ctx context.Context) entity.ReplayConfiguration {
	return s.replayConfiguration
}

func (s *replay) UpdateConfiguration(ctx context.Context, newConf entity.ReplayConfiguration) {
	s.replayConfiguration = newConf
}

func (s *replay) CreateStream(ctx context.Context, req entity.CreateStreamReq) (entity.CreateStreamRes, error) {
	var interval entity.CandleInterval

	switch time.Duration(req.CandleSize) {
	case time.Minute:
		interval = entity.CandleInterval1m
	default:
		return entity.CreateStreamRes{}, apperror.BadRequestError(apperror.AppErrorOpt{
			Code:            http.StatusUnprocessableEntity,
			ResponseMessage: fmt.Sprintf("the interval '%s' is not supported", time.Duration(req.CandleSize).String()),
			Message:         fmt.Sprintf("[service][stream][CreateStream] the interval '%s' is not supported", time.Duration(req.CandleSize).String()),
		})
	}

	channelHash := helper.HashSHA512(fmt.Sprintf("%s%v", time.Duration(req.CandleSize).String(), time.Now().UnixMilli()))
	channel := fmt.Sprintf("ch:%s", channelHash)

	token := common.CreateRandomString(s.tokenLen)

	s.mu.Lock()
	s.chMap[channel] = streamHandler{
		interval:      interval,
		exchange:      s.replayConfiguration.Exchange,
		symbol:        s.replayConfiguration.Symbol,
		playbackSpeed: s.replayConfiguration.PlaybackSpeed,
		startTime:     time.UnixMilli(s.replayConfiguration.StartTimeUnixMilli),
		endTime:       time.UnixMilli(s.replayConfiguration.EndTimeUnixMilli),
		token:         token,

		cleanedAt: time.Now().Add(s.chTtl),
	}
	s.mu.Unlock()

	fmt.Println("test", fmt.Sprintf("%+v", s.chMap))

	return entity.CreateStreamRes{
		Channel: channel,
		Token:   token,
	}, nil
}

func (s *replay) StreamReplay(ctx context.Context, ch chan []byte, channel, token string) error {
	defer close(ch)

	logrus.
		WithField("channel", channel).
		Info("[service][replay][StreamReplay] starting stream...")

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

	logrus.Info("[service][replay][StreamCandles] stream authenticated")

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
	default:
		return apperror.BadRequestError(apperror.AppErrorOpt{
			Code:            http.StatusBadRequest,
			ResponseMessage: fmt.Sprintf("interval of %s is unavailable", streamHandler.interval),
		})
	}

	candlesBytesCh := make(chan [][]byte, limit)
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

		logrus.Info("[service][replay][StreamCandles][emitter] ready to emit")

	loop:
		for {
			pullSignalSent := false

			select {
			case <-doneCh:
				break loop
			case candlesBytes := <-candlesBytesCh:
				candlesCount := len(candlesBytes)

				logrus.WithField("candles_count", candlesCount).Info("[service][replay][StreamCandles][emitter] emiting candles...")

				for i, candleBytes := range candlesBytes {
					if len(candleBytes) == 0 {
						continue
					}

					ch <- candleBytes

					if !pullSignalSent && i > candlesCount/2 {
						logrus.
							WithField("candles_count", candlesCount).
							WithField("i", i).
							Info("[service][replay][StreamCandles][emitter] sending pull signal")
						pullCh <- true
						pullSignalSent = true
					}

					time.Sleep(latency)
				}
			}
		}
	}()

	wg.Add(1)
	go func() {
		defer wg.Done()

		cursor := streamHandler.startTime

	loop:
		for {
			select {
			case <-pullCh:
				logrus.
					WithField("cursor", cursor.String()).
					Info("[service][replay][StreamReplay][puller] got pull signal. pulling candles...")

				candles, err := dbHandler(cursor)
				if err != nil {
					doneCh <- true
					errCh <- fmt.Errorf("[service][replay][StreamReplay][puller] dbHandler error: %w", err)
					break loop
				}

				candlesBytes := [][]byte{}
				for _, candle := range candles {
					candle.Symbol = fmt.Sprintf("%s:%s", candle.Exchange, candle.Pair)

					candleBytes, err := json.Marshal(candle)
					if err != nil {
						doneCh <- true
						errCh <- fmt.Errorf("[service][replay][StreamReplay][json.Marshal(candle)] error: %w", err)
						break loop
					}

					candlesBytes = append(candlesBytes, candleBytes)
				}

				candlesBytesCh <- candlesBytes

				if len(candles) < limit {
					doneCh <- true
					break loop
				}

				lastUnix := candles[len(candles)-1].Epoch
				cursor = time.Unix(lastUnix, 0)
			case <-ctx.Done():
				doneCh <- true
				break loop
			}
		}
	}()

	wg.Wait()

	err, ok := <-errCh
	if ok && err != nil {
		logrus.
			WithError(err).
			WithField("channel", channel).
			Error("[service][replay][StreamReplay]")

		return apperror.InternalServerError(apperror.AppErrorOpt{
			Message: fmt.Sprintf("[service][replay][StreamReplay] error: %v", err),
		})
	}

	logrus.
		WithField("channel", channel).Info("[service][replay][StreamReplay] done")

	return nil
}
