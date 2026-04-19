package server

import (
	"crypto/tls"
	"fmt"
	"net/http"
	"os"
	"s3/internal/filesystem"
	"s3/internal/server/utils"
	"strconv"
	"time"

	_ "github.com/joho/godotenv/autoload"
	"github.com/quic-go/quic-go"
	"github.com/quic-go/quic-go/http3"

	"s3/internal/database"
)

type Server struct {
	port int
	db   database.Service
	fm   filesystem.FileManager

	http3Server *http3.Server
}

func NewServer() (*http.Server, *http3.Server) {
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

	cert := os.Getenv("CERTIFICATE_PUBLIC_KEY")
	key := os.Getenv("CERTIFICATE_PRIVATE_KEY")
	tlsCert, err := tls.LoadX509KeyPair(cert, key)
	if err != nil {
		panic(err)
	}

	tlsConfig := &tls.Config{
		InsecureSkipVerify: utils.IsDev,
		MinVersion:         tls.VersionTLS13,
		Certificates:       []tls.Certificate{tlsCert},
	}

	// HTTP1 Server
	httpServer := &http.Server{
		Addr:         fmt.Sprintf(":%d", NewServer.port),
		Handler:      NewServer.RegisterRoutes(),
		IdleTimeout:  time.Minute,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 30 * time.Second,
		TLSConfig:    tlsConfig,
	}

	http3Server := &http3.Server{
		Addr:       fmt.Sprintf(":%d", NewServer.port),
		Handler:    NewServer.RegisterRoutes(),
		TLSConfig:  http3.ConfigureTLSConfig(tlsConfig),
		QUICConfig: &quic.Config{},
	}

	NewServer.http3Server = http3Server

	return httpServer, http3Server
}
