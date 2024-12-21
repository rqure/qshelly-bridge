package main

import (
	"context"
	"encoding/json"

	mqtt "github.com/eclipse/paho.mqtt.golang"
	qdb "github.com/rqure/qdb/src"
	"github.com/rqure/qlib/pkg/app"
	"github.com/rqure/qlib/pkg/data"
	"github.com/rqure/qlib/pkg/data/binding"
	"github.com/rqure/qlib/pkg/data/notification"
	"github.com/rqure/qlib/pkg/data/query"
	"github.com/rqure/qlib/pkg/log"
)

type MqttMessageJson struct {
	Topic    string      `json:"topic"`
	Qos      int64       `json:"qos"`
	Retained bool        `json:"retained"`
	Payload  interface{} `json:"payload"`
}

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
	store        data.Store
	isLeader     bool
	addrToClient map[string]mqtt.Client
	events       chan IMqttEvent
	tokens       []data.NotificationToken
}

func NewMqttConnectionsHandler(store data.Store) *MqttConnectionsHandler {
	return &MqttConnectionsHandler{
		db:           db,
		addrToClient: make(map[string]mqtt.Client),
		events:       make(chan IMqttEvent, 1024),
		tokens:       []data.NotificationToken{},
	}
}

func (h *MqttConnectionsHandler) Reinitialize() {
	for _, token := range h.tokens {
		token.Unbind()
	}

	h.tokens = []data.NotificationToken{}

	h.tokens = append(h.tokens, h.db.Notify(&qdb.DatabaseNotificationConfig{
		Type:           "MqttServer",
		Field:          "Address",
		NotifyOnChange: true,
		ContextFields: []string{
			"Enabled",
		},
	}, notification.NewCallback(h.onServerAddressChanged)))

	h.tokens = append(h.tokens, h.db.Notify(&qdb.DatabaseNotificationConfig{
		Type:           "MqttServer",
		Field:          "Enabled",
		NotifyOnChange: true,
		ContextFields: []string{
			"Address",
		},
	}, notification.NewCallback(h.onServerEnableChanged)))

	h.tokens = append(h.tokens, h.db.Notify(&qdb.DatabaseNotificationConfig{
		Type:           "MqttServer",
		Field:          "TxMessage",
		NotifyOnChange: false,
		ContextFields: []string{
			"Address",
		},
	}, notification.NewCallback(h.onPublishMessage)))

	servers := query.New(h.db).Find(qdb.SearchCriteria{
		EntityType: "MqttServer",
		Conditions: []qdb.FieldConditionEval{},
	})

	for _, server := range servers {
		addr := server.GetField("Address").ReadString(ctx)
		if _, ok := h.addrToClient[addr]; ok {
			if server.GetField("Enabled").ReadBool() && !h.addrToClient[addr].IsConnected(ctx) && h.isLeader {
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

		if server.GetField("Enabled").ReadBool(ctx) {
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

func (h *MqttConnectionsHandler) OnBecameLeader(context.Context) {
	h.isLeader = true

	h.Reinitialize()
}

func (h *MqttConnectionsHandler) OnLostLeadership(context.Context) {
	h.isLeader = false

	for _, client := range h.addrToClient {
		if client.IsConnected() {
			client.Disconnect(0)
		}

		servers := query.New(h.db).Find(qdb.SearchCriteria{
			EntityType: "MqttServer",
			Conditions: []qdb.FieldConditionEval{
				qdb.NewBoolCondition().Where("Enabled").IsEqualTo(&qdb.Bool{Raw: true}),
			},
		})

		for _, server := range servers {
			server.GetField("ConnectionStatus").WriteValue(ctx, &qdb.ConnectionState{Raw: qdb.ConnectionState_DISCONNECTED})
		}
	}
}

func (h *MqttConnectionsHandler) onPublishMessage(ctx context.Context, notification data.Notification) {
	if !h.isLeader {
		return
	}

	msgAsJson := notification.GetCurrent().GetValue().GetString()
	msg := MqttMessageJson{}
	if err := json.Unmarshal([]byte(msgAsJson), &msg); err != nil {
		log.Error("Error unmarshalling message: %v", err)
		return
	}

	addr := notification.GetContext(0).GetValue().GetString()

	h.DoPublish(addr, msg.Topic, byte(msg.Qos), msg.Retained, msg.Payload)
}

func (h *MqttConnectionsHandler) DoPublish(args ...interface{}) {
	if !h.isLeader {
		return
	}

	addr := args[0].(string)
	topic := args[1].(string)
	qos := args[2].(byte)
	retained := args[3].(bool)
	payload := args[4]

	log.Trace("Publishing message to server: %s, topic: %s, qos: %d, retained: %v, payload: %v", addr, topic, qos, retained, payload)

	if client, ok := h.addrToClient[addr]; ok {
		servers := query.New(h.db).Find(qdb.SearchCriteria{
			EntityType: "MqttServer",
			Conditions: []qdb.FieldConditionEval{},
		})

		if !client.IsConnected() {
			log.Warn("Client not connected for address: %s", addr)

			for _, server := range servers {
				totalDropped := server.GetField("TotalDropped")
				totalDropped.WriteInt(totalDropped.ReadInt(ctx) + 1)
			}

			return
		}

		client.Publish(topic, qos, retained, payload)

		for _, server := range servers {
			totalSent := server.GetField("TotalSent")
			totalSent.WriteInt(ctx, totalSent.ReadInt(ctx)+1)
		}
	} else {
		log.Error("No client found for address: %s", addr)
	}
}

func (h *MqttConnectionsHandler) Init(context.Context, app.Handle) {
}

func (h *MqttConnectionsHandler) Deinit(context.Context) {
	// disconnect from all servers
	for _, client := range h.addrToClient {
		if client.IsConnected() {
			client.Disconnect(0)
		}
	}
}

func (h *MqttConnectionsHandler) DoWork(context.Context) {
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

	log.Info("Connected to server: %s", addr)
	if _, ok := h.addrToClient[addr]; !ok {
		log.Warn("No client found for address: %s", addr)
		return
	}

	client := h.addrToClient[addr]
	servers := query.New(h.db).Find(qdb.SearchCriteria{
		EntityType: "MqttServer",
		Conditions: []qdb.FieldConditionEval{
			qdb.NewStringCondition().Where("Address").IsEqualTo(&qdb.String{Raw: addr}),
		},
	})

	for _, server := range servers {
		server.GetField("ConnectionStatus").WriteValue(ctx, &qdb.ConnectionState{Raw: qdb.ConnectionState_CONNECTED})

		for _, childId := range server.GetChildren() {
			device := binding.NewEntity(ctx, h.db, childId.Raw)
			topic := device.GetField("Topic").ReadString(ctx)
			qos := byte(device.GetField("Qos").ReadInt(ctx))
			client.Subscribe(topic, qos, func(client mqtt.Client, msg mqtt.Message) {
				h.events <- &MqttEvent{
					EventType: MessageReceived,
					Address:   addr,
					EventData: msg,
				}
			})
		}
	}
}

func (h *MqttConnectionsHandler) onMqttServerDisconnected(addr string, err error) {
	if !h.isLeader {
		return
	}

	log.Warn("Disconnected from server: %s (%v)", addr, err)
	if _, ok := h.addrToClient[addr]; !ok {
		log.Warn("No client found for address: %s", addr)
		return
	}

	servers := query.New(h.db).Find(qdb.SearchCriteria{
		EntityType: "MqttServer",
		Conditions: []qdb.FieldConditionEval{
			qdb.NewStringCondition().Where("Address").IsEqualTo(&qdb.String{Raw: addr}),
		},
	})

	for _, server := range servers {
		server.GetField("ConnectionStatus").WriteValue(ctx, &qdb.ConnectionState{Raw: qdb.ConnectionState_DISCONNECTED})
	}
}

func (h *MqttConnectionsHandler) onMqttMessageReceived(addr string, msg mqtt.Message) {
	if !h.isLeader {
		return
	}

	log.Trace("Received message from server: %s (%v)", addr, msg)
	if _, ok := h.addrToClient[addr]; !ok {
		log.Warn("No client found for address: %s", addr)
		return
	}

	servers := query.New(h.db).Find(qdb.SearchCriteria{
		EntityType: "MqttServer",
		Conditions: []qdb.FieldConditionEval{
			qdb.NewStringCondition().Where("Address").IsEqualTo(&qdb.String{Raw: addr}),
		},
	})

	for _, server := range servers {
		totalReceived := server.GetField("TotalReceived")
		totalReceived.WriteInt(ctx, totalReceived.ReadInt(ctx)+1)

		for _, childId := range server.GetChildren() {
			device := binding.NewEntity(ctx, h.db, childId.Raw)
			topic := device.GetField("Topic").ReadString(ctx)

			if topic == msg.Topic() {
				msgAsJson := MqttMessageJson{
					Topic:    msg.Topic(),
					Qos:      int64(msg.Qos()),
					Retained: msg.Retained(),
				}

				msgAsJson.Payload = make(map[string]interface{})
				if err := json.Unmarshal(msg.Payload(), &msgAsJson.Payload); err != nil {
					msgAsJson.Payload = string(msg.Payload())
				}

				if j, err := json.Marshal(msgAsJson); err == nil {
					device.GetField("RxMessageFn").WriteString(string(ctx, j))
				} else {
					log.Error("Error marshalling message: %v", err)
				}

				break
			}
		}
	}
}

func (h *MqttConnectionsHandler) onServerAddressChanged(ctx context.Context, notification data.Notification) {
	if !h.isLeader {
		return
	}

	prev := qdb.ValueCast[*qdb.String](notification.Previous.Value)
	curr := qdb.ValueCast[*qdb.String](notification.GetCurrent().GetValue())
	enabled := qdb.ValueCast[*qdb.Bool](notification.GetContext(0).GetValue())

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

func (h *MqttConnectionsHandler) onServerEnableChanged(ctx context.Context, notification data.Notification) {
	if !h.isLeader {
		return
	}

	addr := qdb.ValueCast[*qdb.String](notification.GetContext(0).GetValue())
	enabled := qdb.ValueCast[*qdb.Bool](notification.GetCurrent().GetValue())

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
		log.Error("No client found for address: %s", addr.Raw)
	}
}
