package main

import (
	"net/http"

	"github.com/labstack/echo/v4"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	k8client "github.com/vivekganesan01/k8sClusterVitals/internals"
	helpers "github.com/vivekganesan01/k8sClusterVitals/pkg"
)

var cacheStore *helpers.KeyValueStore

func init() {
	log.Info().Str("caller", "main.go").Msg("Welcome to k8sClusterVitals ... starting....")
	zerolog.TimeFieldFormat = zerolog.TimeFormatUnix
	log.Info().Str("caller", "main.go").Msg("Initialising cache server....")
	cacheStore = helpers.NewKeyValueStore()
}

func main() {
	go httpServer()
	watcher, err := k8client.NewKubeClient(cacheStore)
	if err != nil {
		log.Error().Str("caller", "main.go").Msg(helpers.LogMsg("Failed to create kubeclient", err.Error()))
	}
	log.Info().Str("caller", "main.go").Msg("starting to watch resources .... starting ....")
	// Start watching resources
	if watcher != nil {
		watcher.StartWatchingResources()
	} else {
		log.Error().Str("caller", "main.go").Msg("watcher is nil, unable to start watching resources")
		return
	}
	// Keep the program running
	select {}
}

func httpServer() {
	e := echo.New()
	e.GET("/healthcheck/v1/health", func(c echo.Context) error {
		if cacheStore.LenAll() >= 1 {
			return c.String(http.StatusServiceUnavailable, "not_ok")
		}
		return c.String(http.StatusOK, "ok")
	})
	e.GET("/healthcheck/v1/status", func(c echo.Context) error {
		status, err := cacheStore.GetAll()
		if err != nil {
			return c.String(http.StatusInternalServerError, "failed to retrieve the status report")
		}
		return c.JSON(http.StatusOK, status)
	})
	e.GET("/healthcheck/v1/triage", func(c echo.Context) error {
		return c.String(http.StatusOK, "todo: triage endpoint")
	})
	e.Logger.Fatal(e.Start(":1323"))
}
