package binance

type FapiGeneralErrorResponse struct {
	Code int    `json:"code"`
	Msg  string `json:"msg"`
}
