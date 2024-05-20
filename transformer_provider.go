package main

import (
	"strings"

	qmq "github.com/rqure/qmq/src"
)

type MqttTransformerProvider struct {
	transformers map[string][]qmq.Transformer
	logger       qmq.Logger
}

func NewMqttTransformerProvider(logger qmq.Logger) qmq.TransformerProvider {
	return &MqttTransformerProvider{
		transformers: make(map[string][]qmq.Transformer),
		logger:       logger,
	}
}

func (p *MqttTransformerProvider) Get(key string) []qmq.Transformer {
	if p.transformers[key] == nil {
		if strings.HasPrefix(key, "consumer") {
			return []qmq.Transformer{
				qmq.NewTracePopTransformer(p.logger),
				qmq.NewMessageToAnyTransformer(p.logger),
				qmq.NewAnyToMqttTransformer(p.logger),
			}
		} else if strings.HasPrefix(key, "producer") {
			return []qmq.Transformer{
				qmq.NewMqttToAnyTransformer(p.logger),
				qmq.NewAnyToMessageTransformer(p.logger, qmq.AnyToMessageTransformerConfig{
					SourceProvider: qmq.NewDefaultSourceProvider("qmq2mqtt"),
				}),
				qmq.NewTracePushTransformer(p.logger),
			}
		}

		return []qmq.Transformer{}
	}

	return p.transformers[key]
}

func (p *MqttTransformerProvider) Set(key string, transformers []qmq.Transformer) {
	p.transformers[key] = transformers
}
