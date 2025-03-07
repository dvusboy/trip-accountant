package main

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"strconv"
	"time"

	"github.com/dvusboy/trip-accountant/trip"
	"github.com/gin-gonic/gin"
	_ "github.com/mattn/go-sqlite3"
	flag "github.com/spf13/pflag"
)

// Globals
var (
	// db is the database handle
	db *sql.DB
	// dbPath is the SQLite3 DB file path, it'd be extracted from dbURL
	dbPath string
	// dbURL is for storing flag --db for DB access URL
	dbURL = "sqlite3:///srv/trip-accountant/data/trips.db"
	// port is the listening port, defaults to 8081
	port = 8081
)

// tripJSON is used for POST to create trips
// this is needed to handle []*Object, as Bind can't seem
// to handle them.
type tripJSON struct {
	Name         string   `json:"name" binding:"required,max=127"`
	Owner        string   `json:"owner" binding:"required"`
	StartDate    string   `json:"start_date" binding:"required"`
	Description  string   `json:"description" binding:"required,max=511"`
	Participants []string `json:"participants" binding:"required"`
}

// Translate maps a tripJSON instance into Trip instance
func (t tripJSON) Translate() (*trip.Trip, error) {
	sd, err := time.Parse(time.DateOnly, t.StartDate)
	if err != nil {
		return nil, err
	}
	return trip.NewTrip(t.Name, t.Owner, t.Description, trip.NewDate(sd), t.Participants), nil
}

// expenseJSON is used for POST to create expense of a trip
type expenseJSON struct {
	Date         string         `json:"date" binding:"required"`
	Description  string         `json:"description" binding:"required"`
	Participants map[string]int `json:"participants" binding:"required"`
}

// Translate maps a expenseJSON into Expense
func (e expenseJSON) Translate() (*trip.Expense, error) {
	sd, err := time.Parse(time.DateOnly, e.Date)
	if err != nil {
		return nil, err
	}
	r := new(trip.Expense)
	r.Date = trip.NewDate(sd)
	r.Description = e.Description
	r.Participants = []trip.Participant{}
	for email, paid := range e.Participants {
		p := trip.Participant{
			Email:  email,
			UserID: 0,
			Paid:   paid,
		}
		r.Participants = append(r.Participants, p)
	}
	return r, nil
}

// init sets up the CLI flags
func init() {
	flag.IntVar(&port, "port", port, "bind port")
	flag.StringVar(&dbURL, "db", dbURL, "database URL")
}

// handlerFunc is our HandlerFunc that takes an additional DB handler argument.
type handlerFunc func(*gin.Context, *sql.DB)

// handlerWrapper wraps our handlerFunc into gin.HandlerFunc
func handlerWrapper(db *sql.DB, f handlerFunc) gin.HandlerFunc {
	return func(c *gin.Context) {
		f(c, db)
	}
}

// jsonBail sends an error status and a JSON message payload
func jsonBail(c *gin.Context, status int, err error) {
	log.Printf("ERROR: jsonBail(status=%d, error=%v", status, err)
	c.Error(err)
	c.JSON(status, c.Errors.JSON())
	c.Abort()
}

// postTrip creates a new trip
func postTrip(c *gin.Context, db *sql.DB) {
	var t tripJSON

	err := c.ShouldBindJSON(&t)
	if err != nil {
		jsonBail(c, http.StatusBadRequest, err)
		return
	}

	trip, err := t.Translate()
	if err != nil {
		jsonBail(c, http.StatusBadRequest, err)
		return
	}

	ctx := context.Background()
	err = trip.Save(ctx, db)
	if err != nil {
		jsonBail(c, http.StatusBadRequest, err)
		return
	}
	c.JSON(http.StatusCreated, gin.H{"trip_id": trip.ID})
}

// getTrips returns the active trips owned by a user
func getTrips(c *gin.Context, db *sql.DB) {
	owner := c.Params.ByName("owner")
	ctx := context.Background()
	trips, err := trip.LoadTripsByOwner(ctx, db, owner)
	switch {
	case err == sql.ErrNoRows:
		jsonBail(c, http.StatusNotFound, err)
		return
	case err != nil:
		jsonBail(c, http.StatusBadRequest, err)
		return
	}
	c.JSON(http.StatusOK, trips)
}

