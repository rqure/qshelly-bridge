package devices

import (
	mqtt "github.com/eclipse/paho.mqtt.golang"
	qdb "github.com/rqure/qdb/src"
)

type IMqttDevice interface {
	ProcessMessage(message mqtt.Message, db qdb.IDatabase)
	ProcessNotification(notification *qdb.DatabaseNotification)
	RegisterNotification(config *qdb.DatabaseNotificationConfig)
}

func MakeMqttDevice(model string) IMqttDevice {
	switch model {
	case "Aqara_LLKZMK12LM":
		return &Aqara_LLKZMK12LM{}
	case "Aqara_MCCGQ11LM":
		return &Aqara_MCCGQ11LM{}
	}

	return nil
}
