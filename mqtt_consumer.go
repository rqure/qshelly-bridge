package main

import (
	mqtt "github.com/eclipse/paho.mqtt.golang"
	qmq "github.com/rqure/qmq/src"
)

type MqttConsumer struct {
	key    string
	conn   *MqttConnection
	readCh chan qmq.Consumable
}

func NewMqttConsumer(key string, conn *MqttConnection) qmq.Consumer {
	c := &MqttConsumer{
		conn:   conn,
		readCh: make(chan qmq.Consumable),
		key:    key,
	}

	c.Register()

	return c
}

func (c *MqttConsumer) Register() {
	c.conn.Subscribe(c.key, 2, func(_ mqtt.Client, m mqtt.Message) {
		c.readCh <- NewMqttConsumable(m)
	})
}

func (c *MqttConsumer) Pop() chan qmq.Consumable {
	return c.readCh
}

func (c *MqttConsumer) Close() {
	c.conn.Unsubscribe(c.key)
	close(c.readCh)
}
