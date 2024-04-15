package main

import (
	"fmt"
	"os"
	"os/signal"
	"syscall"

	qmq "github.com/rqure/qmq/src"
)

type EngineProcessor struct{}

func (c *EngineProcessor) Process(e qmq.EngineComponentProvider) {
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	for {
		select {
		case <-quit:
			return
		case consumable := <-e.WithConsumer("qmq2mqtt:queue").Pop():
			consumable.Ack()

			m := consumable.Data().(*qmq.MqttMessage)
			e.WithLogger().Debug(fmt.Sprintf("Sending MQTT message: %v", m))

			e.WithProducer("mosquitto").Push(m)
		case consumable := <-e.WithConsumer("mosquitto").Pop():
			consumable.Ack()

			m := consumable.Data().(*qmq.MqttMessage)
			key := "qmq2mqtt:exchange:" + m.Topic

			e.WithLogger().Debug(fmt.Sprintf("Received MQTT message: %v", m))
			e.WithProducer(key).Push(m)
		}
	}
}
