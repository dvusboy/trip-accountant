package trip

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"sort"
	"strings"
	"time"
)

// Some global variables storing SQL statements
const (
	tripByOwnerSelect = `SELECT t.trip_id, t.name, t.name_lower, t.created_at, t.start_date, t.end_date, t.description
FROM trip AS t, participant AS p, tuser AS u
WHERE u.user_id = p.user_id
AND p.trip_id = t.trip_id
AND p.is_owner = true
AND t.end_date = 0
AND u.email = ?`
	tripByIDSelet = `SELECT trip_id, name, name_lower, created_at, start_date, end_date, description
FROM trip WHERE trip_id = ?`
	tripInsert = `INSERT INTO trip (name, name_lower, created_at, start_date, end_date, description)
VALUES (?, ?, ?, ?, ?, ?)`
	tripComplete = `UPDATE trip SET end_date = ?
WHERE trip_id = ?`

	peopleSelect = `
SELECT u.user_id, u.email, u.verified, p.is_owner
FROM tuser AS u, participant AS p
WHERE u.user_id = p.user_id
AND p.trip_id = ?`
	peopleInsert = "INSERT INTO participant (trip_id, user_id, is_owner) VALUES (?, ?, ?)"

	expenseSelect = `SELECT expense_id, txn_date, created_at, description
FROM expense WHERE trip_id = ? ORDER BY created_at`
	expenseInsert = `INSERT INTO expense (trip_id, txn_date, created_at, description)
VALUES (?, ?, ?, ?)`

	participantSelect = `SELECT u.email, ep.user_id, ep.amount
FROM expense_participant AS ep, tuser AS u
WHERE ep.user_id = u.user_id
AND ep.expense_id = ?`
	participantInsert = "INSERT INTO expense_participant (expense_id, user_id, amount) VALUES (?, ?, ?)"
)

var (
	// zeroTime is the time.Time object that represent epoch 0 (apparently, it cannot be const)
	zeroTime = time.UnixMicro(0)
)

// Participant is a user that participated in an expenditure event.
type Participant struct {
	// Email is the email address of the participating user
	Email string `json:"user"`
	// UserID is the primary key of the User record in the DB
	UserID int64 `json:"user_id"`
	// Paid is the amount this user paid (in cent)
	Paid int `json:"paid"`
}

// ByAmount is used for sorting the list of Participants by the amount Paid
// The default order is the one who paid the most is at the top (index 0)
type ByAmount []Participant

// Len is part of the sort.Interface
func (p ByAmount) Len() int {
	return len(p)
}

// Less is part of the sort.Interface
func (p ByAmount) Less(i, j int) bool {
	// Since this is a reverse order by Paid the comparison is ">" for Less()
	return p[i].Paid > p[j].Paid
}

// Swap is part of the sort.Interface
func (p ByAmount) Swap(i, j int) {
	p[i], p[j] = p[j], p[i]
}

// Date is just time.Time, but we only care about the date part
type Date struct {
	time.Time
}

// NewDate returns a Date instance with the time part of t set to 0s
func NewDate(t time.Time) Date {
	d := time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, t.Location())
	return Date{d}
}

// MarshalJSON is used to only output the date part of
// if d is zeroTime, then output is '""'
func (d Date) MarshalJSON() ([]byte, error) {
	var out []byte

	if d.Time.Equal(zeroTime) {
		out = []byte(`""`)
	} else {
		out = []byte(`"`)
		out = d.Time.AppendFormat(out, time.DateOnly)
		out = append(out, '"')
	}
	return out, nil
}

// UnmarshalJSON just calls the version of time.Time
func (d *Date) UnmarshalJSON(data []byte) error {
	return d.Time.UnmarshalJSON(data)
}

// Expense records the details of an expenditure event
type Expense struct {
	// ID is the primary key of the table
	ID int64 `json:expense_id`
	// Date is the transaction date in `YYYY-MM-DD` format
	Date Date `json:"date"`
	// Description describes the expenditure event
	Description string `json:description`
	// Participants is a list of the participating users
	Participants []Participant `json:participants`
	// createdAt is the epoch timestamp of entry creation
	createdAt time.Time
	// amount is the sum of the amount paid
	amount int
}

// Expenses is for sorting []*Expense
type Expenses []*Expense

func (e Expenses) Len() int      { return len(e) }
func (e Expenses) Swap(i, j int) { e[i], e[j] = e[j], e[i] }
func (e Expenses) Less(i, j int) bool {
	if e[i].ID == e[j].ID {
		return e[i].Date.Time.Before(e[j].Date.Time)
	}
	return e[i].ID < e[j].ID
}

