package devices

import (
	mqtt "github.com/eclipse/paho.mqtt.golang"
	qdb "github.com/rqure/qdb/src"
)

/*
*
  - Model	MCCGQ11LM
  - Vendor	Aqara
  - Description	Door and window sensor
  - Exposes	battery, contact, device_temperature, voltage,
    power_outage_count, trigger_count, linkquality
*/
type Aqara_MCCGQ11LM struct {
}

func (d *Aqara_MCCGQ11LM) ProcessMessage(message mqtt.Message, db qdb.IDatabase) {
}

func (d *Aqara_MCCGQ11LM) ProcessNotification(notification *qdb.DatabaseNotification, publish *qdb.Signal) {
}

func (d *Aqara_MCCGQ11LM) RegisterNotification(config *qdb.DatabaseNotificationConfig) {
}
