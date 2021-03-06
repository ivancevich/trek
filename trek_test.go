package trek

import (
	"database/sql"
	"errors"
	_ "github.com/go-sql-driver/mysql" // mysql driver
	_ "github.com/lib/pq"              // postgres driver
	"testing"
)

func connect(t *testing.T, kind string) (db *sql.DB) {
	var err error

	var options string
	switch kind {
	case POSTGRES:
		options = "user=postgres dbname=trek sslmode=disable"
		break
	case MYSQL:
		options = "root:@/trek"
		break
	default:
		t.Error("Unsupported database kind")
		return
	}

	db, err = sql.Open(kind, options)
	if err != nil {
		t.Errorf("Error connecting to %s", kind)
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

func teardown(t *testing.T, db *sql.DB) {
	migrations = []migration{}
	if db != nil {
		dropTable(t, db, "migrations")
		db.Close()
	}
}

func TestParseOptionsDefaultValues(t *testing.T) {
	defer teardown(t, nil)
	options := []string{}
	config := parseOptions(options)
	if config.Action != UP {
		t.Error("Action was not UP")
	}
	if config.Database != POSTGRES {
		t.Error("Database was not POSTGRES")
	}
}

func TestParseOptionsCustomValues1(t *testing.T) {
	defer teardown(t, nil)
	options := []string{POSTGRES, UP}
	config := parseOptions(options)
	if config.Action != UP {
		t.Error("Action was not UP")
	}
	if config.Database != POSTGRES {
		t.Error("Database was not POSTGRES")
	}
}

func TestParseOptionsCustomValues2(t *testing.T) {
	defer teardown(t, nil)
	options := []string{MYSQL, DOWN}
	config := parseOptions(options)
	if config.Action != DOWN {
		t.Error("Action was not DOWN")
	}
	if config.Database != MYSQL {
		t.Error("Database was not MYSQL")
	}
}

func TestCreateTableError(t *testing.T) {
	db := connect(t, POSTGRES)
	defer teardown(t, db)
	config := &configuration{Database: "foo", Action: DOWN}
	dtbs := &database{db, config}
	err := createTable(dtbs)
	if err == nil {
		t.Error("Error expected")
	}
	if err != errUnrecognizedDatabase {
		t.Error("Unrecognized database was expected")
	}
}

func TestCreateTablePostgres(t *testing.T) {
	db := connect(t, POSTGRES)
	defer teardown(t, db)
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
}

func TestCreateTableMysql(t *testing.T) {
	db := connect(t, MYSQL)
	defer teardown(t, db)
	config := &configuration{Database: MYSQL, Action: UP}
	dtbs := &database{db, config}
	err := createTable(dtbs)
	if err != nil {
		t.Error(err.Error())
	}
	_, err = db.Query(`SELECT * FROM migrations`)
	if err != nil {
		t.Error(err.Error())
	}
}

func TestGetVersion0(t *testing.T) {
	db := connect(t, POSTGRES)
	defer teardown(t, db)
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
}

func TestGetVersion1(t *testing.T) {
	db := connect(t, POSTGRES)
	defer teardown(t, db)
	config := &configuration{Database: POSTGRES, Action: UP}
	dtbs := &database{db, config}
	err := createTable(dtbs)
	if err != nil {
		t.Error(err.Error())
	}
	_, err = db.Exec(`INSERT INTO migrations (version, running) VALUES (1, false)`)
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
}

func TestGetVersionError(t *testing.T) {
	db := connect(t, POSTGRES)
	defer teardown(t, db)
	config := &configuration{Database: POSTGRES, Action: UP}
	dtbs := &database{db, config}
	err := createTable(dtbs)
	if err != nil {
		t.Error(err.Error())
	}
	_, err = db.Exec(`INSERT INTO migrations (version, running) VALUES (1, true)`)
	if err != nil {
		t.Error(err.Error())
	}
	_, err = getVersion(dtbs)
	if err == nil {
		t.Error("Expected already running migration error")
	}
	if err != errMigrationAlreadyRunning {
		t.Error("Expected already running migration error")
	}
}

func TestRegister(t *testing.T) {
	defer teardown(t, nil)
	var up = func(*sql.DB) error { return nil }
	var down = func(*sql.DB) error { return nil }
	err := Register(1, up, down)
	if err != nil {
		t.Error(err.Error())
	}
	if len(migrations) != 1 {
		t.Error("Expected migrations to have one migration")
	}
	if migrations[0].Version != 1 {
		t.Error("Expected migration to have version equal to 1")
	}
}

func TestRegisterDuplicates(t *testing.T) {
	defer teardown(t, nil)
	var up = func(*sql.DB) error { return nil }
	var down = func(*sql.DB) error { return nil }
	err := Register(1, up, down)
	if err != nil {
		t.Error(err.Error())
	}
	err = Register(1, up, down)
	if err == nil {
		t.Error("Expected duplicate version error")
	}
	if err != errVersionAlreadyRegistered {
		t.Error("Expected duplicate version error")
	}
}

func TestSetVersionError(t *testing.T) {
	db := connect(t, POSTGRES)
	defer teardown(t, db)
	config := &configuration{Database: "foo", Action: UP}
	dtbs := &database{db, config}
	err := setVersion(dtbs, 9, false)
	if err == nil {
		t.Error("Expected unrecognized database error")
	}
	if err != errUnrecognizedDatabase {
		t.Error("Expected unrecognized database error")
	}
}

func TestRunMigrationsError(t *testing.T) {
	db := connect(t, POSTGRES)
	defer teardown(t, db)
	config := &configuration{Database: POSTGRES, Action: "foo"}
	dtbs := &database{db, config}
	newVersion, err := runMigrations(dtbs, 0)
	if newVersion != 0 {
		t.Error("Expected new version to be 0")
	}
	if err == nil {
		t.Error("Expected unrecognized action error")
	}
	if err != errUnrecognizedAction {
		t.Error("Expected unrecognized action error")
	}
}

func TestRunError(t *testing.T) {
	db := connect(t, POSTGRES)
	defer teardown(t, db)
	didChange, newVersion, err := Run(db)
	if didChange {
		t.Error("Expected not to change version")
	}
	if newVersion != 0 {
		t.Error("Expected new version to be 0")
	}
	if err != nil {
		t.Error(err.Error())
	}
}

func TestRunPostgresUp(t *testing.T) {
	db := connect(t, POSTGRES)
	defer teardown(t, db)
	migrations = []migration{
		{
			Version: 1,
			Up:      func(*sql.DB) error { return nil },
			Down:    func(*sql.DB) error { return nil },
		},
	}
	didChange, newVersion, err := Run(db, POSTGRES, UP)
	if !didChange {
		t.Error("Expected to change version")
	}
	if newVersion != 1 {
		t.Error("Expected new version to be 1")
	}
	if err != nil {
		t.Error(err.Error())
	}
}

func TestRunPostgresDown(t *testing.T) {
	db := connect(t, POSTGRES)
	defer teardown(t, db)
	migrations = []migration{
		{
			Version: 1,
			Up:      func(*sql.DB) error { return nil },
			Down:    func(*sql.DB) error { return nil },
		},
	}
	Run(db, POSTGRES, UP) // migrate to version 1
	didChange, newVersion, err := Run(db, POSTGRES, DOWN)
	if !didChange {
		t.Error("Expected to change version")
	}
	if newVersion != 0 {
		t.Error("Expected new version to be 0")
	}
	if err != nil {
		t.Error(err.Error())
	}
}

func TestRunMysqlUp(t *testing.T) {
	db := connect(t, MYSQL)
	defer teardown(t, db)
	migrations = []migration{
		{
			Version: 1,
			Up:      func(*sql.DB) error { return nil },
			Down:    func(*sql.DB) error { return nil },
		},
	}
	didChange, newVersion, err := Run(db, MYSQL, UP)
	if !didChange {
		t.Error("Expected to change version")
	}
	if newVersion != 1 {
		t.Error("Expected new version to be 1")
	}
	if err != nil {
		t.Error(err.Error())
	}
}

func TestRunMysqlDown(t *testing.T) {
	db := connect(t, MYSQL)
	defer teardown(t, db)
	migrations = []migration{
		{
			Version: 1,
			Up:      func(*sql.DB) error { return nil },
			Down:    func(*sql.DB) error { return nil },
		},
	}
	Run(db, MYSQL, UP) // migrate to version 1
	didChange, newVersion, err := Run(db, MYSQL, DOWN)
	if !didChange {
		t.Error("Expected to change version")
	}
	if newVersion != 0 {
		t.Error("Expected new version to be 0")
	}
	if err != nil {
		t.Error(err.Error())
	}
}

func TestRunUpSameVersion(t *testing.T) {
	db := connect(t, POSTGRES)
	defer teardown(t, db)
	migrations = []migration{
		{
			Version: 1,
			Up:      func(*sql.DB) error { return nil },
			Down:    func(*sql.DB) error { return nil },
		},
	}
	Run(db, POSTGRES, UP)                               // migrate to version 1
	didChange, newVersion, err := Run(db, POSTGRES, UP) // migrate to version 1 again
	if didChange {
		t.Error("Expected not to change version")
	}
	if newVersion != 1 {
		t.Error("Expected new version to be 1")
	}
	if err != nil {
		t.Error(err.Error())
	}
}

func TestRunDownSameVersion(t *testing.T) {
	db := connect(t, POSTGRES)
	defer teardown(t, db)
	migrations = []migration{
		{
			Version: 1,
			Up:      func(*sql.DB) error { return nil },
			Down:    func(*sql.DB) error { return nil },
		},
	}
	Run(db, POSTGRES, UP)                                 // migrate to version 1
	Run(db, POSTGRES, DOWN)                               // migrate to version 0
	didChange, newVersion, err := Run(db, POSTGRES, DOWN) // migrate to version 0 again
	if didChange {
		t.Error("Expected not to change version")
	}
	if newVersion != 0 {
		t.Error("Expected new version to be 0")
	}
	if err != nil {
		t.Error(err.Error())
	}
}

func TestRunUpWithError(t *testing.T) {
	db := connect(t, POSTGRES)
	defer teardown(t, db)
	migrations = []migration{
		{
			Version: 1,
			Up:      func(*sql.DB) error { return nil },
			Down:    func(*sql.DB) error { return nil },
		},
		{
			Version: 2,
			Up:      func(*sql.DB) error { return errors.New("Foo") },
			Down:    func(*sql.DB) error { return nil },
		},
	}
	didChange, newVersion, err := Run(db, POSTGRES, UP)
	if !didChange {
		t.Error("Expected to change version")
	}
	if newVersion != 1 {
		t.Error("Expected new version to be 1")
	}
	if err == nil {
		t.Error("Expected 'Foo' error")
	}
	if err.Error() != "Foo" {
		t.Error("Expected 'Foo' error")
	}
}

func TestRunDownWithError(t *testing.T) {
	db := connect(t, POSTGRES)
	defer teardown(t, db)
	migrations = []migration{
		{
			Version: 2,
			Up:      func(*sql.DB) error { return nil },
			Down:    func(*sql.DB) error { return errors.New("Foo") },
		},
		{
			Version: 1,
			Up:      func(*sql.DB) error { return nil },
			Down:    func(*sql.DB) error { return nil },
		},
	}
	Run(db, POSTGRES, UP) // migrate to version 2
	didChange, newVersion, err := Run(db, POSTGRES, DOWN)
	if didChange {
		t.Error("Expected not to change version")
	}
	if newVersion != 2 {
		t.Error("Expected new version to be 2")
	}
	if err == nil {
		t.Error("Expected 'Foo' error")
	}
	if err.Error() != "Foo" {
		t.Error("Expected 'Foo' error")
	}
}

func TestRunDownWithoutMigration(t *testing.T) {
	db := connect(t, POSTGRES)
	defer teardown(t, db)
	migrations = []migration{
		{
			Version: 1,
			Up:      func(*sql.DB) error { return nil },
			Down:    func(*sql.DB) error { return nil },
		},
	}
	Run(db, POSTGRES, UP) // migrate to version 1
	migrations = []migration{
		{
			Version: 2,
			Up:      func(*sql.DB) error { return nil },
			Down:    func(*sql.DB) error { return nil },
		},
	}
	didChange, newVersion, err := Run(db, POSTGRES, DOWN)
	if didChange {
		t.Error("Expected not to change version")
	}
	if newVersion != 1 {
		t.Error("Expected new version to be 1")
	}
	if err == nil {
		t.Error("Expected previous migration error")
	}
	if err != errPreviousMigrationNotFound {
		t.Error("Expected previous migration error")
	}
}
