package devices

import (
	mqtt "github.com/eclipse/paho.mqtt.golang"
	qdb "github.com/rqure/qdb/src"
)

type Aqara_MCCGQ11LM struct {
}

func (d *Aqara_MCCGQ11LM) ProcessMessage(message mqtt.Message, db qdb.IDatabase) {
}

func (d *Aqara_MCCGQ11LM) ProcessNotification(notification *qdb.DatabaseNotification) {
}

func (d *Aqara_MCCGQ11LM) RegisterNotification(config *qdb.DatabaseNotificationConfig) {
}
