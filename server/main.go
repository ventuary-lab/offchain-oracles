package main

import (
	"./routers"
	"context"
//	"golang.org/x/crypto/acme/autocert"
	"net/http"
	"os"
	"os/signal"
	"time"
)

func main() {
	args := os.Args[1:]

	handler := routers.NewRouter()
	srv := &http.Server{
		Addr:         args[0],
		Handler:      handler,
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 10 * time.Second,
		IdleTimeout:  120 * time.Second,
	}

	/*go func() {
		srv.Serve(autocert.NewListener(args[0]))
	}()*/

	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt)
	<-c

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	srv.Shutdown(ctx)
}
