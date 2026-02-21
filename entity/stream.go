package entity

import "encoding/json"

type CreateStreamReq struct {
	Exchange           string  `json:"exchange" form:"exchange" binding:"required"`
	Symbol             string  `json:"symbol" form:"symbol" binding:"required"`
	Interval           string  `json:"interval"`
	PlaybackSpeed      float32 `json:"playback_speed"`
	StartTimeUnixMilli int64   `json:"start_time_unix_milli" form:"start_time_unix_milli" binding:"required"`
	EndTimeUnixMilli   int64   `json:"end_time_unix_milli" form:"end_time_unix_milli" binding:"required"`
}

type CreateStreamRes struct {
	CandlesCount int64  `json:"candle_count"`
	Channel      string `json:"channel"`
	Token        string `json:"token,omitempty"`
}

type WsMessage struct {
	Type string          `json:"type"`
	Data json.RawMessage `json:"data"`
}

type WsMessageType string

const (
	WsMessageTypeAuth WsMessageType = "auth"
)

type WsAuthData struct {
	Channel string `json:"channel"`
	Token   string `json:"token"`
}
