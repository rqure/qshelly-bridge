package devices

import (
	mqtt "github.com/eclipse/paho.mqtt.golang"
	qdb "github.com/rqure/qdb/src"
)

/*
*
  - Model	LLKZMK12LM
  - Vendor	Aqara
  - Description	Dual relay module T2
  - Exposes	switch (state), power, current, energy, voltage, device_temperature,
    switch_type, power_on_behavior, operation_mode, interlock, mode,
    pulse_length, action, linkquality
*/
type Aqara_LLKZMK12LM struct {
}

func (d *Aqara_LLKZMK12LM) GetModel() string {
	return "AqaraLLKZMK12LM"
}

func (d *Aqara_LLKZMK12LM) ProcessMessage(message mqtt.Message, db qdb.IDatabase) {
}

func (d *Aqara_LLKZMK12LM) ProcessNotification(notification *qdb.DatabaseNotification, publish *qdb.Signal) {
}

func (d *Aqara_LLKZMK12LM) GetNotificationConfig() *qdb.DatabaseNotificationConfig {
	return nil
}

func (d *Aqara_LLKZMK12LM) GetSubscriptionConfig(entity qdb.IEntity) []*MqttSubscriptionConfig {
	return []*MqttSubscriptionConfig{
		{
			Topic: entity.GetField("Topic").PullValue(&qdb.String{}).(*qdb.String).Raw,
			Qos:   0,
		},
	}
}
