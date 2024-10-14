package main

import (
	"os"

	qdb "github.com/rqure/qdb/src"
)

func getDatabaseAddress() string {
	addr := os.Getenv("QDB_ADDR")
	if addr == "" {
		addr = "redis:6379"
	}

	return addr
}

func main() {
	db := qdb.NewRedisDatabase(qdb.RedisDatabaseConfig{
		Address: getDatabaseAddress(),
	})

	dbWorker := qdb.NewDatabaseWorker(db)
	leaderElectionWorker := qdb.NewLeaderElectionWorker(db)
	deviceCommandHandler := NewDeviceCommandHandler(db)
	mqttConnectionsHandler := NewMqttConnectionsHandler(db)
	schemaValidator := qdb.NewSchemaValidator(db)

	schemaValidator.AddEntity("Root", "SchemaUpdateTrigger")
	schemaValidator.AddEntity("MqttController")
	schemaValidator.AddEntity("MqttServer", "Address", "ConnectionStatus", "Enabled", "TotalSent", "TotalReceived", "TotalDropped")

	dbWorker.Signals.SchemaUpdated.Connect(qdb.Slot(schemaValidator.ValidationRequired))
	dbWorker.Signals.Connected.Connect(qdb.Slot(schemaValidator.ValidationRequired))
	leaderElectionWorker.AddAvailabilityCriteria(func() bool {
		return dbWorker.IsConnected() && schemaValidator.IsValid()
	})

	dbWorker.Signals.Connected.Connect(qdb.Slot(leaderElectionWorker.OnDatabaseConnected))
	dbWorker.Signals.Disconnected.Connect(qdb.Slot(leaderElectionWorker.OnDatabaseDisconnected))
	dbWorker.Signals.SchemaUpdated.Connect(qdb.Slot(deviceCommandHandler.OnSchemaUpdated))
	dbWorker.Signals.SchemaUpdated.Connect(qdb.Slot(mqttConnectionsHandler.OnSchemaUpdated))
	leaderElectionWorker.Signals.BecameLeader.Connect(qdb.Slot(deviceCommandHandler.OnBecameLeader))
	leaderElectionWorker.Signals.LosingLeadership.Connect(qdb.Slot(deviceCommandHandler.OnLostLeadership))
	leaderElectionWorker.Signals.BecameLeader.Connect(qdb.Slot(mqttConnectionsHandler.OnBecameLeader))
	leaderElectionWorker.Signals.LosingLeadership.Connect(qdb.Slot(mqttConnectionsHandler.OnLostLeadership))
	deviceCommandHandler.Signals.Publish.Connect(qdb.SlotWithArgs(mqttConnectionsHandler.OnPublish))

	// Create a new application configuration
	config := qdb.ApplicationConfig{
		Name: "mqttgateway",
		Workers: []qdb.IWorker{
			dbWorker,
			leaderElectionWorker,
			deviceCommandHandler,
			mqttConnectionsHandler,
		},
	}

	// Create a new application
	app := qdb.NewApplication(config)

	// Execute the application
	app.Execute()
}
