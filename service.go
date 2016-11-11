package schemaregistry

import (
	"errors"
	"io"
	"time"
)

var (
	ErrSchemaExists       = errors.New("schema exists")
	ErrSchemaDoesNotExist = errors.New("schema does not exist")
	ErrSchemaTypeUnknown  = errors.New("unknown schema type")
	ErrSchemaIdRequired   = errors.New("schema id required")
	ErrSchemaTypeRequired = errors.New("schema type required")
)

// Compiler compiles a schema definition and returns a validator that
// can be used to validate messages.
type Compiler interface {
	// Compile compiles a schema and returns a validator for
	// the schema.
	Compile(def map[string]interface{}) (Validator, error)
}

// Validator implements a validation logic for a schema and validates
// whether an input value adheres to the schema.
type Validator interface {
	// Validates a value for the schema definition.
	Validate(val io.Reader) ([]error, error)
}

// Schema represents a generic schema definition.
type Schema struct {
	// Unique ID of the schema.
	ID string `json:"id"`

	// Schema definition type.
	Type string `json:"type"`

	// Time the schema changed.
	Time time.Time `json:"time"`

	// Monotonically increasing number.
	Version uint `json:"version"`

	// Definition of the schema.
	Def map[string]interface{} `json:"def"`
}

// Service is an interface that describes an RPC service for the
// schema registry.
type Service interface {
	// Register registers a compiler by schema type.
	Register(typ string, com Compiler)

	// Types returns a list of all schema types that are registered.
	Types() []string

	// List returns a list of schema ids.
	List() ([]string, error)

	// Get gets the latest version of a schema by id.
	Get(id string) (*Schema, error)

	// GetVersion gets a specific version of a schema.
	GetVersion(id string, ver uint) (*Schema, error)

	// Create creates a new schema definition.
	Create(id, typ string, def map[string]interface{}) (*Schema, error)

	// Update updates the schema definition.
	Update(id string, def map[string]interface{}) (*Schema, error)

	// Delete deletes a schema.
	Delete(id string) error

	// Validate validates value against the target schema.
	Validate(id string, val io.Reader) ([]error, error)
}
