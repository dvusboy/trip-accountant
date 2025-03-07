// Package trip implements the data model for managing trip expenses
// the key purpose is to compute the settlement of the expenses by the
// participants.
//
// This unit implements some unit tests for trip and expense related
// functions.

package trip

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"math"
	"os"
	"path/filepath"
	"testing"
	"time"

	_ "github.com/mattn/go-sqlite3"
)

const (
	tripCreate = `CREATE TABLE IF NOT EXISTS trip (
trip_id INTEGER CONSTRAINT trip_pkey PRIMARY KEY AUTOINCREMENT,
name VARCHAR(128) NOT NULL,
name_lower VARCHAR(128) NOT NULL,
created_at INTEGER NOT NULL,
start_date INTEGER NOT NULL,
end_date INTEGER DEFAULT 0,
description VARCHAR(512))`
	tripDrop = "DROP TABLE IF EXISTS trip"

	participantCreate = `CREATE TABLE IF NOT EXISTS participant (
trip_id INTEGER NOT NULL,
user_id INTEGER NOT NULL,
is_owner BOOLEAN NOT NULL DEFAULT FALSE,
CONSTRAINT participant_pkey PRIMARY KEY (trip_id, user_id))`
	participantDrop = "DROP TABLE IF EXISTS participant"

	expenseCreate = `CREATE TABLE IF NOT EXISTS expense (
expense_id INTEGER CONSTRAINT expense_pkey PRIMARY KEY AUTOINCREMENT,
trip_id INTEGER NOT NULL,
txn_date INTEGER NOT NULL,
created_at INTEGER NOT NULL,
description VARCHAR(512))`
	expenseTripIndex     = "CREATE INDEX IF NOT EXISTS expense_trip_index ON expense(trip_id)"
	expenseDrop          = "DROP TABLE IF EXISTS expense"
	expenseTripIndexDROP = "DROP INDEX IF EXISTS expense_trip_index"

	expenseParticipantCreate = `CREATE TABLE IF NOT EXISTS expense_participant (
expense_id INTEGER NOT NULL,
user_id INTEGER NOT NULL,
amount INTEGER NOT NULL,
CONSTRAINT expense_participant_pkey PRIMARY KEY (expense_id, user_id))`
	expenseParticipantDrop = "DROP TABLE IF EXISTS expense_participant"
)

var (
	db           *sql.DB
	trip1, trip2 *Trip
)

func setupSchema() {
	ctx := context.Background()
	_, err := db.ExecContext(ctx, tuserCreate)
	if err != nil {
		log.Fatal(err)
	}
	_, err = db.ExecContext(ctx, tripCreate)
	if err != nil {
		log.Fatal(err)
	}
	_, err = db.ExecContext(ctx, participantCreate)
	if err != nil {
		log.Fatal(err)
	}
	_, err = db.ExecContext(ctx, expenseCreate)
	if err != nil {
		log.Fatal(err)
	}
	_, err = db.ExecContext(ctx, expenseTripIndex)
	if err != nil {
		log.Fatal(err)
	}
	_, err = db.ExecContext(ctx, expenseParticipantCreate)
	if err != nil {
		log.Fatal(err)
	}
}

// TestMain initializes the DB handle and schema
func TestMain(m *testing.M) {
	tmpDir := os.TempDir()
	dbFile := filepath.Join(tmpDir, "trip_test.db")
	os.Remove(dbFile)
	fmt.Printf("Initializing DB handle to %s\n", dbFile)
	var err error
	db, err = sql.Open("sqlite3", dbFile)
	if err != nil {
		log.Fatalf("Failed to open DB: %v", err)
	}
	defer db.Close()
	os.Remove(dbFile)

	setupSchema()
	rs := m.Run()
	os.Exit(rs)
}

func trip1Setup() {
	startDate := epochToDate(time.Now().Unix())
	trip1 = NewTrip(
		"Trip 1",
		alice,
		"Trip 1 for testing",
		startDate,
		[]string{bob, charlie, david, elise, fred, greg},
	)
}

func createTrip1(ctx context.Context) error {
	trip1Setup()
	return trip1.Save(ctx, db)
}

func trip2Setup() {
	startDate := epochToDate(time.Now().Unix() - 86400*7)
	trip2 = NewTrip(
		"Trip 2",
		alice,
		"Trip 2 for testing",
		startDate,
		[]string{bob, charlie},
	)
}

func createTrip2(ctx context.Context) error {
	trip2Setup()
	return trip2.Save(ctx, db)
}

