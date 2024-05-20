package main

import qmq "github.com/rqure/qmq/src"

type ConsumerFactory struct{}

func (a *ConsumerFactory) Create(key string, components qmq.EngineComponentProvider) qmq.Consumer {
	if key == "mosquitto" {
		mqttConnection := components.WithConnectionProvider().Get("mqtt").(*MqttConnection)
		return NewMqttConsumer("#", mqttConnection)
	}

	redisConnection := components.WithConnectionProvider().Get("redis").(*qmq.RedisConnection)
	transformerKey := "consumer:" + key

	return qmq.NewRedisConsumer(redisConnection, &qmq.RedisConsumerConfig{
		Topic:        key,
		Transformers: components.WithTransformerProvider().Get(transformerKey),
		AckOriginal:  true,
	})
}
