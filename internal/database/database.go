package database

import (
	"context"
	"fmt"
	"log"
	"os"
	"sync"

	"github.com/jackc/pgx/v5/pgxpool"
	_ "github.com/joho/godotenv/autoload"
)

// Service represents a service that interacts with a database.
type Service interface {
	// Close terminates the database connection.
	// It returns an error if the connection cannot be closed.
	Close()

	CreateUser(ctx context.Context, params RegisterParams) (*User, error)

	GetUser(ctx context.Context, id int) (*User, error)

	GetUserByName(ctx context.Context, name string) (*User, error)

	GetUserByEmail(ctx context.Context, email string) (*User, error)

	ValidateToken(token string) bool
}

type service struct {
	db *pgxpool.Pool
}

var (
	database   = os.Getenv("BLUEPRINT_DB_DATABASE")
	password   = os.Getenv("BLUEPRINT_DB_PASSWORD")
	username   = os.Getenv("BLUEPRINT_DB_USERNAME")
	port       = os.Getenv("BLUEPRINT_DB_PORT")
	host       = os.Getenv("BLUEPRINT_DB_HOST")
	schema     = os.Getenv("BLUEPRINT_DB_SCHEMA")
	dbInstance *service
	once       sync.Once
)

func New() Service {
	once.Do(func() {
		log.Println("Initializing database...")
		connStr := fmt.Sprintf("postgres://%s:%s@%s:%s/%s?sslmode=require&search_path=%s", username, password, host, port, database, schema)
		db, err := pgxpool.New(context.Background(), connStr)
		if err != nil {
			log.Fatal(err)
		}

		dbInstance = &service{
			db: db,
		}

		log.Println("Database initialized.")
	})

	return dbInstance
}

// Close closes the database connection.
// It logs a message indicating the disconnection from the specific database.
// If the connection is successfully closed, it returns nil.
// If an error occurs while closing the connection, it returns the error.
func (s *service) Close() {
	log.Printf("Disconnected from database: %s", database)
	s.db.Close()
}