// postExpense add an expenditure even to a trip
func postExpense(c *gin.Context, db *sql.DB) {
	tripID, err := strconv.ParseInt(c.Params.ByName("trip_id"), 10, 64)
	if err != nil {
		jsonBail(c, http.StatusBadRequest, err)
		return
	}

	ctx := context.Background()
	t, err := trip.LoadTripByID(ctx, db, tripID)
	switch {
	case err == sql.ErrNoRows:
		jsonBail(c, http.StatusNotFound, err)
		return
	case err != nil:
		jsonBail(c, http.StatusBadRequest, err)
		return
	}

	var expense expenseJSON
	err = c.ShouldBindJSON(&expense)
	if err != nil {
		jsonBail(c, http.StatusBadRequest, err)
		return
	}

	e, err := expense.Translate()
	if err != nil {
		jsonBail(c, http.StatusBadRequest, err)
		return
	}
	err = t.AddExpense(e.Date, e.Description, e.Participants)
	if err != nil {
		jsonBail(c, http.StatusBadRequest, err)
		return
	}
	err = t.Save(ctx, db)
	if err != nil {
		jsonBail(c, http.StatusBadRequest, err)
		return
	}
	e = t.Expenses[len(t.Expenses)-1]
	c.JSON(http.StatusAccepted, gin.H{"expense_id": e.ID})
}

// getExpenses returns the list of expenses incurred during the trip
func getExpenses(c *gin.Context, db *sql.DB) {
	tripID, err := strconv.ParseInt(c.Params.ByName("trip_id"), 10, 64)
	if err != nil {
		jsonBail(c, http.StatusBadRequest, err)
		return
	}
	ctx := context.Background()
	trip, err := trip.LoadTripByID(ctx, db, tripID)
	switch {
	case err == sql.ErrNoRows:
		jsonBail(c, http.StatusNotFound, err)
		return
	case err != nil:
		jsonBail(c, http.StatusBadRequest, err)
		return
	}
	c.JSON(http.StatusOK, trip.Expenses)
}

// getSettlement returns a settlement object for the trip
func getSettlement(c *gin.Context, db *sql.DB) {
	tripID, err := strconv.ParseInt(c.Params.ByName("trip_id"), 10, 64)
	if err != nil {
		jsonBail(c, http.StatusBadRequest, err)
		return
	}
	ctx := context.Background()
	trip, err := trip.LoadTripByID(ctx, db, tripID)
	switch {
	case err == sql.ErrNoRows:
		jsonBail(c, http.StatusNotFound, err)
		return
	case err != nil:
		jsonBail(c, http.StatusBadRequest, err)
		return
	}
	settlement, err := trip.Complete(ctx, db)
	if err != nil {
		jsonBail(c, http.StatusBadRequest, err)
		return
	}
	c.JSON(http.StatusOK, settlement)
}

func main() {
	flag.Parse()
	dbU, err := url.Parse(dbURL)
	if err != nil {
		log.Fatalf("ERROR: failed to parse database URL: %q: %v", dbURL, err)
	}
	if dbU.Scheme != "sqlite3" {
		log.Fatalf("ERROR: unsupported database: %s", dbU.Scheme)
	}

	db, err := sql.Open(dbU.Scheme, dbU.Path)
	if err != nil {
		log.Fatalf("ERROR: failed to open DB file %q: %v", dbU.Path, err)
	}
	log.Printf("Opened DB file at %s\n", dbU.Path)
	defer db.Close()

	// we don't really use floating point numbers in any JSON doc
	gin.EnableJsonDecoderUseNumber()

	router := gin.Default()
	router.POST("/trips", handlerWrapper(db, postTrip))
	router.GET("/:owner/trips", handlerWrapper(db, getTrips))
	router.POST("/trips/:trip_id/expenses", handlerWrapper(db, postExpense))
	router.GET("/trips/:trip_id/expenses", handlerWrapper(db, getExpenses))
	router.GET("/trips/:trip_id/settlement", handlerWrapper(db, getSettlement))

	bindAddr := fmt.Sprintf(":%d", port)
	router.Run(bindAddr)
}
