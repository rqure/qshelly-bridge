package main

import qdb "github.com/rqure/qdb/src"

// Used to process device commands
type DeviceCommandHandler struct {
	db       qdb.IDatabase
	isLeader bool
}

func NewDeviceSynchronizer(db qdb.IDatabase) *DeviceCommandHandler {
	return &DeviceCommandHandler{
		db: db,
	}
}

func (h *DeviceCommandHandler) OnSchemaUpdated() {

}

func (h *DeviceCommandHandler) OnBecameLeader() {
	h.isLeader = true
}

func (h *DeviceCommandHandler) OnLostLeadership() {
	h.isLeader = false
}

func (h *DeviceCommandHandler) Init() {

}

func (h *DeviceCommandHandler) Deinit() {

}

func (h *DeviceCommandHandler) DoWork() {

}

func (h *DeviceCommandHandler) ProcessNotification(notification *qdb.DatabaseNotification) {

}
