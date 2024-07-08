package main

import (
	"time"

	mqtt "github.com/eclipse/paho.mqtt.golang"
	qdb "github.com/rqure/qdb/src"
)

type MqttConnectionsHandler struct {
	db           qdb.IDatabase
	hasInit      bool
	isLeader     bool
	addrToClient map[string]mqtt.Client

	lastCheckConnectionTime time.Time
	checkConnectionInterval time.Duration
}

func NewMqttConnectionsHandler(db qdb.IDatabase) *MqttConnectionsHandler {
	return &MqttConnectionsHandler{
		db:                      db,
		checkConnectionInterval: 5 * time.Second,
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
			h.addrToClient[addr] = mqtt.NewClient(opts)
		}

		// connect to all servers and subscribe to all topics
		for _, client := range h.addrToClient {
			if !client.IsConnected() {
				client.Connect()
			}
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

	h.processConnectionStatuses()
	h.processIncomingMessages()
}

func (h *MqttConnectionsHandler) ProcessNotification(notification *qdb.DatabaseNotification) {

}

func (h *MqttConnectionsHandler) processConnectionStatuses() {
	if time.Since(h.lastCheckConnectionTime) < h.checkConnectionInterval {
		return
	}
	h.lastCheckConnectionTime = time.Now()

	// check connection status to all servers and store it in the database
	for addr, client := range h.addrToClient {
		connectionStatus := qdb.ConnectionState_UNSPECIFIED
		if !client.IsConnected() {
			connectionStatus = qdb.ConnectionState_DISCONNECTED
		} else {
			connectionStatus = qdb.ConnectionState_CONNECTED
		}

		servers := qdb.NewEntityFinder(h.db).Find(qdb.SearchCriteria{
			EntityType: "MqttServer",
			Conditions: []qdb.FieldConditionEval{
				qdb.NewStringCondition().Where("Address").IsEqualTo(&qdb.String{Raw: addr}),
			},
		})

		for _, server := range servers {
			server.GetField("ConnectionStatus").PushValue(&qdb.ConnectionState{Raw: connectionStatus})
		}
	}
}

func (h *MqttConnectionsHandler) processIncomingMessages() {

}
