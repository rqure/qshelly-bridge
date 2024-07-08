package main

import qdb "github.com/rqure/qdb/src"

type DeviceSynchronizer struct {
	db       qdb.IDatabase
	isLeader bool
}

func NewDeviceSynchronizer(db qdb.IDatabase) *DeviceSynchronizer {
	return &DeviceSynchronizer{
		db: db,
	}
}

func (h *DeviceSynchronizer) OnSchemaUpdated() {

}

func (h *DeviceSynchronizer) OnBecameLeader() {
	h.isLeader = true
}

func (h *DeviceSynchronizer) OnLostLeadership() {
	h.isLeader = false
}

func (h *DeviceSynchronizer) Init() {

}

func (h *DeviceSynchronizer) Deinit() {

}

func (h *DeviceSynchronizer) DoWork() {

}

func (h *DeviceSynchronizer) ProcessNotifications() {

}