// Trip represent a single trip.
type Trip struct {
	// ID is the primary key and is from a sequence
	ID int64 `json:"trip_id"`
	// Name is a short name, and is the main identifier of a trip
	Name string `json:"name" binding:"required,max=127"`
	// Owner is the User that created this trip
	Owner *User `json:"owner" binding:"required"`
	// StartDate is the start of the trip in YYYY-MM-DD format
	StartDate Date `json:"start_date"`
	// EndDate is the end of the trip in YYYY-MM-DD format
	// but can be empty if still active
	EndDate time.Time `json:"end_date"`
	// Description contains additional details on the trip
	Description string `json:"description"`
	// Participants is a list of users, excluding the owner, participating the trip
	Participants []*User `json:"participants" binding:"required"`
	// Expenses is a list of Expense instances incurred during the trip
	Expenses []*Expense `json:"expenses"`
	// nameLower is the normalized version of "Name"
	nameLower string
	// createdAt is the Epoch timestamp in µs of the object creation
	createdAt time.Time
	// emailLookup is a map to lookup User.ID from email address
	emailLookup map[string]int64
	// totalExpense is the sum of all the expenses
	totalExpense int
}

// Payments register the payees and amounts a payer needs to make
// key is the payee
// value is the amount
type Payments map[string]int

// Settlement lay out the payment distribution for all the expenses of a trip
// key is the payer
// value is a list of Payment
type Settlement map[string]Payments

// normalizeName returns the lowercased version of the given name
func normalizeName(name string) string {
	return strings.ToLower(name)
}

// epochToDate returns a time.Time instance from the epoch values stored in DB
func epochToDate(tstamp int64) Date {
	r := time.Unix(int64(tstamp), 0).UTC()
	return NewDate(r)
}

// NewTrip creates an instance of Trip. Only email addresses are provided
// in the arguments, and no DB operation will happen
func NewTrip(name, owner, description string, startDate Date, participants []string) *Trip {
	trip := Trip{
		ID:           0,
		Name:         name,
		Owner:        NewUser(owner),
		StartDate:    startDate,
		EndDate:      zeroTime,
		Description:  description,
		nameLower:    normalizeName(name),
		createdAt:    zeroTime,
		emailLookup:  make(map[string]int64),
		totalExpense: 0,
	}
	for _, p := range participants {
		u := NewUser(p)
		if u.Email != trip.Owner.Email {
			trip.Participants = append(trip.Participants, u)
		} else {
			log.Printf("WARNING: owner '%s' is also in the list of participants '%v', ignoring.\n", owner, participants)
		}
	}
	return &trip
}

// LoadTripsByOwner returns all the Trip instances from the database,
// given the owner email address
func LoadTripsByOwner(ctx context.Context, db *sql.DB, owner string) (map[string]*Trip, error) {
	stmt, err := db.PrepareContext(ctx, tripByOwnerSelect)
	if err != nil {
		return nil, err
	}
	defer stmt.Close()

	rows, err := stmt.QueryContext(ctx, normalizeEmail(owner))
	if err != nil {
		log.Printf("ERROR: tripByOwnerSelect failed: %v\n", err)
		return nil, err
	}
	defer rows.Close()

	rslt := make(map[string]*Trip)
	for rows.Next() {
		var startDate, endDate, createdAt int64

		trip := new(Trip)
		trip.emailLookup = make(map[string]int64)
		err = rows.Scan(&trip.ID, &trip.Name, &trip.nameLower, &createdAt, &startDate, &endDate, &trip.Description)
		if err != nil {
			log.Printf("ERROR: failed to read in trip row with Scan '%v'\n", err)
			return nil, err
		}
		trip.createdAt = time.UnixMicro(createdAt).UTC()
		trip.StartDate = NewDate(time.Unix(startDate, 0).UTC())
		trip.EndDate = time.Unix(endDate, 0).UTC()
		err = trip.loadParts(ctx, db)
		if err != nil {
			return nil, err
		}
		rslt[trip.nameLower] = trip
	}
	err = rows.Err()
	if err != nil {
		log.Printf("ERROR: rows operation failed: %v\n", err)
		return nil, err
	}
	return rslt, nil
}

