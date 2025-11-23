package main

import (
	"log"
	"net/http"
	"os"
	"time"

	"github.com/Guardian1221/prsvc/internal/api"
	"github.com/Guardian1221/prsvc/internal/repo"
	"github.com/Guardian1221/prsvc/internal/service"
)

func main() {
	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		dsn = "postgres://prusr:prpwd@localhost:5432/prsvc?sslmode=disable"
	}
	r, err := repo.NewPostgresRepo(dsn)
	if err != nil {
		log.Fatalf("db connect failed: %v", err)
	}
	defer r.Close()

	svc := service.NewService(r)
	h := api.NewHandler(svc)

	srv := &http.Server{
		Addr:         ":8080",
		Handler:      h,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	log.Printf("listening on %s", srv.Addr)
	if err := srv.ListenAndServe(); err != nil {
		log.Fatalf("server failed: %v", err)
	}
}
