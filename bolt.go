package schemaregistry

import (
	"encoding/binary"
	"encoding/json"
	"io"
	"sort"
	"time"

	"github.com/boltdb/bolt"
)

var (
	schemaBucket = []byte("schema")
)

func itob(v uint) []byte {
	b := make([]byte, binary.MaxVarintLen32)
	n := binary.PutUvarint(b, uint64(v))
	return b[:n]
}

func btoi(b []byte) uint {
	i, _ := binary.Uvarint(b)
	return uint(i)
}

type service struct {
	db    *bolt.DB
	types map[string]Compiler
}

func (s *service) Register(n string, c Compiler) {
	s.types[n] = c
}

func (s *service) Types() []string {
	var a []string
	for k := range s.types {
		a = append(a, k)
	}
	sort.Strings(a)
	return a
}

func (s *service) List() ([]string, error) {
	var ids []string

	err := s.db.View(func(tx *bolt.Tx) error {
		b := tx.Bucket(schemaBucket)
		if b == nil {
			return nil
		}

		return b.ForEach(func(k, _ []byte) error {
			ids = append(ids, string(k))
			return nil
		})
	})

	if err != nil {
		return nil, err
	}

	return ids, nil
}

func (s *service) Get(id string) (*Schema, error) {
	var x Schema

	err := s.db.View(func(tx *bolt.Tx) error {
		b := tx.Bucket(schemaBucket)
		if b == nil {
			return nil
		}

		// Bucket for the schema id.
		b = b.Bucket([]byte(id))
		if b == nil {
			return nil
		}

		// Versions are sorted.
		_, v := b.Cursor().Last()
		return json.Unmarshal(v, &x)
	})

	if err != nil {
		return nil, err
	}

	if x.ID == "" {
		return nil, ErrSchemaDoesNotExist
	}

	return &x, nil
}

func (s *service) GetVersion(id string, ver uint) (*Schema, error) {
	var x Schema

	err := s.db.View(func(tx *bolt.Tx) error {
		b := tx.Bucket(schemaBucket)
		if b == nil {
			return nil
		}

		// Bucket for the schema id.
		b = b.Bucket([]byte(id))
		if b == nil {
			return nil
		}

		// Get specific version.
		v := b.Get(itob(ver))
		if v == nil {
			return nil
		}

		return json.Unmarshal(v, &x)
	})

	if err != nil {
		return nil, err
	}

	if x.ID == "" {
		return nil, ErrSchemaDoesNotExist
	}

	return &x, nil

}

func (s *service) Create(id, typ string, def map[string]interface{}) (*Schema, error) {
	if id == "" {
		return nil, ErrSchemaIdRequired
	}

	if typ == "" {
		return nil, ErrSchemaTypeRequired
	}
	// Ensure a compiler exists.
	c, ok := s.types[typ]
	if !ok {
		return nil, ErrSchemaTypeUnknown
	}

	// Check if the schema definition compiles.
	_, err := c.Compile(def)
	if err != nil {
		return nil, err
	}

	x := &Schema{
		ID:      id,
		Type:    typ,
		Time:    time.Now().UTC(),
		Version: 1,
		Def:     def,
	}

	// Serialize for persistence.
	xb, err := json.Marshal(x)
	if err != nil {
		panic(err)
	}

	err = s.db.Update(func(tx *bolt.Tx) error {
		b, err := tx.CreateBucketIfNotExists(schemaBucket)
		if err != nil {
			return err
		}

		if b.Bucket([]byte(id)) != nil {
			return ErrSchemaExists
		}

		b, err = b.CreateBucket([]byte(id))
		if err != nil {
			return err
		}

		return b.Put(itob(x.Version), xb)
	})

	if err != nil {
		return nil, err
	}

	return x, nil
}

func (s *service) Update(id string, def map[string]interface{}) (*Schema, error) {
	if id == "" {
		return nil, ErrSchemaIdRequired
	}

	x := &Schema{}

	err := s.db.Update(func(tx *bolt.Tx) error {
		b := tx.Bucket(schemaBucket)
		if b == nil {
			return ErrSchemaDoesNotExist
		}

		if b = b.Bucket([]byte(id)); b == nil {
			return ErrSchemaDoesNotExist
		}

		_, v := b.Cursor().Last()
		if err := json.Unmarshal(v, x); err != nil {
			return err
		}

		// Check compilation.
		_, err := s.types[x.Type].Compile(def)
		if err != nil {
			return err
		}

		x.Time = time.Now().UTC()
		x.Version++
		x.Def = def

		// Serialize for persistence.
		xb, err := json.Marshal(x)
		if err != nil {
			return err
		}

		return b.Put(itob(x.Version), xb)
	})

	if err != nil {
		return nil, err
	}

	return x, nil
}

func (s *service) Delete(id string) error {
	return s.db.Update(func(tx *bolt.Tx) error {
		b := tx.Bucket(schemaBucket)
		if b == nil {
			return ErrSchemaDoesNotExist
		}

		if b.Bucket([]byte(id)) == nil {
			return ErrSchemaDoesNotExist
		}

		return b.DeleteBucket([]byte(id))
	})
}

func (s *service) Validate(id string, val io.Reader) ([]error, error) {
	var x Schema

	err := s.db.View(func(tx *bolt.Tx) error {
		b := tx.Bucket(schemaBucket)
		if b == nil {
			return ErrSchemaDoesNotExist
		}

		if b = b.Bucket([]byte(id)); b == nil {
			return ErrSchemaDoesNotExist
		}

		_, v := b.Cursor().Last()
		return json.Unmarshal(v, &x)
	})

	if err != nil {
		return nil, err
	}

	// Check compilation.
	c, err := s.types[x.Type].Compile(x.Def)
	if err != nil {
		return nil, err
	}

	return c.Validate(val)
}

func NewService(db *bolt.DB) Service {
	return &service{
		db:    db,
		types: make(map[string]Compiler),
	}
}
