package main

import "os"

type MqttConnectionDetailsProvider interface {
	Address() string
}

type DefaultMqttConnectionDetailsProvider struct{}

func NewDefaultMqttConnectionDetailsProvider() MqttConnectionDetailsProvider {
	return &DefaultMqttConnectionDetailsProvider{}
}

func (d *DefaultMqttConnectionDetailsProvider) Address() string {
	mqttAddr := os.Getenv("MQTT_ADDR")
	if mqttAddr == "" {
		mqttAddr = "tcp://mosquitto:1883"
	}

	return mqttAddr
}