// LoadTripByID loads a single trip by the primary key
func LoadTripByID(ctx context.Context, db *sql.DB, id int64) (*Trip, error) {
	stmt, err := db.PrepareContext(ctx, tripByIDSelet)
	if err != nil {
		return nil, err
	}
	defer stmt.Close()

	var startDate, endDate, createdAt int64
	trip := new(Trip)
	trip.emailLookup = make(map[string]int64)
	err = stmt.QueryRowContext(ctx, id).Scan(&trip.ID, &trip.Name, &trip.nameLower, &createdAt, &startDate, &endDate, &trip.Description)
	if err != nil {
		return nil, err
	}
	trip.createdAt = time.UnixMicro(createdAt).UTC()
	trip.StartDate = NewDate(time.Unix(startDate, 0).UTC())
	trip.EndDate = time.Unix(endDate, 0).UTC()
	err = trip.loadParts(ctx, db)
	if err != nil {
		return nil, err
	}
	return trip, nil
}

// loadParts loads the list of participants and expenses from the DB
func (trip *Trip) loadParts(ctx context.Context, db *sql.DB) error {
	stmt, err := db.PrepareContext(ctx, peopleSelect)
	if err != nil {
		return err
	}
	defer stmt.Close()

	rows, err := stmt.QueryContext(ctx, trip.ID)
	if err != nil {
		log.Printf("ERROR: Query for participants of trip %d failed '%v'\n", trip.ID, err)
		return err
	}
	defer rows.Close()

	var isOwner bool
	for rows.Next() {
		usr := new(User)
		err = rows.Scan(&usr.ID, &usr.Email, &usr.Verified, &isOwner)
		if err != nil {
			log.Printf("ERROR: failed to read in participant with Scan '%v'\n", err)
			return err
		}
		if isOwner {
			trip.Owner = usr
		} else {
			trip.Participants = append(trip.Participants, usr)
		}
		trip.emailLookup[usr.Email] = usr.ID
	}
	return trip.loadExpenses(ctx, db)
}

// Equals evaluates if 2 instances of Trip are equal
func (trip *Trip) Equals(trip2 *Trip) bool {
	if trip.ID != trip2.ID {
		return false
	}
	if trip.Name != trip2.Name {
		return false
	}
	if *trip.Owner != *trip2.Owner {
		return false
	}
	if !trip.StartDate.Time.Equal(trip2.StartDate.Time) {
		return false
	}
	if !trip.EndDate.Equal(trip2.EndDate) {
		return false
	}
	if trip.Description != trip2.Description {
		return false
	}
	if Users(trip.Participants).Len() != Users(trip2.Participants).Len() {
		return false
	}
	sort.Sort(Users(trip.Participants))
	sort.Sort(Users(trip2.Participants))
	for i := 0; i < Users(trip.Participants).Len(); i++ {
		if *trip.Participants[i] != *trip2.Participants[i] {
			return false
		}
	}
	if len(trip.Expenses) != len(trip2.Expenses) {
		return false
	}
	sort.Sort(Expenses(trip.Expenses))
	sort.Sort(Expenses(trip2.Expenses))
	for i := 0; i < len(trip.Expenses); i++ {
		if trip.Expenses[i].Equals(trip2.Expenses[i]) {
			return false
		}
	}
	return true
}

// createTrip is used in Save() to make that function a bit more compact
// It's expected to be executed within a transaction
func (trip *Trip) createTrip(ctx context.Context, txn *sql.Tx, now time.Time) (err error) {
	var rslt sql.Result
	var tStmt, pStmt *sql.Stmt

	tStmt, err = txn.PrepareContext(ctx, tripInsert)
	if err != nil {
		return err
	}
	defer tStmt.Close()

	pStmt, err = txn.PrepareContext(ctx, peopleInsert)
	if err != nil {
		return err
	}
	defer pStmt.Close()

	// Set createdAt, if necessary
	if trip.createdAt.IsZero() {
		trip.createdAt = now
	}
	rslt, err = tStmt.ExecContext(ctx,
		trip.Name, trip.nameLower,
		trip.createdAt.UnixMicro(),
		trip.StartDate.Unix(), trip.EndDate.Unix(),
		trip.Description)
	if err != nil {
		return err
	}

	trip.ID, err = rslt.LastInsertId()
	if err != nil {
		return err
	}

	rslt, err = pStmt.ExecContext(ctx, trip.ID, trip.Owner.ID, true)
	if err != nil {
		return err
	}
	for _, p := range trip.Participants {
		rslt, err = pStmt.ExecContext(ctx, trip.ID, p.ID, false)
		if err != nil {
			return err
		}
	}
	return nil
}

