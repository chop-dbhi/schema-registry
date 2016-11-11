package schemaregistry

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"

	"github.com/xeipuuv/gojsonschema"
)

// JSONSchemaCompiler is the compiler for JSON Schema-based schema.
var JSONSchemaCompiler Compiler = &jsonschemaCompiler{}

type jsonschemaCompiler struct{}

func (c *jsonschemaCompiler) Compile(def map[string]interface{}) (Validator, error) {
	s, err := gojsonschema.NewSchema(gojsonschema.NewGoLoader(def))
	if err != nil {
		return nil, fmt.Errorf("json-schema compiler: %s", err)
	}

	return &jsonschemaValidator{s: s}, nil
}

type jsonschemaValidator struct {
	s *gojsonschema.Schema
}

func (v *jsonschemaValidator) Validate(val io.Reader) ([]error, error) {
	var m map[string]interface{}
	if err := json.NewDecoder(val).Decode(&m); err != nil {
		return nil, err
	}

	l := gojsonschema.NewGoLoader(m)
	result, err := v.s.Validate(l)
	if err != nil {
		return nil, err
	}

	if result.Valid() {
		return nil, nil
	}

	rerrs := result.Errors()
	errs := make([]error, len(rerrs))
	for i, _ := range errs {
		errs[i] = errors.New(fmt.Sprintf("%s", rerrs[i]))
	}

	return errs, nil
}
