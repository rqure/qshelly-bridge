package main

import qmq "github.com/rqure/qmq/src"

type TransformerProviderFactory struct{}

func (t *TransformerProviderFactory) Create(components qmq.EngineComponentProvider) qmq.TransformerProvider {
	return NewMqttTransformerProvider(components.WithLogger())
}
