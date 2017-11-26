package main

import (
	"flag"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"github.com/go-kit/kit/log"

	"github.com/afrometal/go-travel/gotravel/gotravelsvc"
)

func main() {
	var (
		httpAddr = flag.String("http.addr", ":8080", "HTTP port to listen")
	)
	flag.Parse()

	var logger log.Logger
	{
		logger = log.NewLogfmtLogger(os.Stderr)
		logger = log.With(logger, "ts", log.DefaultTimestampUTC)
		logger = log.With(logger, "caller", log.DefaultCaller)
	}

	logger.Log("msg", "gotravel service started")
	defer logger.Log("msg", "finished")

	var s gotravelsvc.Service
	{
		s = gotravelsvc.NewInmemService()
		s = gotravelsvc.NewLoggingMiddleware(s, logger)
		//s = gotravelsvc.NewInstrumentingMiddleware(s)
	}

	var h http.Handler
	{
		h = gotravelsvc.MakeHTTPHandler(s, log.With(logger, "component", "HTTP"))
	}

	errs := make(chan error)
	go func() {
		c := make(chan os.Signal)
		signal.Notify(c, syscall.SIGINT, syscall.SIGTERM)
		errs <- fmt.Errorf("%s", <-c)
	}()

	go func() {
		logger.Log("transport", "HTTP", "addr", *httpAddr)
		errs <- http.ListenAndServe(*httpAddr, h)
	}()

	logger.Log("exit", <-errs)
}
