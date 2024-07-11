package main

import (
	mqtt "github.com/eclipse/paho.mqtt.golang"
	qdb "github.com/rqure/qdb/src"
	"github.com/rqure/qmqttgateway/devices"
)

type MqttEventType int

const (
	ConnectionEstablished MqttEventType = iota
	ConnectionLost
	MessageReceived
)

type IMqttEvent interface {
	GetEventType() MqttEventType
	GetEventData() interface{}
	GetAddress() string
}

type MqttEvent struct {
	EventType MqttEventType
	EventData interface{}
	Address   string
}

func (e *MqttEvent) GetEventType() MqttEventType {
	return e.EventType
}

func (e *MqttEvent) GetEventData() interface{} {
	return e.EventData
}

func (e *MqttEvent) GetAddress() string {
	return e.Address
}

// Used to manage connections to MQTT servers
type MqttConnectionsHandler struct {
	db           qdb.IDatabase
	hasInit      bool
	isLeader     bool
	addrToClient map[string]mqtt.Client
	events       chan IMqttEvent
}

func NewMqttConnectionsHandler(db qdb.IDatabase) *MqttConnectionsHandler {
	return &MqttConnectionsHandler{
		db:           db,
		addrToClient: make(map[string]mqtt.Client),
		events:       make(chan IMqttEvent, 1024),
	}
}

func (h *MqttConnectionsHandler) OnBecameLeader() {
	h.isLeader = true

	if !h.hasInit {
		// get all mqtt servers from the database
		servers := qdb.NewEntityFinder(h.db).Find(qdb.SearchCriteria{
			EntityType: "MqttServer",
		})

		for _, server := range servers {
			addr := server.GetField("Address").PullString()
			opts := mqtt.NewClientOptions()
			opts.AddBroker(addr)
			opts.AutoReconnect = true
			opts.OnConnect = func(client mqtt.Client) {
				h.events <- &MqttEvent{
					EventType: ConnectionEstablished,
					Address:   addr,
				}
			}
			opts.OnConnectionLost = func(client mqtt.Client, err error) {
				h.events <- &MqttEvent{
					EventType: ConnectionLost,
					Address:   addr,
					EventData: err,
				}
			}
			h.addrToClient[addr] = mqtt.NewClient(opts)
			h.addrToClient[addr].Connect()
		}

		h.hasInit = true
	}
}

func (h *MqttConnectionsHandler) OnLostLeadership() {
	h.isLeader = false
}

func (h *MqttConnectionsHandler) OnPublish(args ...interface{}) {
	if !h.isLeader {
		return
	}

	addr := args[0].(string)
	topic := args[1].(string)
	qos := args[2].(byte)
	retained := args[3].(bool)
	payload := args[4]

	qdb.Trace("[MqttConnectionsHandler::OnPublish] Publishing message to server: %s, topic: %s, qos: %d, retained: %v, payload: %v", addr, topic, qos, retained, payload)

	if client, ok := h.addrToClient[addr]; ok {
		client.Publish(topic, qos, retained, payload)
	} else {
		qdb.Error("[MqttConnectionsHandler::OnPublish] No client found for address: %s", addr)
	}
}

func (h *MqttConnectionsHandler) Init() {
}

func (h *MqttConnectionsHandler) Deinit() {
	// disconnect from all servers
	for _, client := range h.addrToClient {
		if client.IsConnected() {
			client.Disconnect(0)
		}
	}
}

func (h *MqttConnectionsHandler) DoWork() {
	for {
		select {
		case event := <-h.events:
			switch event.GetEventType() {
			case ConnectionEstablished:
				h.onMqttServerConnected(event.GetAddress())
			case ConnectionLost:
				h.onMqttServerDisconnected(event.GetAddress(), event.GetEventData().(error))
			case MessageReceived:
				h.onMqttMessageReceived(event.GetAddress(), event.GetEventData().(mqtt.Message))
			}
		default:
			return
		}
	}
}

func (h *MqttConnectionsHandler) ProcessNotification(notification *qdb.DatabaseNotification) {

}

func (h *MqttConnectionsHandler) onMqttServerConnected(addr string) {
	if !h.isLeader {
		return
	}

	client := h.addrToClient[addr]
	qdb.Info("[MqttConnectionsHandler::onMqttServerConnected] Connected to server: %s", addr)

	servers := qdb.NewEntityFinder(h.db).Find(qdb.SearchCriteria{
		EntityType: "MqttServer",
		Conditions: []qdb.FieldConditionEval{
			qdb.NewStringCondition().Where("Address").IsEqualTo(&qdb.String{Raw: addr}),
		},
	})

	for _, server := range servers {
		server.GetField("ConnectionStatus").PushValue(&qdb.ConnectionState{Raw: qdb.ConnectionState_CONNECTED})

		for _, model := range devices.GetAllModels() {
			devs := qdb.NewEntityFinder(h.db).Find(qdb.SearchCriteria{
				EntityType: model,
				Conditions: []qdb.FieldConditionEval{
					qdb.NewStringCondition().Where("Server->Address").IsEqualTo(&qdb.String{Raw: addr}),
				},
			})

			for _, device := range devs {
				configs := devices.MakeMqttDevice(model).GetSubscriptionConfig(device)
				for _, config := range configs {
					client.Subscribe(config.Topic, config.Qos, func(client mqtt.Client, msg mqtt.Message) {
						h.events <- &MqttEvent{
							EventType: MessageReceived,
							Address:   addr,
							EventData: msg,
						}
					})
				}
			}
		}
	}
}

func (h *MqttConnectionsHandler) onMqttServerDisconnected(addr string, err error) {
	if !h.isLeader {
		return
	}

	qdb.Warn("[MqttConnectionsHandler::onMqttServerDisconnected] Disconnected from server: %s (%v)", addr, err)

	servers := qdb.NewEntityFinder(h.db).Find(qdb.SearchCriteria{
		EntityType: "MqttServer",
		Conditions: []qdb.FieldConditionEval{
			qdb.NewStringCondition().Where("Address").IsEqualTo(&qdb.String{Raw: addr}),
		},
	})

	for _, server := range servers {
		server.GetField("ConnectionStatus").PushValue(&qdb.ConnectionState{Raw: qdb.ConnectionState_DISCONNECTED})
	}
}

func (h *MqttConnectionsHandler) onMqttMessageReceived(addr string, msg mqtt.Message) {
	if !h.isLeader {
		return
	}

	qdb.Trace("[MqttConnectionsHandler::onMqttMessageReceived] Received message from server: %s (%v)", addr, msg)

	// Not the most performant algorithm but should work for now
	for _, model := range devices.GetAllModels() {
		entities := qdb.NewEntityFinder(h.db).Find(qdb.SearchCriteria{
			EntityType: model,
			Conditions: []qdb.FieldConditionEval{
				qdb.NewStringCondition().Where("Server->Address").IsEqualTo(&qdb.String{Raw: addr}),
			},
		})

		for _, entity := range entities {
			device := devices.MakeMqttDevice(model)
			configs := device.GetSubscriptionConfig(entity)
			for _, config := range configs {
				if config.Topic == msg.Topic() {
					device.ProcessMessage(msg, entity)
				}
			}
		}
	}
}