// TestTripCreation creates both Trip 1 and Trip 2 in the DB
func TestTripCreation(t *testing.T) {
	ctx := context.Background()
	err := createTrip1(ctx)
	if err != nil {
		t.Errorf("Failed to create Trip 1: %v", err)
	}
	err = createTrip2(ctx)
	if err != nil {
		t.Errorf("Failed to create Trip 2: %v", err)
	}
}

// TestLoadTripsByOwner test loading trips by owner
func TestLoadTripsByOwner(t *testing.T) {
	ctx := context.Background()
	trips, err := LoadTripsByOwner(ctx, db, alice)
	if err != nil {
		t.Error(err)
	}
	if !trips["trip 1"].Equals(trip1) {
		t.Errorf("trips[\"trip 1\"] %#v != trip1 %#v", *trips["trip 1"], *trip1)
	}
	if !trips["trip 2"].Equals(trip2) {
		t.Errorf("trips[\"trip 2\"] %#v != trip2 %#v", *trips["trip 2"], *trip2)
	}
}

// TestLoadTripByID test loading of a singel trip instance by its ID
func TestLoadTripByID(t *testing.T) {
	ctx := context.Background()
	// load Trip 1
	t1, err := LoadTripByID(ctx, db, 1)
	if err != nil {
		t.Errorf("Failed to load Trip 1 by ID: %v", err)
	}
	if !t1.Equals(trip1) {
		t.Errorf("t1 %#v != trip1 %#v", *t1, *trip1)
	}
}

// TestAddExpense test adding expenditure even to a trip
// Trip2 is used for this
func TestAddExpense(t *testing.T) {
	ctx := context.Background()
	now := time.Now()
	p := []Participant{
		{alice, 0, 6000 /* $60 */},
		{bob, 0, 0},
		{charlie, 0, 0},
	}
	err := trip2.AddExpense(NewDate(time.Unix(now.Unix()-86400*7, 0)), "tickets", p)
	if err != nil {
		t.Error(err)
	}
	err = trip2.Save(ctx, db)
	if err != nil {
		t.Error(err)
	}
	// Now this should fail because "elise" is not part of Trip 2
	pb := []Participant{
		{alice, 0, 0},
		{elise, 0, 1000 /* $10 */},
	}
	err = trip2.AddExpense(NewDate(now), "should fail", pb)
	if err == nil {
		t.Error("An expected-to-fail AddExpense() has succeeded.")
	}
	// ignore the failure
	pc := []Participant{
		{alice, 0, 3000},
		{charlie, 0, 0},
	}
	err = trip2.AddExpense(NewDate(now), "dinner", pc)
	if err != nil {
		t.Error(err)
	}
	err = trip2.Save(ctx, db)
	if err != nil {
		t.Error(err)
	}
}

// TestComplete testing the settlement algorithm
// For Trip 2, we have:
//   - Alice paid for the 3 tickets for a total of $60 (6000 cents)
//   - Alice also paid for dinner with Charlie for $30 (3000 cents)
//
// So, the settlement should be:
//   - Bob pays Alice just for the ticket: $20 or 2000c
//   - Charlie pays Alice for both ticket and dinner: $20 + $15 or 3500c
func TestComplete(t *testing.T) {
	ctx := context.Background()
	s, err := trip2.Complete(ctx, db)
	if err != nil {
		t.Error(err)
	}
	log.Printf("Settlement: %#v\n", s)
	if len(s) != 2 {
		t.Errorf("Settlement should only have 2 entries instead of %d", len(s))
	}
	if s[bob][alice] != 2000 {
		t.Errorf("Settlement for Bob -> Alice is incorrect: %d, should be 2000", s[bob][alice])
	}
	if s[charlie][alice] != 3500 {
		t.Errorf("Settlement for Charlie -> Alice is incorrect: %d, should be 3500", s[charlie][alice])
	}
}

// TestAddExpense2 this deal with Trip 1
func TestAddExpense2(t *testing.T) {
	ctx := context.Background()
	now := time.Now()
	p1 := []Participant{
		{alice, 0, 41500},
		{bob, 0, 0},
		{charlie, 0, 0},
		{david, 0, 0},
		{elise, 0, 0},
		{fred, 0, 0},
		{greg, 0, 2500},
	}
	err := trip1.AddExpense(NewDate(now), "lodging", p1)
	if err != nil {
		t.Error(err)
	}
	err = trip2.Save(ctx, db)
	if err != nil {
		t.Error(err)
	}
	p2 := []Participant{
		{elise, 0, 0},
		{david, 0, 10800},
		{fred, 0, 0},
		{greg, 0, 0},
	}
	err = trip1.AddExpense(NewDate(time.Unix(now.Unix()-86400, 0)), "dinner", p2)
	if err != nil {
		t.Error(err)
	}
	p3 := []Participant{
		{alice, 0, 0},
		{bob, 0, 0},
		{charlie, 0, 5900},
	}
	err = trip1.AddExpense(NewDate(now), "group 1 lunch", p3)
	if err != nil {
		t.Error(err)
	}
	p4 := []Participant{
		{david, 0, 7000},
		{elise, 0, 0},
		{fred, 0, 0},
		{greg, 0, 0},
	}
	err = trip1.AddExpense(NewDate(now), "group 2 lunch", p4)
	if err != nil {
		t.Error(err)
	}
	err = trip1.Save(ctx, db)
	if err != nil {
		t.Error(err)
	}
}

