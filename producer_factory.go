package main

import qmq "github.com/rqure/qmq/src"

type ProducerFactory struct{}

func (a *ProducerFactory) Create(key string, components qmq.EngineComponentProvider) qmq.Producer {
	if key == "mosquitto" {
		mqttConnection := components.WithConnectionProvider().Get("mqtt").(*MqttConnection)
		return NewMqttProducer(mqttConnection)
	}

	maxLength := 10
	redisConnection := components.WithConnectionProvider().Get("redis").(*qmq.RedisConnection)
	transformerKey := "producer:" + key
	return qmq.NewRedisProducer(key, redisConnection, int64(maxLength), components.WithTransformerProvider().Get(transformerKey))
}