// Save writes the Trip instance to database
func (trip *Trip) Save(ctx context.Context, db *sql.DB) (err error) {
	now := time.Now()
	// first we deal with the users
	if trip.Owner.ID == 0 {
		trip.Owner, err = LoadOrCreateUser(ctx, db, trip.Owner.Email)
		if err != nil {
			return err
		}
	}
	trip.emailLookup[trip.Owner.Email] = trip.Owner.ID
	for i, p := range trip.Participants {
		if p.ID == 0 {
			trip.Participants[i], err = LoadOrCreateUser(ctx, db, p.Email)
			if err != nil {
				return err
			}
		}
		trip.emailLookup[trip.Participants[i].Email] = trip.Participants[i].ID
	}

	txn, err := db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}

	var rslt sql.Result
	var eStmt, epStmt *sql.Stmt

	// Do trip and participant insert only when trip.ID is 0
	if trip.ID == 0 {
		err = trip.createTrip(ctx, txn, now)
		if err != nil {
			goto Rollback
		}
	}

	// Deal with expenses
	eStmt, err = txn.PrepareContext(ctx, expenseInsert)
	if err != nil {
		goto Rollback
	}
	defer eStmt.Close()

	epStmt, err = txn.PrepareContext(ctx, participantInsert)
	if err != nil {
		goto Rollback
	}
	defer epStmt.Close()

	for _, e := range trip.Expenses {
		if e.ID != 0 {
			// This expense is already handled
			continue
		}
		if e.createdAt.IsZero() {
			e.createdAt = now
		}
		rslt, err = eStmt.ExecContext(ctx, trip.ID, e.Date.Unix(), e.createdAt.UnixMicro(), e.Description)
		if err != nil {
			goto Rollback
		}
		e.ID, err = rslt.LastInsertId()
		if err != nil {
			goto Rollback
		}
		var ok bool
		for j, ep := range e.Participants {
			if ep.UserID == 0 {
				ep.UserID, ok = trip.emailLookup[normalizeEmail(ep.Email)]
				if !ok {
					log.Printf("ERROR: Expense participant '%s' not in the list of trip participants\n", ep.Email)
					goto Rollback
				}
				// also update the UserID in the array
				e.Participants[j].UserID = ep.UserID
			}
			_, err = epStmt.ExecContext(ctx, e.ID, ep.UserID, ep.Paid)
			if err != nil {
				goto Rollback
			}
		}
	}
	return txn.Commit()

Rollback:
	rollbackErr := txn.Rollback()
	if rollbackErr != nil {
		log.Fatalf("ERROR: trip.Save() failed to rollback transaction on trip '%v': '%v'\n", trip, rollbackErr)
	}
	return err
} // Save()

// loadExpenses loads the Expenses attribute with a list of Expense objects for the trip
func (trip *Trip) loadExpenses(ctx context.Context, db *sql.DB) error {
	eStmt, err := db.PrepareContext(ctx, expenseSelect)
	if err != nil {
		return err
	}
	defer eStmt.Close()

	pStmt, err := db.PrepareContext(ctx, participantSelect)
	if err != nil {
		return err
	}
	defer pStmt.Close()

	eRows, err := eStmt.QueryContext(ctx, trip.ID)
	switch {
	case err == sql.ErrNoRows:
		return nil
	case err != nil:
		return err
	}
	defer eRows.Close()

	var txnDate, createdAt int64
	clear(trip.Expenses)
	for eRows.Next() {
		e := new(Expense)
		err = eRows.Scan(&e.ID, &txnDate, &createdAt, &e.Description)
		if err != nil {
			return err
		}
		e.Date = NewDate(time.Unix(txnDate, 0).UTC())
		e.createdAt = time.UnixMicro(createdAt).UTC()

		pRows, err := pStmt.QueryContext(ctx, e.ID)
		if err != nil {
			return err
		}
		defer pRows.Close()

		for pRows.Next() {
			p := Participant{}
			err = pRows.Scan(&p.Email, &p.UserID, &p.Paid)
			if err != nil {
				return err
			}
			e.Participants = append(e.Participants, p)
			e.amount += p.Paid
		}
		trip.Expenses = append(trip.Expenses, e)
		trip.totalExpense += e.amount
	}
	return nil
}

