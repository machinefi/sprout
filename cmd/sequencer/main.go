package main

import (
	"flag"
	"log"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
)

var (
	logLevel           int
	aggregationAmount  uint
	address            string
	coordinatorAddress string
	databaseDSN        string
)

func init() {
	flag.IntVar(&logLevel, "logLevel", int(slog.LevelDebug), "golang slog level")
	flag.UintVar(&aggregationAmount, "aggregationAmount", 1, "the amount for pack how many messages into one task")
	flag.StringVar(&address, "address", ":9000", "http listen address")
	flag.StringVar(&coordinatorAddress, "coordinatorAddress", "localhost:9001", "coordinator address")
	flag.StringVar(&databaseDSN, "databaseDSN", "postgres://test_user:test_passwd@localhost:5432/test?sslmode=disable", "database dsn")
}

func main() {
	flag.Parse()

	slog.SetDefault(slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.Level(logLevel)})))

	p, err := newPersistence(databaseDSN)
	if err != nil {
		log.Fatal(err)
	}

	go func() {
		if err := newHttpServer(p, aggregationAmount, coordinatorAddress).run(address); err != nil {
			log.Fatal(err)
		}
	}()

	done := make(chan os.Signal, 1)
	signal.Notify(done, syscall.SIGINT, syscall.SIGTERM)
	<-done
}
