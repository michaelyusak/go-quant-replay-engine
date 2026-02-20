package service

import (
	"context"
	"michaelyusak/go-quant-replay-engine.git/entity"
)

type Write interface {
	ImportFromBinance(ctx context.Context, req entity.ImportFromBinanceReq) error
}

type Replay interface{}