// AddExpense adds an Expense object to the Trip object
func (trip *Trip) AddExpense(date Date, description string, participants []Participant) error {
	expense := Expense{
		Date:         date,
		Description:  description,
		Participants: []Participant{},
		createdAt:    zeroTime,
		amount:       0,
	}
	for _, ep := range participants {
		email := normalizeEmail(ep.Email)
		id, ok := trip.emailLookup[email]
		if !ok {
			return fmt.Errorf("Expense participant '%s' not part of the trip", email)
		}
		p := Participant{
			Email:  email,
			UserID: id,
			Paid:   ep.Paid,
		}
		expense.Participants = append(expense.Participants, p)
		expense.amount += p.Paid
	}
	trip.Expenses = append(trip.Expenses, &expense)
	trip.totalExpense += expense.amount
	return nil
}

// Equals evaluates if 2 Expense instances are Equals
func (expense *Expense) Equals(expense2 *Expense) bool {
	if expense.ID != expense2.ID {
		return false
	}
	if expense.Date != expense2.Date {
		return false
	}
	if expense.Description != expense2.Description {
		return false
	}
	if len(expense.Participants) != len(expense2.Participants) {
		return false
	}
	sort.Sort(ByAmount(expense.Participants))
	sort.Sort(ByAmount(expense2.Participants))
	for i := 0; i < len(expense.Participants); i++ {
		if expense.Participants[i] != expense2.Participants[i] {
			return false
		}
	}
	return true
}

// Settle computes the settlement for a single expenditure event
func (expense Expense) Settle() Settlement {
	rslt := make(Settlement)
	n := len(expense.Participants)
	// make a copy of the Participants
	p := make([]Participant, len(expense.Participants))
	copy(p, expense.Participants)
	// sort the list of Participants by amount paid
	sort.Sort(ByAmount(p))
	avg := int(float64(expense.amount)/float64(n) + 0.5) // round up
	var i, j int = 0, n - 1
	var ok bool

	for i < j {
		if p[i].Paid > avg {
			// i paid too much
			if p[j].Paid < avg {
				// j paid too little
				amount := min(avg-p[j].Paid, p[i].Paid-avg)
				_, ok = rslt[p[j].Email]
				if !ok {
					rslt[p[j].Email] = make(Payments)
				}
				rslt[p[j].Email][p[i].Email] += amount
				p[j].Paid += amount
				p[i].Paid -= amount
			} else {
				j--
			}
		} else {
			i++
		}
	}
	return rslt
}

// upsertAmount registers the payment and add a the lookup key
func (s Settlement) upsertAmount(payer, payee string, amount int, lookup map[string]bool) {
	key := fmt.Sprintf("%s>%s", payer, payee)
	_, ok := s[payer]
	if !ok {
		s[payer] = Payments{payee: amount}
	} else {
		s[payer][payee] += amount
	}
	lookup[key] = true
}

// Complete computes the full Settlement for the whole trip and sets the end_date
func (trip *Trip) Complete(ctx context.Context, db *sql.DB) (Settlement, error) {
	now := time.Now()
	rslt := make(Settlement)
	// This is a lookup to catch A pays B and B pays A situation
	lookup := make(map[string]bool)
	var yek string
	for _, e := range trip.Expenses {
		for k, v := range e.Settle() {
			for rcv, amt := range v {
				yek = fmt.Sprintf("%s>%s", rcv, k)
				_, exists := lookup[yek]
				if exists {
					// payee also pays payer
					if rslt[rcv][k] >= amt {
						// payee is paying more
						rslt[rcv][k] -= amt
						if (rslt[rcv][k]) == 0 {
							delete(rslt[rcv], k)
						}
						// no need to call rslt.upsertAmount()
					} else {
						// payer is paying more
						amt -= rslt[rcv][k]
						delete(rslt[rcv], k)
						delete(lookup, yek)
						rslt.upsertAmount(k, rcv, amt, lookup)
					}
				} else {
					rslt.upsertAmount(k, rcv, amt, lookup)
				}
			}
		}
	}
	txn, err := db.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	stmt, err := txn.PrepareContext(ctx, tripComplete)
	if err != nil {
		goto Rollback
	}
	defer stmt.Close()

	_, err = stmt.ExecContext(ctx, now.Unix(), trip.ID)
	if err != nil {
		goto Rollback
	}
	err = txn.Commit()
	if err != nil {
		goto Rollback
	}
	return rslt, nil

Rollback:
	rollbackErr := txn.Rollback()
	if rollbackErr != nil {
		log.Fatalf("ERROR: trip.Complete() failed to rollback transaction on trip '%v': '%v'\n", trip, rollbackErr)
	}
	return nil, err
}
