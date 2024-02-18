package main

import (
	"fmt"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"time"

	qmq "github.com/rqure/qmq/src"
)

func main() {
	app := qmq.NewQMQApplication("logger")
	app.Initialize()
	defer app.Deinitialize()

	key := os.Getenv("APP_NAME") + ":logs"
	app.AddConsumer(key).Initialize()
	app.Consumer(key).ResetLastId()

	tickRateMs, err := strconv.Atoi(os.Getenv("TICK_RATE_MS"))
	if err != nil {
		tickRateMs = 100
	}

	sigint := make(chan os.Signal, 1)
	signal.Notify(sigint, os.Interrupt)

	ticker := time.NewTicker(time.Duration(tickRateMs) * time.Millisecond)
	for {
		select {
		case <-sigint:
			app.Logger().Advise("SIGINT received")
			return
		case <-ticker.C:
			logMsg := &qmq.QMQLog{}

			for {
				popped := app.Consumer(key).Pop(logMsg)
				if popped == nil {
					break
				}

				fmt.Printf("%s | %s | %s | %s\n", logMsg.Timestamp.AsTime().String(), logMsg.Application, strings.Replace(logMsg.Level.String(), "LOG_LEVEL_", "", -1), logMsg.Message)
				popped.Ack()
			}
		}
	}
}
