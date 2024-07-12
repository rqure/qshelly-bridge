package main

import (
	qdb "github.com/rqure/qdb/src"
	"github.com/rqure/qmqttgateway/devices"
)

type DeviceCommandHandlerSignals struct {
	Publish qdb.Signal
}

// Used to process device commands
type DeviceCommandHandler struct {
	db       qdb.IDatabase
	isLeader bool
	tokens   []string
	Signals  DeviceCommandHandlerSignals
}

func NewDeviceCommandHandler(db qdb.IDatabase) *DeviceCommandHandler {
	return &DeviceCommandHandler{
		db: db,
	}
}

func (h *DeviceCommandHandler) Reinitialize() {
	for _, token := range h.tokens {
		h.db.Unnotify(token)
	}

	h.tokens = []string{}

	for _, model := range devices.GetAllModels() {
		configs := devices.MakeMqttDevice(model).GetNotificationConfig()
		for _, config := range configs {
			h.tokens = append(h.tokens, h.db.Notify(config, h.ProcessNotification))
		}
	}
}

func (h *DeviceCommandHandler) OnSchemaUpdated() {
	if !h.isLeader {
		return
	}

	// In case more devices are configured, we should reinitialize to capture
	// notifications for any new devices
	h.Reinitialize()
}

func (h *DeviceCommandHandler) OnBecameLeader() {
	h.isLeader = true

	h.Reinitialize()
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
	if !h.isLeader {
		return
	}

	entity := qdb.NewEntity(h.db, notification.Current.Id)
	device := devices.MakeMqttDevice(entity.GetType())

	if device == nil {
		qdb.Error("[DeviceCommandHandler::ProcessNotification] Could not find device for model: %s", entity.GetType())
		return
	}

	device.ProcessNotification(notification, &h.Signals.Publish)
}
