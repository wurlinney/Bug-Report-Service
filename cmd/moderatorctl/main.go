package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"strings"
	"time"

	"bug-report-service/internal/adapters/config"
	"bug-report-service/internal/adapters/security"

	"github.com/jackc/pgx/v5/pgxpool"
)

func main() {
	var (
		email    = flag.String("email", "", "moderator email")
		password = flag.String("password", "", "moderator password (plain)")
		name     = flag.String("name", "", "display name (optional)")
		dsn      = flag.String("dsn", "", "Postgres DSN (overrides DATABASE_URL)")
	)
	flag.Parse()

	*email = strings.ToLower(strings.TrimSpace(*email))
	*name = strings.TrimSpace(*name)

	if *email == "" || *password == "" {
		_, _ = fmt.Fprintln(os.Stderr, "usage: moderatorctl -email <email> -password <password> [-name <name>]")
		os.Exit(2)
	}

	cfg, cfgErr := config.Load()
	if cfgErr != nil {
		_, _ = fmt.Fprintln(os.Stderr, "config load:", cfgErr)
	}
	dbURL := strings.TrimSpace(*dsn)
	if dbURL == "" {
		dbURL = strings.TrimSpace(os.Getenv("DATABASE_URL"))
	}
	if dbURL == "" {
		// Fallback to whatever config resolved to (can be empty in local).
		dbURL = strings.TrimSpace(cfg.DB.URL)
	}
	if dbURL == "" {
		_, _ = fmt.Fprintln(os.Stderr, "DATABASE_URL is empty (set env or pass -dsn)")
		os.Exit(2)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	pool, err := pgxpool.New(ctx, dbURL)
	if err != nil {
		_, _ = fmt.Fprintln(os.Stderr, "connect db:", err)
		os.Exit(1)
	}
	defer pool.Close()

	hasher := security.NewBCryptPasswordHasher(12)
	hash, err := hasher.HashPassword(*password)
	if err != nil {
		_, _ = fmt.Fprintln(os.Stderr, "hash password:", err)
		os.Exit(1)
	}

	const q = `
INSERT INTO moderators (name, email, password_hash)
VALUES ($1, $2, $3)
ON CONFLICT (email) DO UPDATE SET
  name = EXCLUDED.name,
  password_hash = EXCLUDED.password_hash,
  updated_at = NOW()
`
	if _, err := pool.Exec(ctx, q, *name, *email, hash); err != nil {
		_, _ = fmt.Fprintln(os.Stderr, "upsert moderator:", err)
		os.Exit(1)
	}

	_, _ = fmt.Println("ok")
	_, _ = fmt.Println("email:", *email)
}
