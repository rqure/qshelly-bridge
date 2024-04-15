package main

import (
	qmq "github.com/rqure/qmq/src"
)

type MqttProducer struct {
	conn *MqttConnection
}

func NewMqttProducer(conn *MqttConnection) qmq.Producer {
	return &MqttProducer{
		conn: conn,
	}
}

func (p *MqttProducer) Push(data interface{}) {
	if msg, ok := data.(*qmq.MqttMessage); ok {
		p.conn.Publish(msg.Topic, msg.Payload, 0, false)
	}
}

func (p *MqttProducer) Close() {}
