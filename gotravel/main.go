package gotravel

import (
	"flag"
	"fmt"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"github.com/go-kit/kit/log"
	"github.com/radekwlsk/go-travel/gotravel/gotravelendpoint"
	"github.com/radekwlsk/go-travel/gotravel/gotravelservice"
	"github.com/radekwlsk/go-travel/gotravel/gotraveltransport"
)

func main() {
	var (
		httpAddr = flag.String("http-addr", ":8080", "HTTP port to listen")
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

	var (
		service     = gotravelservice.New(logger)
		endpoints   = gotravelendpoint.New(service, logger)
		httpHandler = gotraveltransport.MakeHTTPHandler(endpoints, logger)
	)

	errs := make(chan error)
	go func() {
		c := make(chan os.Signal)
		signal.Notify(c, syscall.SIGINT, syscall.SIGTERM)
		errs <- fmt.Errorf("%s", <-c)
	}()

	go func() {
		httpListener, err := net.Listen("tcp", *httpAddr)
		if err != nil {
			errs <- err
			return
		}
		logger.Log("transport", "HTTP", "addr", *httpAddr)
		errs <- http.Serve(httpListener, httpHandler)
	}()

	logger.Log("exit", <-errs)
}
