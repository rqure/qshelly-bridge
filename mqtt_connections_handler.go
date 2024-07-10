package main

import (
	mqtt "github.com/eclipse/paho.mqtt.golang"
	qdb "github.com/rqure/qdb/src"
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
			addr := server.GetField("Address").PullValue(&qdb.String{}).(*qdb.String).Raw
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
	if !h.isLeader {
		return
	}

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
	qdb.Info("[MqttConnectionsHandler::onMqttServerConnected] Connected to server: %s", addr)

	servers := qdb.NewEntityFinder(h.db).Find(qdb.SearchCriteria{
		EntityType: "MqttServer",
		Conditions: []qdb.FieldConditionEval{
			qdb.NewStringCondition().Where("Address").IsEqualTo(&qdb.String{Raw: addr}),
		},
	})

	for _, server := range servers {
		server.GetField("ConnectionStatus").PushValue(&qdb.ConnectionState{Raw: qdb.ConnectionState_CONNECTED})
	}
}

func (h *MqttConnectionsHandler) onMqttServerDisconnected(addr string, err error) {
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
	qdb.Trace("[MqttConnectionsHandler::onMqttMessageReceived] Received message from server: %s (%v)", addr, msg)
}
