package main

import (
	mqtt "github.com/eclipse/paho.mqtt.golang"
	qmq "github.com/rqure/qmq/src"
)

type MqttConnectionConfig struct {
	MqttConnectionDetailsProvider MqttConnectionDetailsProvider
}

type MqttConnection struct {
	config MqttConnectionConfig
	client mqtt.Client
}

func NewMqttConnection(config MqttConnectionConfig) qmq.Connection {
	if config.MqttConnectionDetailsProvider == nil {
		config.MqttConnectionDetailsProvider = NewDefaultMqttConnectionDetailsProvider()
	}

	opt := mqtt.NewClientOptions()
	opt.AddBroker(config.MqttConnectionDetailsProvider.Address())

	return &MqttConnection{
		config: config,
		client: mqtt.NewClient(opt),
	}
}

func (m *MqttConnection) Connect() error {
	if token := m.client.Connect(); token.Wait() && token.Error() != nil {
		return token.Error()
	}

	return nil
}

func (m *MqttConnection) Disconnect() {
	m.client.Disconnect(0)
}

func (m *MqttConnection) Subscribe(topic string, qos byte, callback mqtt.MessageHandler) error {
	if token := m.client.Subscribe(topic, qos, callback); token.Wait() && token.Error() != nil {
		return token.Error()
	}

	return nil
}

func (m *MqttConnection) Unsubscribe(topic string) error {
	if token := m.client.Unsubscribe(topic); token.Wait() && token.Error() != nil {
		return token.Error()
	}

	return nil
}

func (m *MqttConnection) Publish(topic string, payload interface{}, qos byte, retained bool) error {
	token := m.client.Publish(topic, qos, retained, payload)
	token.Wait()

	return token.Error()
}
