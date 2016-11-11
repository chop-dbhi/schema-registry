package schemaregistry

import (
	"encoding/json"
	"fmt"
	"io"

	"github.com/linkedin/goavro"
)

// AvroCompiler is the compiler for Apache Avro schema.
var AvroCompiler Compiler = &avroCompiler{}

type avroCompiler struct{}

func (c *avroCompiler) Compile(def map[string]interface{}) (Validator, error) {
	b, err := json.Marshal(def)
	if err != nil {
		return nil, err
	}

	s, err := goavro.NewCodec(string(b))
	if err != nil {
		return nil, fmt.Errorf("avro compiler: %s", err)
	}

	return &avroValidator{s: s}, nil
}

type avroValidator struct {
	s goavro.Codec
}

func (v *avroValidator) Validate(val io.Reader) ([]error, error) {
	_, err := v.s.Decode(val)
	if err != nil {
		return []error{err}, nil
	}
	return nil, nil
}
