package quest

import (
	"context"
	"database/sql"
	"errors"
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

func (r *candles1m) CountCandles1m(ctx context.Context, exchange, symbol string, start, end time.Time) (int64, error) {
	q := `
		SELECT COUNT(*)
		FROM candles_1m
		WHERE exchange = $1
			AND symbol = $2
			AND timestamp
				BETWEEN $3 AND $4
	`

	var count int64
	err := r.db.QueryRowContext(ctx, q, exchange, symbol, start, end).Scan(&count)
	if err != nil {
		return count, fmt.Errorf("[repository][quest][candles1m][CountCandles1m][db.QueryRowContext] error: %w", err)
	}

	return count, nil
}

func (r *candles1m) GetCandles(ctx context.Context, exchange, symbol string, cursor, end time.Time, limit int) ([]entity.Candle, error) {
	q := `
		SELECT timestamp, open, high, low, close, volume 
		FROM candles_1m
		WHERE exchange = $1
			AND symbol = $2
			AND timestamp
				BETWEEN $3 AND $4
		LIMIT $5
		ORDER BY timestamp ASC
	`

	rows, err := r.db.QueryContext(ctx, q, exchange, symbol, cursor, end, limit)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return []entity.Candle{}, nil
		}

		return []entity.Candle{}, fmt.Errorf("[repository][quest][candles1m][GetCandles][db.QueryContext] error: %w", err)
	}
	defer rows.Close()

	candles := []entity.Candle{}

	for rows.Next() {
		var candle entity.Candle
		var candleTs time.Time

		err := rows.Scan(
			&candleTs,
			&candle.Open,
			&candle.High,
			&candle.Low,
			&candle.Close,
			&candle.Volume,
		)
		if err != nil {
			return []entity.Candle{}, fmt.Errorf("[repository][quest][candles1m][GetCandles][rows.Scan] error: %w", err)
		}

		candle.Epoch = candleTs.Unix()
		candle.Exchange = exchange
		candle.Pair = symbol

		candles = append(candles, candle)
	}

	return candles, nil
}
