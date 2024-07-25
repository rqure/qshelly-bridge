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

// Used to manage connections to MQTT servers
type MqttConnectionsHandler struct {
	db           qdb.IDatabase
	isLeader     bool
	addrToClient map[string]mqtt.Client
	events       chan IMqttEvent
	tokens       []qdb.INotificationToken
}

func NewMqttConnectionsHandler(db qdb.IDatabase) *MqttConnectionsHandler {
	return &MqttConnectionsHandler{
		db:           db,
		addrToClient: make(map[string]mqtt.Client),
		events:       make(chan IMqttEvent, 1024),
		tokens:       []qdb.INotificationToken{},
	}
}

func (h *MqttConnectionsHandler) Reinitialize() {
	for _, token := range h.tokens {
		token.Unbind()
	}

	h.tokens = []qdb.INotificationToken{}

	h.tokens = append(h.tokens, h.db.Notify(&qdb.DatabaseNotificationConfig{
		Type:           "MqttServer",
		Field:          "Address",
		NotifyOnChange: true,
		ContextFields: []string{
			"Enabled",
		},
	}, qdb.NewNotificationCallback(h.onServerAddressChanged)))

	h.tokens = append(h.tokens, h.db.Notify(&qdb.DatabaseNotificationConfig{
		Type:           "MqttServer",
		Field:          "Enabled",
		NotifyOnChange: true,
		ContextFields: []string{
			"Address",
		},
	}, qdb.NewNotificationCallback(h.onServerEnableChanged)))

	servers := qdb.NewEntityFinder(h.db).Find(qdb.SearchCriteria{
		EntityType: "MqttServer",
		Conditions: []qdb.FieldConditionEval{},
	})

	for _, server := range servers {
		addr := server.GetField("Address").PullString()
		if _, ok := h.addrToClient[addr]; ok {
			if server.GetField("Enabled").PullBool() && !h.addrToClient[addr].IsConnected() && h.isLeader {
				h.addrToClient[addr].Connect()
			}
			continue
		}

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

		if server.GetField("Enabled").PullBool() {
			if h.isLeader {
				h.addrToClient[addr].Connect()
			}
		}
	}
}

func (h *MqttConnectionsHandler) OnSchemaUpdated() {
	if !h.isLeader {
		return
	}

	// In case more servers are configured, we should reinitialize to capture
	// notifications for any new servers
	h.Reinitialize()
}

func (h *MqttConnectionsHandler) OnBecameLeader() {
	h.isLeader = true

	h.Reinitialize()
}

