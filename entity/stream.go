package entity

import (
	"encoding/json"

	hEntity "github.com/michaelyusak/go-helper/entity"
)

type ReplayConfiguration struct {
	Exchange           string  `json:"exchange" form:"exchange"`
	Symbol             string  `json:"symbol" form:"symbol"`
	PlaybackSpeed      float32 `json:"playback_speed"`
	StartTimeUnixMilli int64   `json:"start_time_unix_milli" form:"start_time_unix_milli"`
	EndTimeUnixMilli   int64   `json:"end_time_unix_milli" form:"end_time_unix_milli"`
}

type CreateStreamReq struct {
	CandleSize hEntity.Duration `json:"candle_size"`
}

type CreateStreamRes struct {
	Channel string `json:"channel"`
	Token   string `json:"token,omitempty"`
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
