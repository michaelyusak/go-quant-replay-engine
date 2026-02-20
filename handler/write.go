package handler

import (
	"michaelyusak/go-quant-replay-engine.git/entity"
	"michaelyusak/go-quant-replay-engine.git/service"

	"github.com/gin-gonic/gin"
	hHelper "github.com/michaelyusak/go-helper/helper"
)

type Write struct {
	writeService service.Write
}

func NewWrite(
	writeService service.Write,
) *Write {
	return &Write{
		writeService: writeService,
	}
}

func (h *Write) ImportFromBinance(ctx *gin.Context) {
	ctx.Header("Content-Type", "application/json")

	var req entity.ImportFromBinanceReq

	err := ctx.ShouldBindJSON(&req)
	if err != nil {
		ctx.Error(err)
		return
	}

	c := ctx.Request.Context()

	err = h.writeService.ImportFromBinance(c, req)
	if err != nil {
		ctx.Error(err)
		return
	}

	hHelper.ResponseOK(ctx, nil)
}
