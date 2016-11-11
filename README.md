# Schema Registry

Schema registry is a service for managing schema for shared data types. The driving use case are validating data records in ETL-related workflows.

## Install

Download from the [releases page](https://github.com/chop-dbhi/schema-registry/releases)

## Run

```
schema-registry [-db registry.db]
                [-http 127.0.0.1:8080]
                [-tlscert tls.cert] [-tlskey tls.key]
```

## Usage

At a minimum a schema must include a unique id, type, and the schema definition. Currently the registry supports [JSON Schema](http://json-schema.org) schema and [Apache Avro](http://avro.apache.org/docs/current/).

### HTTP API

Register a new schema.

```
curl -XPOST localhost:8080/schema -d '
{
  "id": "test",
  "type": "json-schema",
  "def": {
    "type": "object",
    "properties": {
      "first_name": {
        "type": "string"
      },
      "last_name": {
        "type": "string"
      }
    }
  }
}
'
```

Get the current version of a schema.

```
curl localhost:8080/schema/test
```

Update a schema definition.

```
curl -XPUT localhost:8080/schema/test -d '
{
  "type": "object",
  "properties": {
    "first_name": {
      "type": "string"
    },
    "last_name": {
      "type": "string"
    },
    "dob": {
      "type": "string",
      "format": "date-time"
    }
  }
}
'
```

Validate a value against the schema. The response contains an array of errors if any are present.

```
curl -XPOST localhost:8080/schema/test -d '
{
  "first_name": "John",
  "last_name": "Doe"
}
'
```

Get a list of schema IDs.

```
curl localhost:8080/schema
```
