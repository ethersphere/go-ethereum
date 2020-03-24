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

package localstore

import (
	"encoding/binary"
	"errors"
	"fmt"

	"github.com/ethersphere/swarm/chunk"
	"github.com/ethersphere/swarm/log"
	"github.com/ethersphere/swarm/shed"
	"github.com/syndtr/goleveldb/leveldb"
)

var errMissingCurrentSchema = errors.New("could not find current db schema")
var errMissingTargetSchema = errors.New("could not find target db schema")

// BreakingMigrationError is returned from migration functions that require
// manual migration steps.
type BreakingMigrationError struct {
	Manual string
}

// NewBreakingMigrationError returns a new BreakingMigrationError
// with instructions for manual operations.
func NewBreakingMigrationError(manual string) *BreakingMigrationError {
	return &BreakingMigrationError{
		Manual: manual,
	}
}

func (e *BreakingMigrationError) Error() string {
	return "breaking migration"
}

// Migrate checks the schema name in storage dir and compares it
// with the expected schema name to construct a series of data migrations
// if they are required.
func (db *DB) Migrate() (err error) {
	schemaName, err := db.schemaName.Get()
	if err != nil {
		return err
	}
	if schemaName == "" {
		return nil
	}
	// execute possible migrations
	return db.migrate(schemaName)
}

type migration struct {
	name     string             // name of the schema
	fn       func(db *DB) error // the migration function that needs to be performed in order to get to the current schema name
	breaking bool
}

// schemaMigrations contains an ordered list of the database schemes, that is
// in order to run data migrations in the correct sequence
var schemaMigrations = []migration{
	{name: dbSchemaSanctuary, fn: func(db *DB) error { return nil }},
	{name: dbSchemaDiwali, fn: migrateSanctuary},
	{name: dbSchemaForky, fn: migrateDiwali, breaking: true},
}

func (db *DB) migrate(schemaName string) error {
	migrations, err := getMigrations(schemaName, dbSchemaCurrent, schemaMigrations)
	if err != nil {
		return fmt.Errorf("get migrations for current schema %s: %w", schemaName, err)
	}

	// no migrations to run
	if migrations == nil {
		return nil
	}

	log.Info("need to run data migrations on localstore", "numMigrations", len(migrations), "schemaName", schemaName)
	for i := 0; i < len(migrations); i++ {
		err := migrations[i].fn(db)
		if err != nil {
			return err
		}
		err = db.schemaName.Put(migrations[i].name) // put the name of the current schema
		if err != nil {
			return err
		}
		schemaName, err = db.schemaName.Get()
		if err != nil {
			return err
		}
		log.Info("successfully ran migration", "migrationId", i, "currentSchema", schemaName)
	}
	return nil
}

// migrationFn is a function that takes a localstore.DB and
// returns an error if a migration has failed
type migrationFn func(db *DB) error

// getMigrations returns an ordered list of migrations that need be executed
// with no errors in order to bring the localstore to the most up-to-date
// schema definition
func getMigrations(currentSchema, targetSchema string, allSchemeMigrations []migration) (migrations []migration, err error) {
	foundCurrent := false
	foundTarget := false
	if currentSchema == dbSchemaCurrent {
		return nil, nil
	}
	for i, v := range allSchemeMigrations {
		if v.name == targetSchema {
			foundTarget = true
		}
		if v.name == currentSchema {
			if foundCurrent {
				return nil, errors.New("found schema name for the second time when looking for migrations")
			}
			foundCurrent = true
			log.Info("found current localstore schema", "currentSchema", currentSchema, "migrateTo", dbSchemaCurrent, "total migrations", len(allSchemeMigrations)-i)
			continue // current schema migration should not be executed (already has been when schema was migrated to)
		}
		if foundCurrent {
			if v.breaking {
				// discard all migrations before a breaking one
				migrations = []migration{v}
			} else {
				migrations = append(migrations, v)
			}
		}
		if foundTarget {
			break
		}
	}
	if !foundCurrent {
		return nil, errMissingCurrentSchema
	}
	if !foundTarget {
		return nil, errMissingTargetSchema
	}
	return migrations, nil
}

// this function migrates Sanctuary schema to the Diwali schema
func migrateSanctuary(db *DB) error {
	// just rename the pull index
	renamed, err := db.shed.RenameIndex("PO|BinID->Hash", "PO|BinID->Hash|Tag")
	if err != nil {
		return err
	}
	if !renamed {
		return errors.New("pull index was not successfully renamed")
	}

	if db.tags == nil {
		return errors.New("had an error accessing the tags object")
	}

	batch := new(leveldb.Batch)
	db.batchMu.Lock()
	defer db.batchMu.Unlock()

	// since pullIndex points to the Tag value, we should eliminate possible
	// pushIndex leak due to items that were used by previous pull sync tag
	// increment logic. we need to build the index first since db object is
	// still not initialised at this stage
	db.pushIndex, err = db.shed.NewIndex("StoreTimestamp|Hash->Tags", shed.IndexFuncs{
		EncodeKey: func(fields shed.Item) (key []byte, err error) {
			key = make([]byte, 40)
			binary.BigEndian.PutUint64(key[:8], uint64(fields.StoreTimestamp))
			copy(key[8:], fields.Address[:])
			return key, nil
		},
		DecodeKey: func(key []byte) (e shed.Item, err error) {
			e.Address = key[8:]
			e.StoreTimestamp = int64(binary.BigEndian.Uint64(key[:8]))
			return e, nil
		},
		EncodeValue: func(fields shed.Item) (value []byte, err error) {
			tag := make([]byte, 4)
			binary.BigEndian.PutUint32(tag, fields.Tag)
			return tag, nil
		},
		DecodeValue: func(keyItem shed.Item, value []byte) (e shed.Item, err error) {
			if value != nil {
				e.Tag = binary.BigEndian.Uint32(value)
			}
			return e, nil
		},
	})

	err = db.pushIndex.Iterate(func(item shed.Item) (stop bool, err error) {
		tag, err := db.tags.Get(item.Tag)
		if err != nil {
			if err == chunk.TagNotFoundErr {
				return false, nil
			}
			return true, err
		}

		// anonymous tags should no longer appear in pushIndex
		if tag != nil && tag.Anonymous {
			db.pushIndex.DeleteInBatch(batch, item)
		}
		return false, nil
	}, nil)
	if err != nil {
		return err
	}

	return db.shed.WriteBatch(batch)
}

func migrateDiwali(db *DB) error {
	return NewBreakingMigrationError(fmt.Sprintf(`
Swarm chunk storage layer has changed.

You can choose either to manually migrate the data in your local store to the new data store or to discard the data altogether.

Preserving data requires additional storage roughly the size of the data directory and may take longer time depending on storage performance.

To continue by discarding data, just remove %[1]s directory and start the swarm binary again.

To preserve data:
  - export data
    swarm db export %[1]s data.tar %[2]x
  - remove data directory %[1]s
  - import data
    swarm db import %[1]s data.tar %[2]x
  - start the swarm
`, db.path, db.baseKey))
}
