package server

import (
	"context"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"michaelyusak/go-quant-replay-engine.git/config"

	hHelper "github.com/michaelyusak/go-helper/helper"
	"github.com/sirupsen/logrus"
)

var (
	APP_HEALTHY = false
)

func Init() {
	conf, err := config.Init()
	if err != nil {
		logrus.Panic(err)
	}

	err = hHelper.SetupLogrus(conf.Log.Level, conf.Log.Dir)
	if err != nil {
		logrus.Panic(err)
	}

	router := newRouter(&conf)

	srv := http.Server{
		Handler: router,
		Addr:    conf.Service.Port,
	}

	go func() {
		logrus.Infof("Sever running on port %s", conf.Service.Port)
		APP_HEALTHY = true

		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logrus.Fatalf("listen: %s\n", err)
		}
	}()

	quit := make(chan os.Signal, 10)

	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	<-quit

	logrus.Infof("Server shutting down in %s ...", time.Duration(conf.Service.GracefulPeriod).String())

	APP_HEALTHY = false

	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(conf.Service.GracefulPeriod))
	defer cancel()

	<-ctx.Done()

	if err := srv.Shutdown(ctx); err != nil {
		logrus.Fatalf("Server shut down with error: %s", err.Error())
	}

	logrus.Info("Server shut down")
}