// TestTrip1Complete testing settlement on Trip 1
// For Trip 1, we have:
//   - Alice paid for the bulk of lodging cost $415,
//     Greg also paid $25 for incidentals
//     All 7 participants are part of this group
//   - David paid for dinner in the amount of $108
//     Elise, Fred, and Greg joined him
//   - 2 lunch groups:
//   - David paid $70 for lunch with Elise, Fred, and Greg
//   - Charlie paid $59 for lunch with Alice, and Bob
//
// The settlement should look like this:
//
//	For lodging, total cost is 44000c, and everyone
//	  owes Alice. Each should pay 44000c/7 = 6286c, rounded
//	- Bob > Alice: 6286c
//	- Charlie > Alice: 6286c
//	- David > Alice: 6286c
//	- Elise > Alice: 6286c
//	- Fred > Alice: 6286c
//	- Greg > Alice: 44000-6*6286-2500 = 3784c
//
//	For dinner, total cost is 10800c. Each pays 2700c
//	- Elise > David: 2700c
//	- Fred > David: 2700c
//	- Greg > David: 2700c
//
//	For group 1 lunch, total cost is 5900c. Each pays 1967c (rounded)
//	- Alice > Charlie: 1967c
//	- Bob > Charlie: 1967c
//
//	For group 2 lunch: total cost is 7000c. Each pays 1750c
//	- Elise > David: 1750c
//	- Fred > David: 1750c
//	- Greg > David: 1750c
//
// The net would then be:
//   - Bob > Alice: 6286c, Bob > Charlie: 1967c
//   - Charlie > Alice: 6286-1966 = 4320c
//   - David > Alice: 6286c
//   - Elise > Alice: 6286c, Elise > David: 2700+1750 = 4450c
//   - Fred > Alice: 6286c, Fred > David: 2700+1750c
//   - Greg > Alice: 3784c, Gred > David: 2700+1750 = 4450c
//
// NOTE: Since we have rounding, if | v1 - v2 | < 3 then they
// can be considered equal
func TestTrip1Complete(t *testing.T) {
	ctx := context.Background()
	s, err := trip1.Complete(ctx, db)
	if err != nil {
		t.Error(err)
	}
	log.Printf("Settlement: %#v\n", s)
	if len(s) != 6 {
		t.Errorf("Expect 6 entries in settlement, got %d", len(s))
	}
	if math.Abs(float64(s[bob][alice]-6286)) >= 3 {
		t.Errorf("Bob is paying Alice too much: %d vs 6286", s[bob][alice])
	}
	if math.Abs(float64(s[bob][charlie]-1967)) >= 3 {
		t.Errorf("Bob is paying Charlie too much: %d vs 1967", s[bob][charlie])
	}
	if math.Abs(float64(s[charlie][alice]-4320)) >= 3 {
		t.Errorf("Charlie is paying Alice too much: %d vs 4320", s[charlie][alice])
	}
	if math.Abs(float64(s[david][alice]-6286)) >= 3 {
		t.Errorf("David is paying Alice too much: %d vs 6286", s[david][alice])
	}
	if math.Abs(float64(s[elise][alice]-6286)) >= 3 {
		t.Errorf("Elise is paying Alice too much: %d vs 6286", s[elise][alice])
	}
	if math.Abs(float64(s[elise][david]-4450)) >= 3 {
		t.Errorf("Elise is paying David too much: %d vs 4450", s[elise][david])
	}
	if math.Abs(float64(s[fred][alice]-6286)) >= 3 {
		t.Errorf("Fred is paying Alice too much: %d vs 6286", s[fred][alice])
	}
	if math.Abs(float64(s[fred][david]-4450)) >= 3 {
		t.Errorf("Fred is paying David too much: %d vs 1750", s[fred][david])
	}
	if math.Abs(float64(s[greg][alice]-3784)) >= 3 {
		t.Errorf("Greg is paying Alice too much: %d vs 3784", s[greg][alice])
	}
	if math.Abs(float64(s[greg][david]-4450)) >= 3 {
		t.Errorf("Greg is paying David too much: %d vs 4450", s[greg][david])
	}
}
