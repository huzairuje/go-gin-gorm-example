package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"github.com/gookit/event"
	"go-gin-gorm-example/utils"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"go-gin-gorm-example/boot"
	"go-gin-gorm-example/infrastructure/config"
	"go-gin-gorm-example/router"
)

func main() {
	flag.StringVar(&config.Env, "env", "local", "A config name that used by server")
	flag.Parse()

	setup := boot.MakeHandler()

	//set startup
	setup.TriggerStartUp()

	handlerRouter := router.NewHandlerRouter(setup)
	app := handlerRouter.RouterWithMiddleware()

	// add listener for shut down event
	setup.ListenForShutdownEvent()

	port := fmt.Sprintf(":%v", config.Conf.Port)
	if port == "" {
		port = fmt.Sprintf(":%v", 1234)
	}

	log.Printf("Server running on port %s", port)
	serve := &http.Server{
		Addr:    port,
		Handler: app,
	}

	// Start server
	go func() {
		if err := serve.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Fatal("shutting down the server")
		}
	}()

	// Wait for interrupt signal to gracefully shut down the server with
	// a timeout of 1 second.
	quit := make(chan os.Signal)
	// kill (no param) default sends syscall.SIGTERM
	// kill -2 is syscall.SIGINT
	// kill -9 is syscall. SIGKILL but can"t be caught, so don't need to add it
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	log.Println("Shutdown Server ...")

	event.MustFire(utils.ShutDownEvent, nil)

	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()
	if err := serve.Shutdown(ctx); err != nil {
		log.Fatal("Server Shutdown:", err)
	}
	select {
	case <-ctx.Done():
		log.Println("timeout of 1 seconds.")
	}
	log.Println("Server exiting")
}
