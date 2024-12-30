package main

import (
	"context"
	"encoding/json"

	mqtt "github.com/eclipse/paho.mqtt.golang"
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
		store:        store,
		addrToClient: make(map[string]mqtt.Client),
		events:       make(chan IMqttEvent, 1024),
		tokens:       []data.NotificationToken{},
	}
}

func (h *MqttConnectionsHandler) Reinitialize(ctx context.Context) {
	for _, token := range h.tokens {
		token.Unbind(ctx)
	}

	h.tokens = []data.NotificationToken{}

	h.tokens = append(h.tokens, h.store.Notify(
		ctx,
		notification.NewConfig().
			SetEntityType("Root").
			SetFieldName("SchemaUpdateTrigger"),
		notification.NewCallback(h.OnSchemaUpdated)))

	h.tokens = append(h.tokens, h.store.Notify(
		ctx,
		notification.NewConfig().
			SetEntityType("MqttServer").
			SetFieldName("Address").
			SetNotifyOnChange(true).
			SetContextFields(
				"Enabled",
			),
		notification.NewCallback(h.onServerAddressChanged)))

	h.tokens = append(h.tokens, h.store.Notify(
		ctx,
		notification.NewConfig().
			SetEntityType("MqttServer").
			SetFieldName("Enabled").
			SetNotifyOnChange(true).
			SetContextFields(
				"Address",
			),
		notification.NewCallback(h.onServerEnableChanged)))

	h.tokens = append(h.tokens, h.store.Notify(
		ctx,
		notification.NewConfig().
			SetEntityType("MqttServer").
			SetFieldName("TxMessage").
			SetNotifyOnChange(false),
		notification.NewCallback(h.onPublishMessage)))

	servers := query.New(h.store).
		Select("Address", "Enabled").
		From("MqttServer").
		Execute(ctx)

	for _, server := range servers {
		addr := server.GetField("Address").GetString()
		if _, ok := h.addrToClient[addr]; ok {
			if server.GetField("Enabled").GetBool() && !h.addrToClient[addr].IsConnected() && h.isLeader {
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

		if server.GetField("Enabled").GetBool() {
			if h.isLeader {
				h.addrToClient[addr].Connect()
			}
		}
	}
}

func (h *MqttConnectionsHandler) OnSchemaUpdated(ctx context.Context, n data.Notification) {
	if !h.isLeader {
		return
	}

	// In case more servers are configured, we should reinitialize to capture
	// notifications for any new servers
	h.Reinitialize(ctx)
}

func (h *MqttConnectionsHandler) OnBecameLeader(ctx context.Context) {
	h.isLeader = true

	h.Reinitialize(ctx)
}

func (h *MqttConnectionsHandler) OnLostLeadership(ctx context.Context) {
	h.isLeader = false

	for _, client := range h.addrToClient {
		if client.IsConnected() {
			client.Disconnect(0)
		}

		multi := binding.NewMulti(h.store)
		servers := query.New(multi).
			Select().
			From("MqttServer").
			Where("Enabled").Equals(true).
			Execute(ctx)

		for _, server := range servers {
			server.GetField("IsConnected").WriteBool(ctx, false, data.WriteChanges)
		}

		multi.Commit(ctx)
	}
}

func (h *MqttConnectionsHandler) onPublishMessage(ctx context.Context, n data.Notification) {
	if !h.isLeader {
		return
	}

	msgAsJson := n.GetCurrent().GetValue().GetString()
	msg := MqttMessageJson{}
	if err := json.Unmarshal([]byte(msgAsJson), &msg); err != nil {
		log.Error("Error unmarshalling message: %v", err)
		return
	}

	addr := n.GetContext(0).GetValue().GetString()

	h.DoPublish(ctx, addr, msg.Topic, byte(msg.Qos), msg.Retained, msg.Payload)
}

func (h *MqttConnectionsHandler) DoPublish(ctx context.Context, args ...interface{}) {
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
		servers := query.New(h.store).
			Select("TotalDropped", "TotalSent").
			From("MqttServer").
			Execute(ctx)

		if !client.IsConnected() {
			log.Warn("Client not connected for address: %s", addr)

			for _, server := range servers {
				totalDropped := server.GetField("TotalDropped")
				totalDropped.WriteInt(ctx, totalDropped.GetInt()+1)
			}

			return
		}

		client.Publish(topic, qos, retained, payload)

		for _, server := range servers {
			totalSent := server.GetField("TotalSent")
			totalSent.WriteInt(ctx, totalSent.GetInt()+1)
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

func (h *MqttConnectionsHandler) DoWork(ctx context.Context) {
	for {
		select {
		case event := <-h.events:
			switch event.GetEventType() {
			case ConnectionEstablished:
				h.onMqttServerConnected(ctx, event.GetAddress())
			case ConnectionLost:
				h.onMqttServerDisconnected(ctx, event.GetAddress(), event.GetEventData().(error))
			case MessageReceived:
				h.onMqttMessageReceived(ctx, event.GetAddress(), event.GetEventData().(mqtt.Message))
			}
		default:
			return
		}
	}
}

func (h *MqttConnectionsHandler) onMqttServerConnected(ctx context.Context, addr string) {
	if !h.isLeader {
		return
	}

	log.Info("Connected to server: %s", addr)
	if _, ok := h.addrToClient[addr]; !ok {
		log.Warn("No client found for address: %s", addr)
		return
	}

	client := h.addrToClient[addr]
	servers := query.New(h.store).
		Select().
		From("MqttServer").
		Where("Address").Equals(addr).
		Execute(ctx)

	for _, server := range servers {
		server.GetField("IsConnected").WriteBool(ctx, true, data.WriteChanges)

		for _, childId := range server.GetChildrenIds() {
			device := binding.NewEntity(ctx, h.store, childId)
			device.DoMulti(ctx, func(device data.EntityBinding) {
				device.GetField("Topic").ReadString(ctx)
				device.GetField("Qos").ReadInt(ctx)
			})

			topic := device.GetField("Topic").GetString()
			qos := byte(device.GetField("Qos").GetInt())

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

func (h *MqttConnectionsHandler) onMqttServerDisconnected(ctx context.Context, addr string, err error) {
	if !h.isLeader {
		return
	}

	log.Warn("Disconnected from server: %s (%v)", addr, err)
	if _, ok := h.addrToClient[addr]; !ok {
		log.Warn("No client found for address: %s", addr)
		return
	}

	multi := binding.NewMulti(h.store)
	servers := query.New(multi).
		Select().
		From("MqttServer").
		Where("Address").Equals(addr).
		Execute(ctx)

	for _, server := range servers {
		server.GetField("IsConnected").WriteBool(ctx, false, data.WriteChanges)
	}

	multi.Commit(ctx)
}

func (h *MqttConnectionsHandler) onMqttMessageReceived(ctx context.Context, addr string, msg mqtt.Message) {
	if !h.isLeader {
		return
	}

	log.Trace("Received message from server: %s (%v)", addr, msg)
	if _, ok := h.addrToClient[addr]; !ok {
		log.Warn("No client found for address: %s", addr)
		return
	}

	servers := query.New(h.store).
		Select("TotalReceived").
		From("MqttServer").
		Where("Address").Equals(addr).
		Execute(ctx)

	for _, server := range servers {
		totalReceived := server.GetField("TotalReceived")
		totalReceived.WriteInt(ctx, totalReceived.GetInt()+1)

		for _, childId := range server.GetChildrenIds() {
			device := binding.NewEntity(ctx, h.store, childId)
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
					device.GetField("RxMessageFn").WriteString(ctx, string(j))
				} else {
					log.Error("Error marshalling message: %v", err)
				}

				break
			}
		}
	}
}

func (h *MqttConnectionsHandler) onServerAddressChanged(ctx context.Context, n data.Notification) {
	if !h.isLeader {
		return
	}

	prev := n.GetPrevious().GetValue().GetString()
	curr := n.GetCurrent().GetValue().GetString()
	enabled := n.GetContext(0).GetValue().GetBool()

	if client, ok := h.addrToClient[prev]; ok {
		if client.IsConnected() {
			client.Disconnect(0)
		}

		delete(h.addrToClient, prev)
	}

	if _, ok := h.addrToClient[curr]; ok {
		return
	}

	opts := mqtt.NewClientOptions()
	opts.AddBroker(curr)
	opts.AutoReconnect = true
	opts.OnConnect = func(client mqtt.Client) {
		h.events <- &MqttEvent{
			EventType: ConnectionEstablished,
			Address:   curr,
		}
	}
	opts.OnConnectionLost = func(client mqtt.Client, err error) {
		h.events <- &MqttEvent{
			EventType: ConnectionLost,
			Address:   curr,
			EventData: err,
		}
	}
	h.addrToClient[curr] = mqtt.NewClient(opts)

	if enabled {
		h.addrToClient[curr].Connect()
	}
}

func (h *MqttConnectionsHandler) onServerEnableChanged(ctx context.Context, n data.Notification) {
	if !h.isLeader {
		return
	}

	addr := n.GetContext(0).GetValue().GetString()
	enabled := n.GetCurrent().GetValue().GetBool()

	if client, ok := h.addrToClient[addr]; ok {
		if enabled {
			if !client.IsConnected() {
				client.Connect()
			}
		} else {
			if client.IsConnected() {
				client.Disconnect(0)
			}
		}
	} else {
		log.Error("No client found for address: %s", addr)
	}
}
