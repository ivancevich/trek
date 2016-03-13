package trek

import (
	"database/sql"
	"errors"
	"reflect"
	"sort"
)

var (
	migrations                   []migration
	errNoRows                    = errors.New("sql: no rows in result set")
	errUnrecognizedDatabase      = errors.New("trek: unrecognized database")
	errUnrecognizedAction        = errors.New("trek: unrecognized action")
	errPreviousMigrationNotFound = errors.New("trek: previous migration not found")
	errVersionAlreadyRegistered  = errors.New("Version already registered")
)

const (
	// UP migrates the database to the latest version
	UP = "up"
	// DOWN migrates the database to a previous version
	DOWN = "down"
	// POSTGRES is a supported database
	POSTGRES = "postgres"
	// MYSQL is a supported database
	MYSQL = "mysql"
)

// Register adds migrations to be runned
func Register(version int64, up, down migrationHandler) error {
	if versionAlreadyRegistered(version) {
		return errVersionAlreadyRegistered
	}
	migrations = append(migrations, migration{
		Version: version,
		Up:      up,
		Down:    down,
	})
	return nil
}

// Run executes database migrations
func Run(db *sql.DB, options ...interface{}) (didChange bool, newVersion int64, err error) {
	if len(migrations) == 0 {
		return
	}

	sort.Sort(byVersion(migrations))

	config := parseOptions(options)
	dtbs := &database{db, config}

	err = createTable(dtbs)
	if err != nil {
		return
	}

	oldVersion, err := getVersion(dtbs)
	if err != nil {
		return
	}

	newVersion, err = runMigrations(dtbs, oldVersion)
	didChange = oldVersion != newVersion
	return
}

func parseOptions(options []interface{}) *configuration {
	config := configuration{Action: UP, Database: POSTGRES}

	for _, opt := range options {
		value := reflect.ValueOf(opt)
		if value.Kind() == reflect.String {
			str := value.String()
			switch str {
			case UP:
			case DOWN:
				config.Action = str
				break
			case POSTGRES:
			case MYSQL:
				config.Database = str
				break
			}
		}
	}

	return &config
}

func createTable(db *database) error {
	var query string

	switch db.Database {
	case POSTGRES:
		query = `CREATE TABLE IF NOT EXISTS migrations (id SERIAL PRIMARY KEY, version BIGINT NOT NULL, created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW())`
		break
	case MYSQL:
		query = `CREATE TABLE IF NOT EXISTS migrations (id BIGINT PRIMARY KEY AUTO_INCREMENT, version BIGINT NOT NULL, created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP)`
		break
	default:
		return errUnrecognizedDatabase
	}

	_, err := db.Exec(query)
	return err
}

func getVersion(db *database) (currentVersion int64, err error) {
	err = db.QueryRow(`SELECT version FROM migrations ORDER BY id DESC LIMIT 1`).Scan(&currentVersion)
	if err != nil && err.Error() == errNoRows.Error() {
		currentVersion = 0
		err = nil
	}
	return
}

func runMigrations(db *database, oldVersion int64) (newVersion int64, err error) {
	switch db.Action {
	case UP:
		newVersion, err = runUp(db, oldVersion)
		break
	case DOWN:
		newVersion, err = runDown(db, oldVersion)
		break
	default:
		err = errUnrecognizedAction
	}

	return
}

func runUp(db *database, oldVersion int64) (newVersion int64, err error) {
	for _, m := range migrations {
		if m.Version <= oldVersion {
			continue
		}

		if m.Up != nil {
			err = m.Up(db.DB)
			if err != nil {
				return
			}
		}

		err = setVersion(db, m.Version)
		if err != nil {
			return
		}

		newVersion = m.Version
	}

	return
}

func runDown(db *database, oldVersion int64) (newVersion int64, err error) {
	if oldVersion == 0 {
		return
	}

	var m *migration
	for i := len(migrations) - 1; i >= 0; i-- {
		if migrations[i].Version <= oldVersion {
			m = &migrations[i]
			break
		}
	}

	if m == nil {
		err = errPreviousMigrationNotFound
		return
	}

	if m.Down != nil {
		err = m.Down(db.DB)
		if err != nil {
			return
		}
	}

	newVersion = m.Version - 1
	err = setVersion(db, newVersion)
	return
}

func setVersion(db *database, version int64) error {
	var query string

	switch db.Database {
	case POSTGRES:
		query = `INSERT INTO migrations (version) VALUES ($1)`
		break
	case MYSQL:
		query = `INSERT INTO migrations (version) VALUES (?)`
		break
	default:
		return errUnrecognizedDatabase
	}

	stmt, err := db.Prepare(query)
	if err != nil {
		return err
	}

	_, err = stmt.Exec(version)
	return err
}

func versionAlreadyRegistered(version int64) bool {
	for _, m := range migrations {
		if m.Version == version {
			return true
		}
	}
	return false
}

type migrationHandler func(*sql.DB) error

type configuration struct {
	Action   string
	Database string
}

type database struct {
	*sql.DB
	*configuration
}

type migration struct {
	Version int64
	Up      migrationHandler
	Down    migrationHandler
}

type byVersion []migration

func (s byVersion) Len() int { return len(s) }

func (s byVersion) Swap(i, j int) { s[i], s[j] = s[j], s[i] }

func (s byVersion) Less(i, j int) bool { return s[i].Version < s[j].Version }
