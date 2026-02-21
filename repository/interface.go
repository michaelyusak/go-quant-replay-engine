package repository

import (
	"context"
	"michaelyusak/go-quant-replay-engine.git/entity"
	"time"
)

type Candles1m interface {
	InsertMany(ctx context.Context, candles []entity.Candle) error
	CountCandles1m(ctx context.Context, exchange, symbol string, start, end time.Time) (int64, error)
	GetCandles(ctx context.Context, exchange, symbol string, cursor, end time.Time, limit int) ([]entity.Candle, error)
}
