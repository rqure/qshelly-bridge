package devices

import (
	mqtt "github.com/eclipse/paho.mqtt.golang"
	qdb "github.com/rqure/qdb/src"
)

type Aqara_LLKZMK12LM struct {
}

func (d *Aqara_LLKZMK12LM) ProcessMessage(message mqtt.Message, db qdb.IDatabase) {
}

func (d *Aqara_LLKZMK12LM) ProcessNotification(notification *qdb.DatabaseNotification) {
}

func (d *Aqara_LLKZMK12LM) RegisterNotification(config *qdb.DatabaseNotificationConfig) {
}
