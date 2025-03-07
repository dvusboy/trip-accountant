// Package trip implements the data model for managing trip expenses
// the key purpose is to compute the settlement of the expenses by the
// participants.
//
// This unit runs some unit tests against the user model.

package trip

import (
	"context"
	"testing"

	_ "github.com/mattn/go-sqlite3"
)

const (
	tuserCreate = `CREATE TABLE IF NOT EXISTS tuser (
user_id INTEGER CONSTRAINT user_pkey PRIMARY KEY AUTOINCREMENT,
email VARCHAR(256) NOT NULL UNIQUE,
verified BOOLEAN DEFAULT FALSE)`
	tuserDrop = "DROP TABLE IF EXISTS tuser"

	alice   = "alice@test.com"
	bob     = "bob@test.com"
	charlie = "charlie@test.com"
	david   = "david@test.com"
	elise   = "elise@test.com"
	fred    = "fred@test.com"
	greg    = "greg@test.com"
	henry   = "henry@test.com"
)

func TestLoadOrCreateUser(t *testing.T) {
	ctx := context.Background()
	usr, err := LoadOrCreateUser(ctx, db, "ALICE@test.com")
	if err != nil {
		t.Errorf("LoadOrCreateUser(alice) failed: %v", err)
	}
	if usr.ID == 0 {
		t.Error("ID field should not be 0")
	}
	if usr.ID != 1 {
		t.Error("Alice should have ID 1")
	}
	if usr.Email != alice {
		t.Error("Email should be normalized")
	}
}

func TestSave(t *testing.T) {
	ctx := context.Background()
	usr, err := LoadOrCreateUser(ctx, db, bob)
	if err != nil {
		t.Errorf("Failed to create bob: %v", err)
	}
	usr.Verified = true
	err = usr.Save(ctx, db)
	if err != nil {
		t.Errorf("Save() failed: %v", err)
	}
	ubob, err := LoadOrCreateUser(ctx, db, usr.Email)
	if err != nil {
		t.Errorf("Failed to load bob again: %v", err)
	}
	if usr.ID != ubob.ID {
		t.Error("ID mismatch")
	}
	if usr.Verified != ubob.Verified {
		t.Error("Save() failed to update: Verified mismatch")
	}
}
