package main

import qmq "github.com/rqure/qmq/src"

type ConnectionProviderFactory struct{}

func (f *ConnectionProviderFactory) Create() qmq.ConnectionProvider {
	connectionProvider := qmq.NewDefaultConnectionProvider()
	connectionProvider.Set("redis", qmq.NewRedisConnection(&qmq.DefaultRedisConnectionDetailsProvider{}))
	connectionProvider.Set("mqtt", NewMqttConnection(MqttConnectionConfig{
		MqttConnectionDetailsProvider: NewDefaultMqttConnectionDetailsProvider(),
	}))
	return connectionProvider
}
