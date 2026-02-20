package service

import (
	"context"
	"fmt"
	binancehttp "michaelyusak/go-quant-replay-engine.git/adapter/binance_http"
	"michaelyusak/go-quant-replay-engine.git/entity"
	"michaelyusak/go-quant-replay-engine.git/repository"
	"net/http"
	"time"

	"github.com/michaelyusak/go-helper/apperror"
	"github.com/sirupsen/logrus"
)

type write struct {
	candles1mRepo      repository.Candles1m
	binanceHttpAdapter *binancehttp.Adapter
}

func NewWrite(
	candles1mRepo repository.Candles1m,
	binanceHttpAdapter *binancehttp.Adapter,
) *write {
	return &write{
		candles1mRepo:      candles1mRepo,
		binanceHttpAdapter: binanceHttpAdapter,
	}
}

func (s *write) ImportFromBinance(ctx context.Context, req entity.ImportFromBinanceReq) error {
	intervalSeconds := map[string]int64{
		"1m": 60,
	}

	logrus.
		WithField("start", time.UnixMilli(req.StartTimeUnixMilli).String()).
		WithField("end", time.UnixMilli(req.StartTimeUnixMilli).String()).
		WithField("limit", req.Limit).
		WithField("symbol", req.Symbol).
		WithField("interval", req.Interval).
		Info("[service][write][ImportFromBinance] import started")

	for {
		candles, err := s.binanceHttpAdapter.GetCandleStickData(ctx, req.StartTimeUnixMilli, req.EndTimeUnixMilli, req.Limit, req.Symbol, req.Interval)
		if err != nil {
			logrus.
				WithError(err).
				WithField("start", time.UnixMilli(req.StartTimeUnixMilli).String()).
				WithField("end", time.UnixMilli(req.StartTimeUnixMilli).String()).
				WithField("limit", req.Limit).
				WithField("symbol", req.Symbol).
				WithField("interval", req.Interval).
				Error("[service][write][ImportFromBinance][binanceHttpAdapter.GetCandleStickData]")

			return apperror.InternalServerError(apperror.AppErrorOpt{
				Message: fmt.Sprintf("[service][write][ImportFromBinance][binanceHttpAdapter.GetCandleStickData] error: %v", err),
			})
		}

		if len(candles) == 0 {
			break
		}

		switch req.Interval {
		case "1m":
			err = s.candles1mRepo.InsertMany(ctx, candles)
			if err != nil {
				logrus.
					WithError(err).
					WithField("start", time.UnixMilli(req.StartTimeUnixMilli).String()).
					WithField("end", time.UnixMilli(req.StartTimeUnixMilli).String()).
					WithField("limit", req.Limit).
					WithField("symbol", req.Symbol).
					WithField("interval", req.Interval).
					Error("[service][write][ImportFromBinance][candles1mRepo.InsertMany]")

				return apperror.InternalServerError(apperror.AppErrorOpt{
					Message: fmt.Sprintf("[service][write][ImportFromBinance][candles1mRepo.InsertMany] error: %v", err),
				})
			}
		default:
			logrus.
				WithError(err).
				WithField("start", time.UnixMilli(req.StartTimeUnixMilli).String()).
				WithField("end", time.UnixMilli(req.StartTimeUnixMilli).String()).
				WithField("limit", req.Limit).
				WithField("symbol", req.Symbol).
				WithField("interval", req.Interval).
				Error("[service][write][ImportFromBinance] interval not implemented")

			return apperror.BadRequestError(apperror.AppErrorOpt{
				Code:            http.StatusUnprocessableEntity,
				ResponseMessage: fmt.Sprintf("the interval '%s' is not supported", req.Interval),
				Message:         fmt.Sprintf("[service][write][ImportFromBinance] the interval '%s' is not supported", req.Interval),
			})
		}

		var lastEpoch int64
		for _, c := range candles {
			if c.Epoch > lastEpoch {
				lastEpoch = c.Epoch
			}
		}

		increment := intervalSeconds[req.Interval]

		req.StartTimeUnixMilli = (lastEpoch + increment) * 1000

		if req.EndTimeUnixMilli != 0 && req.StartTimeUnixMilli > req.EndTimeUnixMilli {
			logrus.
				WithField("end", time.UnixMilli(req.StartTimeUnixMilli).String()).
				WithField("limit", req.Limit).
				WithField("symbol", req.Symbol).
				WithField("interval", req.Interval).
				WithField("last_imported", time.Unix(lastEpoch, 0).String()).
				Info("[service][write][ImportFromBinance] import done")
			break
		}

		logrus.
			WithField("end", time.UnixMilli(req.StartTimeUnixMilli).String()).
			WithField("limit", req.Limit).
			WithField("symbol", req.Symbol).
			WithField("interval", req.Interval).
			WithField("last_imported", time.Unix(lastEpoch, 0).String()).
			WithField("next", time.UnixMilli(req.StartTimeUnixMilli).String()).
			Info("[service][write][ImportFromBinance] import in progress")

		time.Sleep(100 * time.Millisecond)
	}

	return nil
}
