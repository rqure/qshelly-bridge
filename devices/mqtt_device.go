package devices

import (
	mqtt "github.com/eclipse/paho.mqtt.golang"
	qdb "github.com/rqure/qdb/src"
)

type MqttSubscriptionConfig struct {
	Topic string
	Qos   byte
}

type IMqttDevice interface {
	GetModel() string
	ProcessMessage(message mqtt.Message, db qdb.IDatabase)
	ProcessNotification(notification *qdb.DatabaseNotification, publish *qdb.Signal)
	GetNotificationConfig() *qdb.DatabaseNotificationConfig
	GetSubscriptionConfig(entity qdb.IEntity) []*MqttSubscriptionConfig
}

func GetAllDevices() []IMqttDevice {
	devs := []IMqttDevice{
		&Aqara_LLKZMK12LM{},
		&Aqara_MCCGQ11LM{},
	}

	return devs
}

func GetAllModels() []string {
	devs := GetAllDevices()

	models := make([]string, len(devs))
	for i, dev := range devs {
		models[i] = dev.GetModel()
	}

	return models
}

func MakeMqttDevice(model string) IMqttDevice {
	devs := GetAllDevices()

	for _, dev := range devs {
		if dev.GetModel() == model {
			return dev
		}
	}

	return nil
}