func (h *MqttConnectionsHandler) OnLostLeadership() {
	h.isLeader = false

	for _, client := range h.addrToClient {
		if client.IsConnected() {
			client.Disconnect(0)
		}

		servers := qdb.NewEntityFinder(h.db).Find(qdb.SearchCriteria{
			EntityType: "MqttServer",
			Conditions: []qdb.FieldConditionEval{
				qdb.NewBoolCondition().Where("Enabled").IsEqualTo(&qdb.Bool{Raw: true}),
			},
		})

		for _, server := range servers {
			server.GetField("ConnectionStatus").PushValue(&qdb.ConnectionState{Raw: qdb.ConnectionState_DISCONNECTED})
		}
	}
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
		servers := qdb.NewEntityFinder(h.db).Find(qdb.SearchCriteria{
			EntityType: "MqttServer",
			Conditions: []qdb.FieldConditionEval{},
		})

		if !client.IsConnected() {
			qdb.Warn("[MqttConnectionsHandler::OnPublish] Client not connected for address: %s", addr)

			for _, server := range servers {
				totalDropped := server.GetField("TotalDropped")
				totalDropped.PushInt(totalDropped.PullInt() + 1)
			}

			return
		}

		client.Publish(topic, qos, retained, payload)

		for _, server := range servers {
			totalSent := server.GetField("TotalSent")
			totalSent.PushInt(totalSent.PullInt() + 1)
		}
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

func (h *MqttConnectionsHandler) onMqttServerConnected(addr string) {
	if !h.isLeader {
		return
	}

	qdb.Info("[MqttConnectionsHandler::onMqttServerConnected] Connected to server: %s", addr)
	if _, ok := h.addrToClient[addr]; !ok {
		qdb.Warn("[MqttConnectionsHandler::onMqttServerConnected] No client found for address: %s", addr)
		return
	}

	client := h.addrToClient[addr]
	servers := qdb.NewEntityFinder(h.db).Find(qdb.SearchCriteria{
		EntityType: "MqttServer",
		Conditions: []qdb.FieldConditionEval{
			qdb.NewStringCondition().Where("Address").IsEqualTo(&qdb.String{Raw: addr}),
		},
	})

	for _, server := range servers {
		server.GetField("ConnectionStatus").PushValue(&qdb.ConnectionState{Raw: qdb.ConnectionState_CONNECTED})

		for _, model := range GetAllModels() {
			devs := qdb.NewEntityFinder(h.db).Find(qdb.SearchCriteria{
				EntityType: model,
				Conditions: []qdb.FieldConditionEval{
					qdb.NewStringCondition().Where("Server->Address").IsEqualTo(&qdb.String{Raw: addr}),
				},
			})

			for _, device := range devs {
				configs := MakeMqttDevice(model).GetSubscriptionConfig(device)
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
	if _, ok := h.addrToClient[addr]; !ok {
		qdb.Warn("[MqttConnectionsHandler::onMqttServerDisconnected] No client found for address: %s", addr)
		return
	}

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
	if _, ok := h.addrToClient[addr]; !ok {
		qdb.Warn("[MqttConnectionsHandler::onMqttMessageReceived] No client found for address: %s", addr)
		return
	}

	servers := qdb.NewEntityFinder(h.db).Find(qdb.SearchCriteria{
		EntityType: "MqttServer",
		Conditions: []qdb.FieldConditionEval{
			qdb.NewStringCondition().Where("Address").IsEqualTo(&qdb.String{Raw: addr}),
		},
	})

	for _, server := range servers {
		totalReceived := server.GetField("TotalReceived")
		totalReceived.PushInt(totalReceived.PullInt() + 1)
	}

	// Not the most performant algorithm but should work for now
	for _, model := range GetAllModels() {
		entities := qdb.NewEntityFinder(h.db).Find(qdb.SearchCriteria{
			EntityType: model,
			Conditions: []qdb.FieldConditionEval{
				qdb.NewStringCondition().Where("Server->Address").IsEqualTo(&qdb.String{Raw: addr}),
			},
		})

		for _, entity := range entities {
			device := MakeMqttDevice(model)
			configs := device.GetSubscriptionConfig(entity)
			for _, config := range configs {
				if config.Topic == msg.Topic() {
					device.ProcessMessage(msg, entity)
				}
			}
		}
	}
}

func (h *MqttConnectionsHandler) onServerAddressChanged(notification *qdb.DatabaseNotification) {
	if !h.isLeader {
		return
	}

	prev := qdb.ValueCast[*qdb.String](notification.Previous.Value)
	curr := qdb.ValueCast[*qdb.String](notification.Current.Value)
	enabled := qdb.ValueCast[*qdb.Bool](notification.Context[0].Value)

	if client, ok := h.addrToClient[prev.Raw]; ok {
		if client.IsConnected() {
			client.Disconnect(0)
		}

		delete(h.addrToClient, prev.Raw)
	}

	if _, ok := h.addrToClient[curr.Raw]; ok {
		return
	}

	opts := mqtt.NewClientOptions()
	opts.AddBroker(curr.Raw)
	opts.AutoReconnect = true
	opts.OnConnect = func(client mqtt.Client) {
		h.events <- &MqttEvent{
			EventType: ConnectionEstablished,
			Address:   curr.Raw,
		}
	}
	opts.OnConnectionLost = func(client mqtt.Client, err error) {
		h.events <- &MqttEvent{
			EventType: ConnectionLost,
			Address:   curr.Raw,
			EventData: err,
		}
	}
	h.addrToClient[curr.Raw] = mqtt.NewClient(opts)

	if enabled.Raw {
		h.addrToClient[curr.Raw].Connect()
	}
}

func (h *MqttConnectionsHandler) onServerEnableChanged(notification *qdb.DatabaseNotification) {
	if !h.isLeader {
		return
	}

	addr := qdb.ValueCast[*qdb.String](notification.Context[0].Value)
	enabled := qdb.ValueCast[*qdb.Bool](notification.Current.Value)

	if client, ok := h.addrToClient[addr.Raw]; ok {
		if enabled.Raw {
			if !client.IsConnected() {
				client.Connect()
			}
		} else {
			if client.IsConnected() {
				client.Disconnect(0)
			}
		}
	} else {
		qdb.Error("[MqttConnectionsHandler::onServerEnableChanged] No client found for address: %s", addr.Raw)
	}
}
