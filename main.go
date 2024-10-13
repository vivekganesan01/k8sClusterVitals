package main

import (
	"context"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"github.com/labstack/echo/v4"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	k8client "github.com/vivekganesan01/k8sClusterVitals/internals"
	helpers "github.com/vivekganesan01/k8sClusterVitals/pkg"
)

var cacheStore *helpers.KeyValueStore

const LabelSelector = "k8sclustervitals.io/scrape=true"

func init() {
	log.Info().Str("caller", "main.go").Msg("Welcome to k8sClusterVitals ... starting....")
	zerolog.TimeFieldFormat = zerolog.TimeFormatUnix
	log.Info().Str("caller", "main.go").Msg("Initialising cache server....")
	cacheStore = helpers.NewKeyValueStore()
}

func main() {
	ctx, cancel := context.WithCancel(context.Background())
	go httpServer(ctx)
	defer cancel() // Ensure the context is cancelled when the main function exits

	watcher, err := k8client.NewKubeClient(cacheStore)
	if err != nil {
		log.Error().Str("caller", "main.go").Msg(helpers.LogMsg("Failed to create kubeclient", err.Error()))
	}
	log.Info().Str("caller", "main.go").Msg("starting to watch resources .... starting ....")
	// Start watching resources
	if watcher != nil {
		// Handle system signals for graceful shutdown
		sigs := make(chan os.Signal, 1)
		signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)
		go func() {
			sig := <-sigs // Wait for a termination signal
			log.Info().Str("signal", sig.String()).Msg("Termination signal received. Initiating shutdown...")
			cancel() // Cancel the context to stop goroutines
		}()

		watcher.StartWatchingResources(ctx, LabelSelector)
		watcher.Wg.Wait()
	} else {
		log.Error().Str("caller", "main.go").Msg("watcher is nil, unable to start watching resources")
		return
	}

	// select{} // here this is not need as we use waitgroup and graceful shutdown
}

func httpServer(ctx context.Context) {
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
		return c.String(http.StatusOK, "todo: triage endpoint under development")
	})
	e.GET("/healthcheck/v1/scrape_configuration", func(c echo.Context) error {
		kv := cacheStore.GoCacheGetAll()
		// Convert map to JSON
		// jsonData, err := json.Marshal(kv)
		// if err != nil {
		// 	return c.String(http.StatusInternalServerError, "failed to retrieve scrape config")
		// }
		return c.JSON(http.StatusOK, kv)
	})
	// Start the server in a goroutine
	go func() {
		log.Info().Msg("Starting HTTP server on :1323...")
		if err := e.Start(":1323"); err != nil && err != http.ErrServerClosed {
			log.Fatal().Msg("Failed to start server")
			log.Fatal().Msg(err.Error())
		}
	}()
	// Wait for the context to be canceled
	<-ctx.Done() // Block until context is canceled
	// Initiate graceful shutdown
	log.Info().Msg("Shutting down HTTP server...")
	if err := e.Shutdown(context.Background()); err != nil {
		log.Fatal().Msg("HTTP server forced to shutdown")
	}
	log.Info().Msg("HTTP server exited gracefully")
}
