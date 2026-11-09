package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"time"

	"telemetry-daemon/internal/collector"
)

func runCollector(ctx context.Context, out chan<- collector.SystemMetrics) {
	defer close(out)

	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				m, err := collector.FetchMetrics()
				if err != nil {
					log.Println("collect:", err)
					continue
				}
				out <- m
		}
	}
}


func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()

	metrics := make(chan collector.SystemMetrics)
	go runCollector(ctx, metrics)

	for m := range metrics {
		fmt.Printf("%+v\n", m)
	}
}
