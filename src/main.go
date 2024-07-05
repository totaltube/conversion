package main

import (
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/gin-contrib/requestid"
	"github.com/gin-gonic/gin"
	"github.com/jpillora/overseer"
)
var port = os.Getenv("TOTALTUBE_CONVERSION_PORT")
var version = "dev"
var conversionPath = os.Getenv("TOTALTUBE_CONVERSION_PATH")
var conversionApiKey = os.Getenv("TOTALTUBE_CONVERSION_API_KEY")

func main() {
	log.SetFlags(log.Lshortfile)
	if port == "" {
		port = "8080"
	}
	if conversionPath == "" {
		conversionPath = "/data"
	}
	if conversionApiKey == "" {
		log.Fatalln("Please, set TOTALTUBE_CONVERSION_API_KEY environment variable")
	}
	overseer.Run(overseer.Config{
		Program: server,
		Address: fmt.Sprintf(":%s", port),
		Debug:   false,
	})
}

func server(state overseer.State) {
	gin.SetMode(gin.ReleaseMode)
	app := gin.New()
	app.RedirectTrailingSlash = false
	app.RedirectFixedPath = false
	app.Use(gin.Recovery())
	app.Use(requestid.New())
	app.Use(authorizationMiddleware())
	setRoutes(app)
	log.Println("Starting totaltube conversion version", version, "on port", port)
	go func() {
		err := app.RunListener(state.Listener)
		if err != nil {
			log.Println(err)
		}
	}()
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt)
	signal.Notify(c, syscall.SIGTERM)
	signal.Notify(c, syscall.SIGABRT)
	signal.Notify(c, syscall.SIGKILL)
	select {
	case <-c:
	case <-state.GracefulShutdown:
	}
	log.Println("Cleaning before exit...")
}
