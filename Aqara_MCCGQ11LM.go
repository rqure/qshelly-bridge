package main

import (
	"encoding/json"

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
	Battery           int64 `json:"battery"`
	Contact           bool  `json:"contact"`
	DeviceTemperature int64 `json:"device_temperature"`
	Voltage           int64 `json:"voltage"`
	PowerOutageCount  int64 `json:"power_outage_count"`
	LinkQuality       int64 `json:"linkquality"`
	TriggerCount      int64 `json:"trigger_count"`
}

func (d *Aqara_MCCGQ11LM) GetModel() string {
	return "AqaraMCCGQ11LM"
}

func (d *Aqara_MCCGQ11LM) ProcessMessage(message mqtt.Message, entity qdb.IEntity) {
	err := json.Unmarshal(message.Payload(), d)
	if err != nil {
		qdb.Error("[Aqara_MCCGQ11LM::ProcessMessage] Error parsing message payload: %v", err)
		return
	}

	entity.GetField("Battery").PushInt(d.Battery)
	entity.GetField("Contact").PushBool(d.Contact)
	entity.GetField("DeviceTemperature").PushInt(d.DeviceTemperature)
	entity.GetField("Voltage").PushInt(d.Voltage)
	entity.GetField("PowerOutageCount").PushInt(d.PowerOutageCount)
	entity.GetField("LinkQuality").PushInt(d.LinkQuality)
	entity.GetField("TriggerCount").PushInt(d.TriggerCount)
}

func (d *Aqara_MCCGQ11LM) ProcessNotification(notification *qdb.DatabaseNotification, publish *qdb.Signal) {

}

func (d *Aqara_MCCGQ11LM) GetNotificationConfig() []*qdb.DatabaseNotificationConfig {
	return []*qdb.DatabaseNotificationConfig{}
}

func (d *Aqara_MCCGQ11LM) GetSubscriptionConfig(entity qdb.IEntity) []*MqttSubscriptionConfig {
	return []*MqttSubscriptionConfig{
		{
			Topic: entity.GetField("Topic").PullString(),
			Qos:   0,
		},
	}
}
