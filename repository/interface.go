package repository

import (
	"context"
	"michaelyusak/go-quant-replay-engine.git/entity"
)

type Candles1m interface {
	InsertMany(ctx context.Context, candles []entity.Candle) error
}
