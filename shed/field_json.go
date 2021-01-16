// Copyright 2019 The Swarm Authors
// This file is part of the Swarm library.
//
// The Swarm library is free software: you can redistribute it and/or modify
// it under the terms of the GNU Lesser General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// The Swarm library is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Lesser General Public License for more details.
//
// You should have received a copy of the GNU Lesser General Public License
// along with the Swarm library. If not, see <http://www.gnu.org/licenses/>.

package shed

import (
	"encoding/json"

	"github.com/syndtr/goleveldb/leveldb"
)

// JSONField is a helper to store complex structure by
// encoding it in JSON format.
type JSONField struct {
	db  *DB
	key []byte
}

// NewJSONField returns a new JSONField.
// It validates its name and type against the database schema.
func (db *DB) NewJSONField(name string) (f JSONField, err error) {
	key, err := db.schemaFieldKey(name, "json")
	if err != nil {
		return f, err
	}
	return JSONField{
		db:  db,
		key: key,
	}, nil
}

// Get unmarshals data from the database to a provided val.
// If the data is not found leveldb.ErrNotFound is returned.
func (f JSONField) Get(val interface{}) (err error) {
	b, err := f.db.Get(f.key)
	if err != nil {
		return err
	}
	return json.Unmarshal(b, val)
}

// Put marshals provided val and saves it to the database.
func (f JSONField) Put(val interface{}) (err error) {
	b, err := json.Marshal(val)
	if err != nil {
		return err
	}
	return f.db.Put(f.key, b)
}

// PutInBatch marshals provided val and puts it into the batch.
func (f JSONField) PutInBatch(batch *leveldb.Batch, val interface{}) (err error) {
	b, err := json.Marshal(val)
	if err != nil {
		return err
	}
	batch.Put(f.key, b)
	return nil
}
