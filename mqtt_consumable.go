package main

import (
	mqtt "github.com/eclipse/paho.mqtt.golang"
	qmq "github.com/rqure/qmq/src"
)

type MqttConsumable struct {
	m *qmq.MqttMessage
}

func NewMqttConsumable(m mqtt.Message) qmq.Consumable {
	msg := &qmq.MqttMessage{
		Topic:     m.Topic(),
		Payload:   m.Payload(),
		Id:        uint32(m.MessageID()),
		Qos:       int32(m.Qos()),
		Retained:  m.Retained(),
		Duplicate: m.Duplicate(),
	}

	return &MqttConsumable{
		m: msg,
	}
}

func (c *MqttConsumable) Data() interface{} {
	return c.m
}

func (c *MqttConsumable) Ack() {}

func (c *MqttConsumable) Nack() {}
