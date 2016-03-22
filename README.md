# trek
Simple database migrations in Golang

## Start using it

1) Download and install it:

```bash
$ go get github.com/ivancevich/trek
```

2) Import it in your code:

```go
import "github.com/ivancevich/trek"
```

## Register migrations

```go
// migration/1_initial.go

package migration

import (
	"database/sql"
	"github.com/ivancevich/trek"
)

func init() {
	var up = func(db *sql.DB) error {
		_, err := db.Exec(`
			CREATE TABLE foo
			(
				id SERIAL PRIMARY KEY,
				bar VARCHAR(20) NOT NULL,
				baz SMALLINT NOT NULL
			);
		`)
		return err
	}

	var down = func(db *sql.DB) error {
		_, err := db.Exec(`
			DROP TABLE IF EXISTS foo;
		`)
		return err
	}

	trek.Register(1, up, down)
}
```

## Run migrations

```go
// migration/migration.go

package migration

import (
	"database/sql"
	"github.com/ivancevich/trek"
	"log"
)

// Migrate runs database migrations
func Migrate(db *sql.DB) {
	didChange, version, err := trek.Run(db, trek.POSTGRES, trek.UP) // or trek.MYSQL / trek.DOWN
	if err != nil {
		log.Fatalln("Error running migrations", err.Error())
		return
	}
	if didChange {
		log.Printf("The database was migrated to version %d\n", version)
	} else {
		log.Printf("The database was already at version %d\n", version)
	}
}
```
