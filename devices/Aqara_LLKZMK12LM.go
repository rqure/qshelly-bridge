package devices

import (
	"encoding/json"

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
    {
    "consumption": 8.300000190734863,
    "current": 0,
    "device_temperature": 36,
    "energy": 8.3,
    "interlock": "OFF",
    "led_disabled_night": false,
    "linkquality": 54,
    "mode": "power",
    "operation_mode_l1": "control_relay",
    "operation_mode_l2": "control_relay",
    "power": 0,
    "power_on_behavior": "previous",
    "power_outage_count": 13,
    "pulse_length": 200,
    "state": "OFF",
    "state_l1": "OFF",
    "state_l2": "OFF",
    "switch_type": "toggle",
    "voltage": 114.2
    }
*/
type Aqara_LLKZMK12LM struct {
	Consumption       float64 `json:"consumption"`
	Current           int64   `json:"current"`
	DeviceTemperature int64   `json:"device_temperature"`
	Energy            float64 `json:"energy"`
	Interlock         string  `json:"interlock"`
	LedDisabledNight  bool    `json:"led_disabled_night"`
	LinkQuality       int64   `json:"linkquality"`
	Mode              string  `json:"mode"`
	OperationModeL1   string  `json:"operation_mode_l1"`
	OperationModeL2   string  `json:"operation_mode_l2"`
	Power             int64   `json:"power"`
	PowerOnBehavior   string  `json:"power_on_behavior"`
	PowerOutageCount  int64   `json:"power_outage_count"`
	PulseLength       int64   `json:"pulse_length"`
	State             string  `json:"state"`
	StateL1           string  `json:"state_l1"`
	StateL2           string  `json:"state_l2"`
	SwitchType        string  `json:"switch_type"`
	Voltage           float64 `json:"voltage"`
}

func (d *Aqara_LLKZMK12LM) GetModel() string {
	return "AqaraLLKZMK12LM"
}

func (d *Aqara_LLKZMK12LM) ProcessMessage(message mqtt.Message, entity qdb.IEntity) {
	err := json.Unmarshal(message.Payload(), d)
	if err != nil {
		qdb.Error("[Aqara_LLKZMK12LM::ProcessMessage] Error parsing message payload: %v", err)
		return
	}

	entity.GetField("Consumption").PushFloat(d.Consumption)
	entity.GetField("Current").PushInt(d.Current)
	entity.GetField("DeviceTemperature").PushInt(d.DeviceTemperature)
	entity.GetField("Energy").PushFloat(d.Energy)
	entity.GetField("Interlock").PushString(d.Interlock)
	entity.GetField("LedDisabledNight").PushBool(d.LedDisabledNight)
	entity.GetField("LinkQuality").PushInt(d.LinkQuality)
	entity.GetField("Mode").PushString(d.Mode)
	entity.GetField("OperationModeL1").PushString(d.OperationModeL1)
	entity.GetField("OperationModeL2").PushString(d.OperationModeL2)
	entity.GetField("Power").PushInt(d.Power)
	entity.GetField("PowerOnBehavior").PushString(d.PowerOnBehavior)
	entity.GetField("PowerOutageCount").PushInt(d.PowerOutageCount)
	entity.GetField("PulseLength").PushInt(d.PulseLength)
	entity.GetField("State").PushString(d.State)
	entity.GetField("StateL1").PushString(d.StateL1)
	entity.GetField("StateL2").PushString(d.StateL2)
	entity.GetField("SwitchType").PushString(d.SwitchType)
	entity.GetField("Voltage").PushFloat(d.Voltage)
}

func (d *Aqara_LLKZMK12LM) ProcessNotification(notification *qdb.DatabaseNotification, publish *qdb.Signal) {
	if len(notification.Context) < 2 {
		qdb.Error("[Aqara_LLKZMK12LM::ProcessNotification] Missing notification context: %v", notification.Context)
		return
	}

	addr := &qdb.String{}
	topic := &qdb.String{}
	qos := uint8(0)
	retained := false

	err := notification.Context[0].Value.UnmarshalTo(addr)
	if err != nil {
		qdb.Error("[Aqara_LLKZMK12LM::ProcessNotification] Error parsing notification context: %v", err)
		return
	}

	err = notification.Context[1].Value.UnmarshalTo(topic)
	if err != nil {
		qdb.Error("[Aqara_LLKZMK12LM::ProcessNotification] Error parsing notification context: %v", err)
		return
	}

	switch notification.Current.Name {
	case "GetTrigger":
		payload, err := json.Marshal(map[string]interface{}{
			"state": "",
		})
		if err != nil {
			qdb.Error("[Aqara_LLKZMK12LM::ProcessNotification] Error marshalling payload: %v", err)
			return
		}
		publish.Emit(addr.Raw, topic.Raw+"/get", qos, retained, payload)
	case "StateOnTrigger":
		payload, err := json.Marshal(map[string]interface{}{
			"state_l1": "ON",
		})
		if err != nil {
			qdb.Error("[Aqara_LLKZMK12LM::ProcessNotification] Error marshalling payload: %v", err)
			return
		}
		publish.Emit(addr.Raw, topic.Raw+"/set", qos, retained, payload)
	case "StateOffTrigger":
		payload, err := json.Marshal(map[string]interface{}{
			"state_l1": "OFF",
		})
		if err != nil {
			qdb.Error("[Aqara_LLKZMK12LM::ProcessNotification] Error marshalling payload: %v", err)
			return
		}
		publish.Emit(addr.Raw, topic.Raw+"/set", qos, retained, payload)
	}
}

func (d *Aqara_LLKZMK12LM) GetNotificationConfig() []*qdb.DatabaseNotificationConfig {
	return []*qdb.DatabaseNotificationConfig{
		{
			Type:  d.GetModel(),
			Field: "StateOnTrigger",
			ContextFields: []string{
				"Server->Address",
				"Topic",
			},
		},
		{
			Type:  d.GetModel(),
			Field: "StateOffTrigger",
			ContextFields: []string{
				"Server->Address",
				"Topic",
			},
		},
		{
			Type:  d.GetModel(),
			Field: "GetTrigger",
			ContextFields: []string{
				"Server->Address",
				"Topic",
			},
		},
	}
}

func (d *Aqara_LLKZMK12LM) GetSubscriptionConfig(entity qdb.IEntity) []*MqttSubscriptionConfig {
	return []*MqttSubscriptionConfig{
		{
			Topic: entity.GetField("Topic").PullString(),
			Qos:   0,
		},
	}
}
