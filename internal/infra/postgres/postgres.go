package postgres

import (
	"database/sql"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/kenziehh/cashflow-be/config"
)

func InitDB(cfg *config.Config) *sql.DB {
	dsn := fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=disable",
		cfg.DBHost, cfg.DBPort, cfg.DBUser, cfg.DBPassword, cfg.DBName)

	db, err := sql.Open("postgres", dsn)
	if err != nil {
		log.Fatal("Failed to connect to database:", err)
	}

	db.SetMaxOpenConns(25)
	db.SetMaxIdleConns(5)
	db.SetConnMaxLifetime(5 * time.Minute)

	if err := db.Ping(); err != nil {
		log.Fatal("Failed to ping database:", err)
	}

	// Run migrations
	RunMigrations(db)

	log.Println("Database connected successfully")
	return db
}

func RunMigrations(db *sql.DB) {
    const migrationsDir = "database/migrations"

    ensureMigrationsTable(db)
    migrations := loadMigrationFiles(migrationsDir)

    applyPendingMigrations(db, migrations)

    log.Println("✅ Custom migrations completed")
}

func ensureMigrationsTable(db *sql.DB) {
    _, err := db.Exec(`
        CREATE TABLE IF NOT EXISTS app_schema_migrations (
            version   INT PRIMARY KEY,
            applied_at TIMESTAMP NOT NULL DEFAULT NOW()
        );
    `)
    if err != nil {
        log.Fatalf("failed to ensure schema_migrations table: %v", err)
    }
}

type migrationFile struct {
    Version int
    Name    string
    Path    string
}

func loadMigrationFiles(dir string) []migrationFile {
    files, err := ioutil.ReadDir(dir)
    if err != nil {
        log.Fatalf("failed to read migrations dir: %v", err)
    }

    var migrations []migrationFile
    for _, f := range files {
        mf, ok := toMigrationFile(dir, f)
        if !ok {
            continue
        }
        migrations = append(migrations, mf)
    }

    sort.Slice(migrations, func(i, j int) bool {
        return migrations[i].Version < migrations[j].Version
    })

    return migrations
}

func toMigrationFile(dir string, f os.FileInfo) (migrationFile, bool) {
    if f.IsDir() || !strings.HasSuffix(f.Name(), ".sql") {
        return migrationFile{}, false
    }

    parts := strings.SplitN(f.Name(), "_", 2)
    if len(parts) < 2 {
        log.Printf("skip file without version prefix: %s", f.Name())
        return migrationFile{}, false
    }

    v, err := strconv.Atoi(parts[0])
    if err != nil {
        log.Printf("skip file with invalid version prefix: %s", f.Name())
        return migrationFile{}, false
    }

    return migrationFile{
        Version: v,
        Name:    f.Name(),
        Path:    filepath.Join(dir, f.Name()),
    }, true
}

func applyPendingMigrations(db *sql.DB, migrations []migrationFile) {
    for _, m := range migrations {
        if migrationAlreadyApplied(db, m.Version) {
            log.Printf("migration %03d already applied, skipping (%s)", m.Version, m.Name)
            continue
        }
        applyMigration(db, m)
    }
}

func migrationAlreadyApplied(db *sql.DB, version int) bool {
    var exists bool
    err := db.QueryRow(
        `SELECT EXISTS(SELECT 1 FROM app_schema_migrations WHERE version = $1)`,
        version,
    ).Scan(&exists)
    if err != nil {
        log.Fatalf("failed to check migration %d: %v", version, err)
    }
    return exists
}

func applyMigration(db *sql.DB, m migrationFile) {
    content, err := ioutil.ReadFile(m.Path)
    if err != nil {
        log.Fatalf("failed to read migration file %s: %v", m.Path, err)
    }

    if _, err := db.Exec(string(content)); err != nil {
        log.Fatalf("failed to execute migration %s: %v", m.Name, err)
    }

    if _, err := db.Exec(
        `INSERT INTO app_schema_migrations (version) VALUES ($1)`,
        m.Version,
    ); err != nil {
        log.Fatalf("failed to record migration %d: %v", m.Version, err)
    }
}
