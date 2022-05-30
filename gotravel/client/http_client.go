package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"os"

	"github.com/kr/pretty"
	"github.com/radekwlsk/go-travel/gotravel/gotravelservice"
	"github.com/radekwlsk/go-travel/gotravel/gotravelservice/trip"
	"github.com/radekwlsk/go-travel/gotravel/gotraveltransport"
)

func main() {
	var (
		httpAddr = flag.String("http-addr", "127.0.0.1:8080",
			"HTTP address of gotravelcli in host:port format")
		method = flag.String("method", "tripplan", "tripplan, ")
	)
	flag.Parse()

	var (
		svc gotravelservice.Service
		err error
	)

	if *httpAddr != "" {
		svc, err = gotraveltransport.MakeHTTPClient(*httpAddr)
	}

	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}

	switch *method {
	case "tripplan":
		raw, err := ioutil.ReadFile(flag.Args()[0])
		if err != nil {
			fmt.Fprintf(os.Stderr, "error reading JSON file: %v\n", err)
			os.Exit(1)
		}
		var tc trip.Configuration
		json.Unmarshal(raw, &tc)
		ctx := context.Background()
		tripPlan(ctx, svc, tc)

	default:
		fmt.Fprintf(os.Stderr, "error: invalid method %q\n", method)
		os.Exit(1)
	}

}

func tripPlan(ctx context.Context, service gotravelservice.Service, tc trip.Configuration) {
	t, err := service.TripPlan(ctx, tc)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
	fmt.Fprintf(os.Stdout, "%s", pretty.Sprint(t))
}
