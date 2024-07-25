package main

import (
	"encoding/json"

	mqtt "github.com/eclipse/paho.mqtt.golang"
	qdb "github.com/rqure/qdb/src"
)

type Ikea_LED2005R5 struct {
	LinkQuality      int64  `json:"linkquality"`
	State            string `json:"state"`
	Brightness       int64  `json:"brightness"`
	ColorTemp        int64  `json:"color_temp"`
	ColorTempStartup int64  `json:"color_temp_startup"`
	ColorMode        string `json:"color_mode"`
	PowerOnBehavior  string `json:"power_on_behavior"`
}

func (d *Ikea_LED2005R5) GetModel() string {
	return "IkeaLED2005R5"
}

func (d *Ikea_LED2005R5) ProcessMessage(message mqtt.Message, entity qdb.IEntity) {
	err := json.Unmarshal(message.Payload(), d)
	if err != nil {
		qdb.Error("[Ikea_LED2005R5::ProcessMessage] Error parsing message payload: %v", err)
		return
	}

	entity.GetField("LinkQuality").PushInt(d.LinkQuality)
	entity.GetField("State").PushString(d.State)
	entity.GetField("Brightness").PushInt(d.Brightness)
	entity.GetField("ColorTemp").PushInt(d.ColorTemp)
	entity.GetField("ColorTempStartup").PushInt(d.ColorTempStartup)
	entity.GetField("ColorMode").PushString(d.ColorMode)
	entity.GetField("PowerOnBehavior").PushString(d.PowerOnBehavior)
}

func (d *Ikea_LED2005R5) ProcessNotification(notification *qdb.DatabaseNotification, publish *qdb.Signal) {

}

func (d *Ikea_LED2005R5) GetNotificationConfig() []*qdb.DatabaseNotificationConfig {
	return []*qdb.DatabaseNotificationConfig{}
}

func (d *Ikea_LED2005R5) GetSubscriptionConfig(entity qdb.IEntity) []*MqttSubscriptionConfig {
	return []*MqttSubscriptionConfig{
		{
			Topic: entity.GetField("Topic").PullString(),
			Qos:   0,
		},
	}
}
