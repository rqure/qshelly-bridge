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

func (d *Aqara_MCCGQ11LM) GetModel() string {
	return "AqaraMCCGQ11LM"
}

func (d *Aqara_MCCGQ11LM) ProcessMessage(message mqtt.Message, entity qdb.IEntity) {
	entity.GetField("Battery").PushValue(&qdb.Int{Raw: 100})
}

func (d *Aqara_MCCGQ11LM) ProcessNotification(notification *qdb.DatabaseNotification, publish *qdb.Signal) {
	// Read only device -- no notification expected
}

func (d *Aqara_MCCGQ11LM) GetNotificationConfig() []*qdb.DatabaseNotificationConfig {
	// Read only device -- no notification config necessary
	return []*qdb.DatabaseNotificationConfig{}
}

func (d *Aqara_MCCGQ11LM) GetSubscriptionConfig(entity qdb.IEntity) []*MqttSubscriptionConfig {
	return []*MqttSubscriptionConfig{
		{
			Topic: entity.GetField("Topic").PullValue(&qdb.String{}).(*qdb.String).Raw,
			Qos:   0,
		},
	}
}
