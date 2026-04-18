package database

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"net/url"
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

	// Admin
	CreateUser(ctx context.Context, params RegisterParams) (*User, error)
	GetUser(ctx context.Context, id int) (*User, error)
	GetUserByName(ctx context.Context, name string) (*User, error)
	GetUserByEmail(ctx context.Context, email string) (*User, error)

	// Bucket
	GetBucketByName(ctx context.Context, name string) (*Bucket, error)
	CreateBucket(ctx context.Context, name string, ownerId int, region string) (*Bucket, error)

	// Auth
	ValidatePresignedUrl(ctx context.Context, query url.Values) (*User, *ApiKey, *AuthError)
	ValidateHeaderAuth(ctx context.Context, r *http.Request) (*User, *ApiKey, *AuthError)
}

type service struct {
	db *pgxpool.Pool
}

var (
	database   = os.Getenv("DB_DATABASE")
	password   = os.Getenv("DB_PASSWORD")
	username   = os.Getenv("DB_USERNAME")
	port       = os.Getenv("DB_PORT")
	host       = os.Getenv("DB_HOST")
	schema     = os.Getenv("DB_SCHEMA")
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
