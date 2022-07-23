package main

import (
	"context"
	"flag"
	"github.com/docker/docker/client"
	"github.com/felipecruz91/vackup-docker-extension/internal/handler"
	"github.com/labstack/echo"
	"github.com/sirupsen/logrus"
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
	"time"
)

var h *handler.Handler

func main() {
	var socketPath string
	flag.StringVar(&socketPath, "socket", "/run/guest/extension-vackup.sock", "Unix domain socket to listen on")
	flag.Parse()

	_ = os.RemoveAll(socketPath)

	logrus.New().Infof("Starting listening on %s\n", socketPath)
	router := echo.New()
	router.HideBanner = true

	ln, err := net.Listen("unix", socketPath)
	if err != nil {
		log.Fatal(err)
	}
	router.Listener = ln

	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		log.Fatal(err)
	}

	h = handler.New(context.Background(), cli)

	router.GET("/volumes", h.Volumes)
	router.GET("/volumes/:volume/size", h.VolumeSize)
	router.GET("/volumes/:volume/export", h.ExportVolume)
	router.GET("/volumes/:volume/import", h.ImportTarGzFile)
	router.GET("/volumes/:volume/save", h.SaveVolume)
	router.GET("/volumes/:volume/load", h.LoadImage)

	// Start server
	go func() {
		if err := router.Start(""); err != nil && err != http.ErrServerClosed {
			logrus.Fatal("shutting down the server")
		}
	}()

	// Wait for interrupt signal to gracefully shut down the server with a timeout of 10 seconds.
	// Use a buffered channel to avoid missing signals as recommended for signal.Notify
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, os.Interrupt)
	<-quit
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := router.Shutdown(ctx); err != nil {
		logrus.Fatal(err)
	}
}
