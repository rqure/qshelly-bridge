package main

import (
	"os"
	"os/signal"
	"strconv"
	"time"

	mqtt "github.com/eclipse/paho.mqtt.golang"
	qmq "github.com/rqure/qmq/src"
	"google.golang.org/protobuf/encoding/protojson"
)

func main() {
	mqttAddr := os.Getenv("MQTT_ADDR")
	if mqttAddr == "" {
		mqttAddr = "mosquitto"
	}

	defaultProducerLength, err := strconv.Atoi(os.Getenv("QMQ_DEFAULT_PRODUCER_LENGTH"))
	if err != nil {
		defaultProducerLength = 10
	}

	tickRateMs, err := strconv.Atoi(os.Getenv("TICK_RATE_MS"))
	if err != nil {
		tickRateMs = 100
	}

	app := qmq.NewQMQApplication("qmq2mqtt")
	app.Initialize()
	defer app.Deinitialize()
	app.AddConsumer("qmq2mqtt:queue").Initialize()

	opt := mqtt.NewClientOptions()
	opt.AddBroker(mqttAddr)
	client := mqtt.NewClient(opt)
	client.Connect()
	defer client.Disconnect(0)

	client.Subscribe("#", 2, func(c mqtt.Client, m mqtt.Message) {
		key := "qmq2mqtt:exchange:" + m.Topic()

		p := app.Producer(key)
		if p != nil {
			p = app.AddProducer(key)
			p.Initialize(int64(defaultProducerLength))
		}

		msg := &qmq.QMQMqttMessage{
			Topic:     m.Topic(),
			Payload:   m.Payload(),
			Id:        uint32(m.MessageID()),
			Qos:       int32(m.Qos()),
			Retained:  m.Retained(),
			Duplicate: m.Duplicate(),
		}

		app.Logger().Debug("Received MQTT message: " + protojson.Format(msg))
		p.Push(msg)
	})

	sigint := make(chan os.Signal, 1)
	signal.Notify(sigint, os.Interrupt)

	ticker := time.NewTicker(time.Duration(tickRateMs) * time.Millisecond)
	for {
		select {
		case <-sigint:
			app.Logger().Advise("SIGINT received")
			return
		case <-ticker.C:
			if !client.IsConnected() {
				continue
			}

			for {
				msg := &qmq.QMQMqttMessage{}

				popped := app.Consumer("qmq2mqtt:queue").Pop(msg)
				if popped == nil {
					break
				}

				app.Logger().Debug("Sending MQTT message: " + protojson.Format(msg))
				client.Publish(msg.Topic, byte(msg.Qos), msg.Retained, msg.Payload)

				popped.Ack()
			}
		}
	}
}
