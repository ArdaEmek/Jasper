package database

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"os"
	"strconv"
	"time"

	_ "github.com/jackc/pgx/v5/stdlib"
	_ "github.com/joho/godotenv/autoload"
)

// Service represents a service that interacts with a database.
type Service interface {
	// Health returns a map of health status information.
	// The keys and values in the map are service-specific.
	Health() map[string]string

	// Close terminates the database connection.
	// It returns an error if the connection cannot be closed.
	Close() error

	// FindFirst retrieves a single row from the database.
	FindFirst(ctx context.Context, query string, args ...interface{}) *sql.Row

	// FindAll retrieves multiple rows from the database.
	FindAll(ctx context.Context, query string, args ...interface{}) (*sql.Rows, error)

	// Create inserts a new record into the database.
	Create(ctx context.Context, query string, args ...interface{}) (sql.Result, error)

	// Update modifies an existing record in the database.
	Update(ctx context.Context, query string, args ...interface{}) (sql.Result, error)

	// Delete removes a record from the database.
	Delete(ctx context.Context, query string, args ...interface{}) (sql.Result, error)
}

type service struct {
	db *sql.DB
}

var (
	database   = os.Getenv("BLUEPRINT_DB_DATABASE")
	password   = os.Getenv("BLUEPRINT_DB_PASSWORD")
	username   = os.Getenv("BLUEPRINT_DB_USERNAME")
	port       = os.Getenv("BLUEPRINT_DB_PORT")
	host       = os.Getenv("BLUEPRINT_DB_HOST")
	schema     = os.Getenv("BLUEPRINT_DB_SCHEMA")
	dbInstance *service
)

func New() Service {
	// Reuse Connection
	if dbInstance != nil {
		return dbInstance
	}
	log.Println("Initializing database...")
	connStr := fmt.Sprintf("postgres://%s:%s@%s:%s/%s?sslmode=disable&search_path=%s", username, password, host, port, database, schema)
	db, err := sql.Open("pgx", connStr)
	if err != nil {
		log.Fatal(err)
	}
	dbInstance = &service{
		db: db,
	}

	log.Println("Database initialized.")
	return dbInstance
}

// Health checks the health of the database connection by pinging the database.
// It returns a map with keys indicating various health statistics.
func (s *service) Health() map[string]string {
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()

	stats := make(map[string]string)

	// Ping the database
	err := s.db.PingContext(ctx)
	if err != nil {
		stats["status"] = "down"
		stats["error"] = fmt.Sprintf("db down: %v", err)
		log.Fatalf("db down: %v", err) // Log the error and terminate the program
		return stats
	}

	// Database is up, add more statistics
	stats["status"] = "up"
	stats["message"] = "It's healthy"

	// Get database stats (like open connections, in use, idle, etc.)
	dbStats := s.db.Stats()
	stats["open_connections"] = strconv.Itoa(dbStats.OpenConnections)
	stats["in_use"] = strconv.Itoa(dbStats.InUse)
	stats["idle"] = strconv.Itoa(dbStats.Idle)
	stats["wait_count"] = strconv.FormatInt(dbStats.WaitCount, 10)
	stats["wait_duration"] = dbStats.WaitDuration.String()
	stats["max_idle_closed"] = strconv.FormatInt(dbStats.MaxIdleClosed, 10)
	stats["max_lifetime_closed"] = strconv.FormatInt(dbStats.MaxLifetimeClosed, 10)

	// Evaluate stats to provide a health message
	if dbStats.OpenConnections > 40 { // Assuming 50 is the max for this example
		stats["message"] = "The database is experiencing heavy load."
	}

	if dbStats.WaitCount > 1000 {
		stats["message"] = "The database has a high number of wait events, indicating potential bottlenecks."
	}

	if dbStats.MaxIdleClosed > int64(dbStats.OpenConnections)/2 {
		stats["message"] = "Many idle connections are being closed, consider revising the connection pool settings."
	}

	if dbStats.MaxLifetimeClosed > int64(dbStats.OpenConnections)/2 {
		stats["message"] = "Many connections are being closed due to max lifetime, consider increasing max lifetime or revising the connection usage pattern."
	}

	return stats
}

// FindFirst retrieves a single row from the database based on a query.
// Always use placeholders ($1, $2, etc.) for parameters to prevent SQL injection.
// Example: FindFirst(ctx, "SELECT * FROM users WHERE id = $1", userID)
func (s *service) FindFirst(ctx context.Context, query string, args ...interface{}) *sql.Row {
	return s.db.QueryRowContext(ctx, query, args...)
}

// FindAll retrieves multiple rows from the database based on a query.
// Always use placeholders ($1, $2, etc.) for parameters to prevent SQL injection.
// Remember to defer rows.Close() after iterating.
// Example: rows, err := FindAll(ctx, "SELECT * FROM users WHERE active = $1", true)
func (s *service) FindAll(ctx context.Context, query string, args ...interface{}) (*sql.Rows, error) {
	return s.db.QueryContext(ctx, query, args...)
}

// Create inserts a new record into the database.
// Always use placeholders ($1, $2, etc.) for parameters to prevent SQL injection.
// Example: result, err := Create(ctx, "INSERT INTO users (name, email) VALUES ($1, $2)", name, email)
func (s *service) Create(ctx context.Context, query string, args ...interface{}) (sql.Result, error) {
	return s.db.ExecContext(ctx, query, args...)
}

// Update modifies an existing record in the database.
// Always use placeholders ($1, $2, etc.) for parameters to prevent SQL injection.
// Example: result, err := Update(ctx, "UPDATE users SET name = $1 WHERE id = $2", newName, userID)
func (s *service) Update(ctx context.Context, query string, args ...interface{}) (sql.Result, error) {
	return s.db.ExecContext(ctx, query, args...)
}

// Delete removes a record from the database.
// Always use placeholders ($1, $2, etc.) for parameters to prevent SQL injection.
// Example: result, err := Delete(ctx, "DELETE FROM users WHERE id = $1", userID)
func (s *service) Delete(ctx context.Context, query string, args ...interface{}) (sql.Result, error) {
	return s.db.ExecContext(ctx, query, args...)
}

// Close closes the database connection.
// It logs a message indicating the disconnection from the specific database.
// If the connection is successfully closed, it returns nil.
// If an error occurs while closing the connection, it returns the error.
func (s *service) Close() error {
	log.Printf("Disconnected from database: %s", database)
	return s.db.Close()
}
