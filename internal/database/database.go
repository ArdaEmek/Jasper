package database

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"sync"

	"github.com/jackc/pgx/v5/pgxpool"
	_ "github.com/joho/godotenv/autoload"
)

// Service represents a service that interacts with a database.
type Service interface {
	// Close safely terminates the database pool and cleans up active connections.
	Close()

	// CreateUser inserts a new user into the database.
	// Returns the created User.
	CreateUser(ctx context.Context, params RegisterParams) (*User, error)
	// GetUser gets the User by their id.
	GetUser(ctx context.Context, id int) (*User, error)
	// GetUserByName gets the User by their username.
	GetUserByName(ctx context.Context, name string) (*User, error)
	// GetUserByEmail gets the User by their email.
	GetUserByEmail(ctx context.Context, email string) (*User, error)

	// GetBucketByName gets the Bucket by its name.
	GetBucketByName(ctx context.Context, name string) (*Bucket, error)
	// CreateBucket inserts a new Bucket into the database.
	// Returns the created Bucket.
	CreateBucket(ctx context.Context, name string, ownerId int, region string) (*Bucket, error)

	// CreateObject registers a newly uploaded file (object) in the database under the specified bucket.
	// It stores the object's metadata, sizing, and custom headers. Returns the created Object.
	CreateObject(ctx context.Context, bucketId int, objectId, objectKey string, sizeBytes int64,
		contentType, etag, contentDisp, contentLang string,
		customMeta map[string]string,
	) (*Object, error)
	GetObjectByKey(ctx context.Context, bucketId int, objectKey string) (*Object, error)
	GetObject(ctx context.Context, objectId string) (*Object, error)

	// ValidatePresignedUrl verifies an S3-compatible presigned URL's expiration and constraints.
	// Returns the authenticated User and their ApiKey, or an AuthError if validation fails.
	ValidatePresignedUrl(ctx context.Context, r *http.Request) (*User, *ApiKey, *AuthError)
	// ValidateHeaderAuth verifies S3-compatible authentication credentials provided in the HTTP request headers.
	// Returns the authenticated User and their ApiKey, or an AuthError if validation fails.
	ValidateHeaderAuth(ctx context.Context, r *http.Request) (*User, *ApiKey, *AuthError)

	// CreateApiKey creates a new API key for the given user ID.
	CreateApiKey(ctx context.Context, userId int, permissionLevel string) (*ApiKey, error)
	// GetApiKeysByUserId gets all API keys for the given user ID.
	GetApiKeysByUserId(ctx context.Context, userId int) ([]ApiKey, error)
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
		connStr := fmt.Sprintf("postgres://%s:%s@%s:%s/%s?sslmode=disable&search_path=%s", username, password, host, port, database, schema)
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
