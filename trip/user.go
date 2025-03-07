// Package trip implements the data model for managing trip expenses
// the key purpose is to compute the settlement of the expenses by the
// participants.
//
// This unit focuses on user data model. All participants in a trip
// are necessarily users.

package trip

import (
	"context"
	"database/sql"
	"log"
	"strings"
)

// Some global constants used to store SQL statements
const (
	userSelect         = "SELECT user_id, verified FROM tuser WHERE email=?"
	userInsert         = "INSERT INTO tuser (email, verified) VALUES (?, ?)"
	userUpdateVerified = "UPDATE tuser SET verified = ? WHERE user_id = ?"
)

// User refers to a registered user of the program.
// All participants of a trip, or an expenditure event
// must be a user.
type User struct {
	// This is the user_id column, value is from a sequence
	ID int64 `json:"id"`
	// This is the normalized email address of the user
	Email string `json:"email"`
	// The boolean reflects whether the email address has been verified
	Verified bool `json:"verified"`
}

// Users is for supporting sorting of []*User
type Users []*User

func (u Users) Len() int      { return len(u) }
func (u Users) Swap(i, j int) { u[i], u[j] = u[j], u[i] }
func (u Users) Less(i, j int) bool {
	if u[i].ID == u[j].ID {
		return u[i].Email < u[j].Email
	}
	return u[i].ID < u[j].ID
}

// Normalize an email address.
// Here, all it does is return a lowercased version of the address
func normalizeEmail(email string) string {
	return strings.ToLower(email)
}

// NewUser just returns an instance of User on the heap, with the given email address.
// It will normalize said address.
func NewUser(email string) *User {
	return &User{
		0,
		normalizeEmail(email),
		false,
	}
}

// LoadOrCreateUser returns a User instance by querying the database with the given
// email address. If the user doesn't exist, it'll create one.
func LoadOrCreateUser(ctx context.Context, db *sql.DB, email string) (*User, error) {
	stmt, err := db.PrepareContext(ctx, userSelect)
	if err != nil {
		return nil, err
	}
	defer stmt.Close()

	usr := NewUser(email)
	err = stmt.QueryRowContext(ctx, usr.Email).Scan(&usr.ID, &usr.Verified)
	switch {
	case err == sql.ErrNoRows:
		err = usr.Save(ctx, db)
		if err != nil {
			return nil, err
		}
	case err != nil:
		return nil, err
	}
	return usr, nil
}

// Save writes the User instance to the database.
// If the "ID" field is non-zero, then it would be an UPDATE operation.
// Otherwise, it will be an INSERT operation.
func (usr *User) Save(ctx context.Context, db *sql.DB) error {
	txn, err := db.BeginTx(ctx, nil)
	if err != nil {
		log.Printf("ERROR: Begin failed: %v\n", err)
		return err
	}

	var rslt sql.Result
	var stmt *sql.Stmt
	if usr.ID != 0 {
		stmt, err = txn.PrepareContext(ctx, userUpdateVerified)
		// We have an ID, so, we are updating instead. Since email
		// isn't really mutable, that means we are only updating the
		// "verified" column
	} else {
		stmt, err = txn.PrepareContext(ctx, userInsert)
	}
	if err != nil {
		log.Printf("ERROR: PrepareContext failed: %v\n", err)
		goto Rollback
	}
	defer stmt.Close()

	if usr.ID != 0 {
		rslt, err = stmt.ExecContext(ctx, usr.Verified, usr.ID)
		if err != nil {
			log.Printf("ERROR: update failed: %v\n", err)
			goto Rollback
		}
		cnt, err := rslt.RowsAffected()
		if err != nil {
			log.Printf("ERROR: RowsAffected() failed: %v\n", err)
			goto Rollback
		}
		if cnt != 1 {
			log.Printf("ERROR: Update affecting more than one row (%d) for user_id %d\n", cnt, usr.ID)
			goto Rollback
		}
	} else {
		rslt, err = stmt.ExecContext(ctx, usr.Email, usr.Verified)
		if err != nil {
			log.Printf("ERROR: insert failed: %v\n", err)
			goto Rollback
		}
		usr.ID, err = rslt.LastInsertId()
		if err != nil {
			log.Printf("ERROR: failed to get user_id: %v\n", err)
			goto Rollback
		}
	}
	err = txn.Commit()
	if err != nil {
		log.Printf("ERROR: commit failed: %v\n", err)
	}
	return err

Rollback:
	rollbackErr := txn.Rollback()
	if rollbackErr != nil {
		// If rollback fails, we should just abort
		log.Fatalf("ERROR: failed to rollback transaction on tuser '%v': '%v'", usr, rollbackErr)
	}
	return err
}
