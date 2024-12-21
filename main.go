package main

import (
	"os"

	qdb "github.com/rqure/qdb/src"
	"github.com/rqure/qlib/pkg/app"
	"github.com/rqure/qlib/pkg/app/workers"
	"github.com/rqure/qlib/pkg/data/store"
)

func getDatabaseAddress() string {
	addr := os.Getenv("Q_ADDR")
	if addr == "" {
		addr = "ws://webgateway:20000/ws"
	}

	return addr
}

func main() {
	db := store.NewWeb(store.WebConfig{
		Address: getDatabaseAddress(),
	})

	storeWorker := workers.NewStore(db)
	leadershipWorker := workers.NewLeadership(db)
	mqttConnectionsHandler := NewMqttConnectionsHandler(db)
	schemaValidator := leadershipWorker.GetEntityFieldValidator()

	schemaValidator.RegisterEntityFields("Root", "SchemaUpdateTrigger")
	schemaValidator.RegisterEntityFields("MqttController")
	schemaValidator.RegisterEntityFields("MqttServer", "Address", "ConnectionStatus", "Enabled", "TotalSent", "TotalReceived", "TotalDropped", "TxMessage")

	storeWorker.Connected.Connect(leadershipWorker.OnStoreConnected)
	storeWorker.Disconnected.Connect(leadershipWorker.OnStoreDisconnected)
	storeWorker.SchemaUpdated.Connect(mqttConnectionsHandler.OnSchemaUpdated)
	leadershipWorker.BecameLeader().Connect(mqttConnectionsHandler.OnBecameLeader)
	leadershipWorker.LosingLeadership().Connect(mqttConnectionsHandler.OnLostLeadership)

	// Create a new application configuration
	config := qdb.ApplicationConfig{
		Name: "mqttgateway",
		Workers: []qdb.IWorker{
			storeWorker,
			leadershipWorker,
			mqttConnectionsHandler,
		},
	}

	app := app.NewApplication(config)

	app.Execute()
}
