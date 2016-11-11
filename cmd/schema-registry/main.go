package main

import (
	"context"
	"flag"
	stdlog "log"
	"net/http"
	"os"

	"github.com/boltdb/bolt"
	schemaregistry "github.com/chop-dbhi/schema-registry"
	"github.com/chop-dbhi/schema-registry/transport"
	"github.com/go-kit/kit/log"
)

func main() {
	stdlog.SetFlags(0)

	var (
		dbpath   string
		httpaddr string
		tlscert  string
		tlskey   string
	)

	flag.StringVar(&dbpath, "db", "schema-registry.db", "Path to database.")
	flag.StringVar(&httpaddr, "http", "127.0.0.1:8080", "Address for HTTP transport.")
	flag.StringVar(&tlscert, "tlscert", "", "Path to TLS certificate.")
	flag.StringVar(&tlskey, "tlskey", "", "Path to TLS key.")

	flag.Parse()

	db, err := bolt.Open(dbpath, 0600, nil)
	if err != nil {
		stdlog.Fatal(err)
	}
	defer db.Close()

	s := schemaregistry.NewService(db)

	// Register compilers.
	s.Register("json-schema", schemaregistry.JSONSchemaCompiler)
	s.Register("avro", schemaregistry.AvroCompiler)

	cxt := context.Background()
	logger := log.NewJSONLogger(os.Stderr)

	// HTTP transport.
	ht := transport.NewHTTP(cxt, s, logger)
	if tlscert == "" {
		stdlog.Fatal(http.ListenAndServe(httpaddr, ht))
	} else {
		stdlog.Fatal(http.ListenAndServeTLS(httpaddr, tlscert, tlskey, ht))
	}
}
