package trek

import (
	"database/sql"
	_ "github.com/lib/pq" // postgres driver
	"strings"
	"testing"
)

func connect(t *testing.T) (db *sql.DB) {
	parts := []string{"user=postgres", "dbname=trek", "sslmode=disable"}

	var err error

	db, err = sql.Open(POSTGRES, strings.Join(parts, " "))
	if err != nil {
		t.Error("Error connecting to Postgres")
		return
	}

	err = db.Ping()
	if err != nil {
		t.Error("Error pinging the database")
		return
	}

	return
}

func dropTable(t *testing.T, db *sql.DB, table string) {
	_, err := db.Exec(`DROP TABLE IF EXISTS ` + table)
	if err != nil {
		t.Error(err.Error())
	}
}

func TestParseOptionsDefaultValues(t *testing.T) {
	options := []interface{}{}
	config := parseOptions(options)
	if config.Action != UP {
		t.Error("Action was not UP")
	}
	if config.Database != POSTGRES {
		t.Error("Database was not POSTGRES")
	}
}

func TestParseOptionsCustomValues1(t *testing.T) {
	options := []interface{}{POSTGRES, UP}
	config := parseOptions(options)
	if config.Action != UP {
		t.Error("Action was not UP")
	}
	if config.Database != POSTGRES {
		t.Error("Database was not POSTGRES")
	}
}

func TestParseOptionsCustomValues2(t *testing.T) {
	options := []interface{}{MYSQL, DOWN}
	config := parseOptions(options)
	if config.Action != DOWN {
		t.Error("Action was not DOWN")
	}
	if config.Database != MYSQL {
		t.Error("Database was not MYSQL")
	}
}

func TestCreateTableError(t *testing.T) {
	db := connect(t)
	defer db.Close()
	config := &configuration{Database: "foo", Action: DOWN}
	dtbs := &database{db, config}
	err := createTable(dtbs)
	if err == nil {
		t.Error("Error expected")
	}
	if err.Error() != errUnrecognizedDatabase.Error() {
		t.Error("Unrecognized database was expected")
	}
}

func TestCreateTablePostgres(t *testing.T) {
	db := connect(t)
	defer db.Close()
	config := &configuration{Database: POSTGRES, Action: UP}
	dtbs := &database{db, config}
	err := createTable(dtbs)
	if err != nil {
		t.Error(err.Error())
	}
	_, err = db.Query(`SELECT * FROM migrations`)
	if err != nil {
		t.Error(err.Error())
	}
	dropTable(t, db, "migrations")
}

func TestGetVersion0(t *testing.T) {
	db := connect(t)
	defer db.Close()
	config := &configuration{Database: POSTGRES, Action: UP}
	dtbs := &database{db, config}
	err := createTable(dtbs)
	if err != nil {
		t.Error(err.Error())
	}
	currentVersion, err := getVersion(dtbs)
	if err != nil {
		t.Error(err.Error())
	}
	if currentVersion != 0 {
		t.Error("Expected version different from 0")
	}
	dropTable(t, db, "migrations")
}

func TestGetVersion1(t *testing.T) {
	db := connect(t)
	defer db.Close()
	config := &configuration{Database: POSTGRES, Action: UP}
	dtbs := &database{db, config}
	err := createTable(dtbs)
	if err != nil {
		t.Error(err.Error())
	}
	_, err = db.Exec(`INSERT INTO migrations (version) VALUES (1)`)
	if err != nil {
		t.Error(err.Error())
	}
	currentVersion, err := getVersion(dtbs)
	if err != nil {
		t.Error(err.Error())
	}
	if currentVersion != 1 {
		t.Error("Expected version different from 1")
	}
	dropTable(t, db, "migrations")
}

