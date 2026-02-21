package server

import (
	binancehttp "michaelyusak/go-quant-replay-engine.git/adapter/binance_http"
	"michaelyusak/go-quant-replay-engine.git/config"
	"michaelyusak/go-quant-replay-engine.git/handler"
	"michaelyusak/go-quant-replay-engine.git/repository/quest"
	"michaelyusak/go-quant-replay-engine.git/service"
	"net/http"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
	hAdaptor "github.com/michaelyusak/go-helper/adaptor"
	hHandler "github.com/michaelyusak/go-helper/handler"
	hMiddleware "github.com/michaelyusak/go-helper/middleware"
	"github.com/sirupsen/logrus"
)

type routerOpts struct {
	handler struct {
		common *hHandler.Common
		write  *handler.Write
		replay *handler.Replay
	}
}

func newRouter(config *config.AppConfig) *gin.Engine {
	db, err := hAdaptor.ConnectDB(hAdaptor.PSQL, config.Service.Db)
	if err != nil {
		logrus.Panicf("Failed to connect to db: %v", err)
	}
	logrus.Info("Connected to postgres")

	candles1mRepo := quest.NewCandles1m(db)

	binanceHttpAdapter := binancehttp.NewAdapter(config.Adapter.BinanceHttp.FapiBaseUrl)

	upgrader := websocket.Upgrader{
		ReadBufferSize:  1024,
		WriteBufferSize: 1024,
		CheckOrigin: func(r *http.Request) bool {
			return true
		},
	}

	writeService := service.NewWrite(candles1mRepo, binanceHttpAdapter)
	replayService := service.NewReplay(candles1mRepo)

	commonHandler := hHandler.NewCommon(&APP_HEALTHY)
	writeHandler := handler.NewWrite(writeService)
	replayHandler := handler.NewReplay(replayService, upgrader)

	return createRouter(routerOpts{
		handler: struct {
			common *hHandler.Common
			write  *handler.Write
			replay *handler.Replay
		}{
			common: commonHandler,
			write:  writeHandler,
			replay: replayHandler,
		},
	},
		config.Cors.AllowedOrigins,
	)
}

func createRouter(opts routerOpts, allowedOrigins []string) *gin.Engine {
	router := gin.New()

	corsConfig := cors.DefaultConfig()

	router.ContextWithFallback = true

	router.Use(
		hMiddleware.Logger(logrus.New()),
		hMiddleware.RequestIdHandlerMiddleware,
		hMiddleware.ErrorHandlerMiddleware,
		gin.Recovery(),
	)

	corsRouting(router, corsConfig, allowedOrigins)
	commonRouting(router, opts.handler.common)
	writeRouting(router, opts.handler.write)

	return router
}

func corsRouting(router *gin.Engine, corsConfig cors.Config, allowedOrigins []string) {
	corsConfig.AllowOrigins = allowedOrigins
	corsConfig.AllowMethods = []string{"POST", "GET", "PUT", "PATCH", "DELETE"}
	corsConfig.AllowHeaders = []string{"Origin", "Authorization", "Content-Type", "Accept", "User-Agent", "Cache-Control", "Device-Info", "X-Device-Id"}
	corsConfig.ExposeHeaders = []string{"Content-Length"}
	corsConfig.AllowCredentials = true
	router.Use(cors.New(corsConfig))
}

func commonRouting(router *gin.Engine, handler *hHandler.Common) {
	router.GET("/health", handler.Health)
	router.NoRoute(handler.NoRoute)
}

func staticRouting(router *gin.Engine, localStorageStaticPath, localStorageDirectory string) {
	router.Static(localStorageStaticPath, localStorageDirectory)
}

func writeRouting(router *gin.Engine, handler *handler.Write) {
	router.POST("/v1/write/binance", handler.ImportFromBinance)
}

func replayRouting(router *gin.Engine, handler *handler.Replay) {
	router.POST("/v1/replay/create", handler.CreateStream)
	router.GET("/v1/replay/stream", handler.StreamReplay)
}
