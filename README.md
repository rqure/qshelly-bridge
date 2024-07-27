# qmqttgateway

## Overview

This application serves as a MQTT gateway to the QDB, allowing edge devices to read, write, and notify on changes to the database.

![MQTT Gateway drawio](https://github.com/user-attachments/assets/917a1901-f6e5-42d5-ac37-fd35ba224a28)

## API

Below is the list of API endpoints. Payload is expected to be in JSON format.

### Who am I

This is a request expected to be sent by the edge device to help it identify its friendly name and database entity ID.

#### Request

Topic: qdb/whoami/request

Payload:

```json
{
  "mac": "<mac address>"
}
```

#### Response

Topic: qdb/whoami/response/<mac address>

Payload:

```json
{
  "name": "<friend name>",
  "entityId": "<uuid-4>"
}
```

### Database Read

This is a request that allows the edge device to read some data from the database.

#### Request

Topic: qdb/read/request

Payload:

```json
{
  "entityId": "<entity ID received from whoami request>",
  "correlationId": "<Id for correlating the response to a request. This is useful for differentiating responses from multiple simultaneous requests.>"
  "fields": [
    "Field1",
    "Indirect->Field2"
  ]
}
```

#### Response

Topic: qdb/read/response/<correlationId>

Payload:

```
{
  "entityId": "The entity ID of database entity being read from"
  "correlationId": "<Correlation ID provided in request>"
  "Field1": value1,
  "Indirect->Field2": value2
}
```

### Database Write

This is a request that allows the edge device to write some data to the database.

#### Request

Topic: qdb/write/request
