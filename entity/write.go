package entity

type ImportFromBinanceReq struct {
	Symbol             string `json:"symbol" form:"symbol"`
	Interval           string `json:"interval" form:"interval"`
	Limit              int    `json:"limit" form:"limit"`
	StartTimeUnixMilli int64  `json:"start_time_unix_milli" form:"start_time_unix_milli"`
	EndTimeUnixMilli   int64  `json:"end_time_unix_milli" form:"end_time_unix_milli"`
}
