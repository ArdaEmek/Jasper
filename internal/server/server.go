package server

import (
	"fmt"
	"net/http"
	"os"
	"s3/internal/filesystem"
	"strconv"
	"time"

	_ "github.com/joho/godotenv/autoload"

	"s3/internal/database"
)

type Server struct {
	port int

	db database.Service
	fm filesystem.FileManager
}

func NewServer() *http.Server {
	port, _ := strconv.Atoi(os.Getenv("PORT"))
	fm, err := filesystem.New(os.Getenv("STORAGE_DIR"))
	if err != nil {
		panic(err)
	}

	NewServer := &Server{
		port: port,
		fm:   fm,

		db: database.New(),
	}

	// Declare Server config
	server := &http.Server{
		Addr:         fmt.Sprintf(":%d", NewServer.port),
		Handler:      NewServer.RegisterRoutes(),
		IdleTimeout:  time.Minute,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 30 * time.Second,
	}

	return server
}
