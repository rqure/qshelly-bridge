package main

import (
	qmq "github.com/rqure/qmq/src"
)

func main() {
	engine := qmq.NewDefaultEngine(qmq.DefaultEngineConfig{
		NameProvider:               &NameProvider{},
		ConnectionProviderFactory:  &ConnectionProviderFactory{},
		ConsumerFactory:            &ConsumerFactory{},
		ProducerFactory:            &ProducerFactory{},
		TransformerProviderFactory: &TransformerProviderFactory{},
		EngineProcessor:            &EngineProcessor{},
	})
	engine.Run()
}
