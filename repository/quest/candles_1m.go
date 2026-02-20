package quest

import (
	"context"
	"database/sql"
	"fmt"
	"michaelyusak/go-quant-replay-engine.git/entity"
	"strings"
	"time"
)

type candles1m struct {
	db *sql.DB
}

func NewCandles1m(db *sql.DB) *candles1m {
	return &candles1m{
		db: db,
	}
}

func (r *candles1m) InsertMany(ctx context.Context, candles []entity.Candle) error {
	var sb strings.Builder
	sb.WriteString("INSERT INTO candles_1m (timestamp, exchange, symbol, open, high, low, close, volume) VALUES ")

	vals := make([]any, 0, len(candles)*8)
	for i, candle := range candles {
		if i > 0 {
			sb.WriteString(",")
		}

		fmt.Fprintf(&sb, "($%d,$%d,$%d,$%d,$%d,$%d,$%d,$%d)", i*8+1, i*8+2, i*8+3, i*8+4, i*8+5, i*8+6, i*8+7, i*8+8)

		openFl, _ := candle.Open.Float64()
		highFl, _ := candle.High.Float64()
		lowFl, _ := candle.Low.Float64()
		closeFl, _ := candle.Close.Float64()
		volFl, _ := candle.Volume.Float64()

		vals = append(vals, time.Unix(candle.Epoch, 0), candle.Exchange, candle.Pair, openFl, highFl, lowFl, closeFl, volFl)
	}

	_, err := r.db.ExecContext(ctx, sb.String(), vals...)
	if err != nil {
		return fmt.Errorf("[repository][quest][candles1m][InsertMany][db.ExecContext] error: %w", err)
	}

	return nil
}
