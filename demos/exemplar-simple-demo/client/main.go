package main

import (
	"context"
	"fmt"
	"io"
	"log"
	"net/http"
	"time"

	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"

	"go.opentelemetry.io/otel"
)

func main() {
	time.Sleep(5 * time.Second)
	// init opentelemetry trace provider
	shutdown := initTracerProvider()
	defer shutdown()
	log.Println("init tracer provider done")

	client := http.Client{Transport: otelhttp.NewTransport(http.DefaultTransport)}
	ctx := context.Background()
	var body []byte

	tr := otel.Tracer("demo-client-hello")
	for {
		err := func(ctx context.Context) error {
			ctx, span := tr.Start(ctx, "say hello")
			defer span.End()
			req, _ := http.NewRequestWithContext(ctx, "GET", "http://localhost:7777/hello", nil)

			fmt.Printf("sending request...\n")
			res, err := client.Do(req)
			if err != nil {
				panic(err)
			}
			body, err = io.ReadAll(res.Body)
			_ = res.Body.Close()

			return err
		}(ctx)

		if err != nil {
			log.Fatal(err)
		}

		fmt.Printf("response received: %s\n\n\n", body)
		time.Sleep(500 * time.Millisecond)
	}
}
