package binancehttp

import (
	"context"
	"fmt"
	"michaelyusak/go-quant-replay-engine.git/entity"
	"net/http"
	"strconv"

	binanceEntity "michaelyusak/go-quant-replay-engine.git/entity/binance"

	"github.com/shopspring/decimal"
	"github.com/sirupsen/logrus"
)

func (a *Adapter) GetCandleStickData(ctx context.Context, startTime, endTime int64, limit int, symbol, interval string) ([]entity.Candle, error) {
	var res [][]any
	var errRes binanceEntity.FapiGeneralErrorResponse

	r, err := a.client.R().
		SetContext(ctx).
		SetResult(&res).
		SetError(&errRes).
		SetQueryParams(map[string]string{
			"symbol":    symbol,
			"limit":     strconv.Itoa(limit),
			"interval":  interval,
			"startTime": strconv.Itoa(int(startTime)),
		}).
		Get(fmt.Sprintf("%s/v1/klines", a.fapiBaseUrl))
	if err != nil {
		return nil, fmt.Errorf("[adapter][BinanceHttp][GetCandleStickData][Get] error: %w", err)
	}
	if r.StatusCode() >= http.StatusBadRequest {
		return nil, fmt.Errorf("[adapter][BinanceHttp][GetCandleStickData][StatusCode: %v] res: %+v", r.StatusCode(), errRes)
	}

	return a.normalizedCandle(res, symbol), nil
}

func (a *Adapter) normalizedCandle(raw [][]any, symbol string) []entity.Candle {
	normalized := []entity.Candle{}

	for i, data := range raw {
		if len(data) < 6 {
			logrus.
				WithField("row", i).
				WithField("data", data).
				Warn("[adapter][BinanceHttp][NormalizedCandle] skipping row. not enough columns")
			continue
		}

		epochFloat, ok := data[0].(float64)
		if !ok {
			logrus.
				WithField("row", i).
				WithField("epoch", data[0]).
				Warn("[adapter][BinanceHttp][NormalizedCandle] invalid epoch")
			continue
		}
		epochMs := int64(epochFloat)

		openStr, ok1 := data[1].(string)
		highStr, ok2 := data[2].(string)
		lowStr, ok3 := data[3].(string)
		closeStr, ok4 := data[4].(string)
		volStr, ok5 := data[5].(string)
		if !(ok1 && ok2 && ok3 && ok4 && ok5) {
			logrus.
				WithField("row", i).
				WithField("data", data).
				Warn("[adapter][BinanceHttp][NormalizedCandle] invalid price/volume strings")
			continue
		}

		openDec, err := decimal.NewFromString(openStr)
		if err != nil {
			logrus.
				WithField("row", i).
				WithField("open", openStr).
				Warn("[adapter][BinanceHttp][NormalizedCandle] invalid open decimal")
			continue
		}
		highDec, err := decimal.NewFromString(highStr)
		if err != nil {
			logrus.
				WithField("row", i).
				WithField("high", highStr).
				Warn("[adapter][BinanceHttp][NormalizedCandle] invalid high decimal")
			continue
		}
		lowDec, err := decimal.NewFromString(lowStr)
		if err != nil {
			logrus.
				WithField("row", i).
				WithField("low", lowStr).
				Warn("[adapter][BinanceHttp][NormalizedCandle] invalid low decimal")
			continue
		}
		closeDec, err := decimal.NewFromString(closeStr)
		if err != nil {
			logrus.
				WithField("row", i).
				WithField("close", closeStr).
				Warn("[adapter][BinanceHttp][NormalizedCandle] invalid close decimal")
			continue
		}
		volDec, err := decimal.NewFromString(volStr)
		if err != nil {
			logrus.
				WithField("row", i).
				WithField("volume", volStr).
				Warn("[adapter][BinanceHttp][NormalizedCandle] invalid volume decimal")
			continue
		}

		candle := entity.Candle{
			Epoch:    epochMs / 1000,
			Pair:     symbol,
			Exchange: "binance",
			Open:     openDec,
			High:     highDec,
			Low:      lowDec,
			Close:    closeDec,
			Volume:   volDec,
		}

		normalized = append(normalized, candle)
	}

	return normalized
}
