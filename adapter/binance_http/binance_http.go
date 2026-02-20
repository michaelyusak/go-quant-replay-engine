package binancehttp

import (
	"github.com/go-resty/resty/v2"
)

type Adapter struct {
	fapiBaseUrl string
	client      *resty.Client
}

func NewAdapter(fapiBaseUrl string) *Adapter {
	return &Adapter{
		fapiBaseUrl: fapiBaseUrl,
		client:      resty.New(),
	}
}
