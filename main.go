package main

import (
	"os"

	"github.com/rqure/qlib/pkg/app"
	"github.com/rqure/qlib/pkg/app/workers"
	"github.com/rqure/qlib/pkg/data/store"
)

func getStoreAddress() string {
	addr := os.Getenv("Q_ADDR")
	if addr == "" {
		addr = "ws://webgateway:20000/ws"
	}

	return addr
}

func main() {
	s := store.NewWeb(store.WebConfig{
		Address: getStoreAddress(),
	})

	storeWorker := workers.NewStore(s)
	leadershipWorker := workers.NewLeadership(s)
	mqttConnectionsHandler := NewMqttConnectionsHandler(s)
	schemaValidator := leadershipWorker.GetEntityFieldValidator()

	schemaValidator.RegisterEntityFields("Root", "SchemaUpdateTrigger")
	schemaValidator.RegisterEntityFields("MqttController")
	schemaValidator.RegisterEntityFields("MqttServer", "Address", "IsConnected", "Enabled", "TotalSent", "TotalReceived", "TotalDropped", "TxMessage")

	storeWorker.Connected.Connect(leadershipWorker.OnStoreConnected)
	storeWorker.Disconnected.Connect(leadershipWorker.OnStoreDisconnected)
	leadershipWorker.BecameLeader().Connect(mqttConnectionsHandler.OnBecameLeader)
	leadershipWorker.LosingLeadership().Connect(mqttConnectionsHandler.OnLostLeadership)

	a := app.NewApplication("mqttgateway")
	a.AddWorker(storeWorker)
	a.AddWorker(leadershipWorker)
	a.AddWorker(mqttConnectionsHandler)
	a.Execute()
}
