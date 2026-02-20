package entity

import (
	"github.com/shopspring/decimal"
)

type Candle struct {
	Epoch    int64           `json:"epoch"`
	Pair     string          `json:"pair"`
	Exchange string          `json:"exchange"`
	Open     decimal.Decimal `json:"open"`
	High     decimal.Decimal `json:"high"`
	Low      decimal.Decimal `json:"low"`
	Close    decimal.Decimal `json:"close"`
	Volume   decimal.Decimal `json:"volume"`

	//internal
	Dirty bool `json:"-"`
}
