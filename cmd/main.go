package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"github.com/xenn00/chat-system/config"
	"github.com/xenn00/chat-system/internal/routers"
	"github.com/xenn00/chat-system/internal/websocket"
	"github.com/xenn00/chat-system/internal/worker"
	"github.com/xenn00/chat-system/state"
)

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	// initialize the application
	log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stderr, TimeFormat: time.RFC3339})

	err := config.LoadConfig()
	if err != nil {
		log.Fatal().Err(err).Msg("failed to load configuration")
	}

	state, err := state.InitAppState(ctx, stop)
	if err != nil {
		log.Fatal().Err(err).Msg("failed to initialize application state")
	}
	defer state.Close()

	r := routers.NewRouter(state)

	server := &http.Server{
		Addr:    config.Conf.App.Port,
		Handler: r,
	}

	wsHub := websocket.NewHub()
	workerPool := worker.NewWorkerPool(state.Redis, 5, wsHub)

	go workerPool.Start(ctx)
	go workerPool.StartDLQWorker(ctx)

	// serve the application
	go func() {
		log.Info().Msgf("Starting server on http://localhost%s", config.Conf.App.Port)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			panic(fmt.Sprintf("ListenAndServe failed: %v", err))
		}
	}()

	<-ctx.Done()
	log.Info().Msg("Shutdown initiated...")
	// gracefully shutdown the application
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	// perform shutdown tasks here, like closing connections, etc.
	if err := server.Shutdown(shutdownCtx); err != nil {
		fmt.Printf("Graceful shutdown failed: %v\n", err)
	} else {
		fmt.Println("Server exited gracefully.")
	}
	workerPool.Wait()
}
