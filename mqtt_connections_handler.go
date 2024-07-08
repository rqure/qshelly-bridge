package main

import qdb "github.com/rqure/qdb/src"

type MqttConnectionsHandler struct {
	db       qdb.IDatabase
	isLeader bool
}

func NewMqttConnectionsHandler(db qdb.IDatabase) *MqttConnectionsHandler {
	return &MqttConnectionsHandler{
		db: db,
	}
}

func (h *MqttConnectionsHandler) OnSchemaUpdated() {

}

func (h *MqttConnectionsHandler) OnBecameLeader() {
	h.isLeader = true
}

func (h *MqttConnectionsHandler) OnLostLeadership() {
	h.isLeader = false
}

func (h *MqttConnectionsHandler) Init() {

}

func (h *MqttConnectionsHandler) Deinit() {

}

func (h *MqttConnectionsHandler) DoWork() {

}

func (h *MqttConnectionsHandler) ProcessNotifications() {

}
