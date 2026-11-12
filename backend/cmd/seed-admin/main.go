// Command seed-admin creates an administrator account.
//
// There is deliberately no default admin and no admin sign-up endpoint:
// someone with access to the server creates the first admin by hand.
//
//	make seed-admin email=admin@example.com name="Admin"
//
// The password is read from the ADMIN_PASSWORD environment variable or, if
// that is empty, from standard input. It is never a flag (flags show up in
// shell history and process lists) and never printed.
package main

import (
	"bufio"
	"context"
	"errors"
	"flag"
	"fmt"
	"log/slog"
	"os"
	"strings"

	"github.com/RavaGhiffari/ecommerceCopyGoReact/backend/internal/config"
	"github.com/RavaGhiffari/ecommerceCopyGoReact/backend/internal/database"
	"github.com/RavaGhiffari/ecommerceCopyGoReact/backend/internal/modules/auth"
	"github.com/RavaGhiffari/ecommerceCopyGoReact/backend/internal/modules/users"
	"github.com/RavaGhiffari/ecommerceCopyGoReact/backend/internal/platform/authctx"
	"github.com/RavaGhiffari/ecommerceCopyGoReact/backend/internal/platform/password"
	"github.com/RavaGhiffari/ecommerceCopyGoReact/backend/internal/platform/validation"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, "seed-admin:", err)
		os.Exit(1)
	}
}

func run() error {
	email := flag.String("email", "", "admin email (required)")
	name := flag.String("name", "", "admin display name (required)")
	flag.Parse()
	if *email == "" || *name == "" {
		flag.Usage()
		return errors.New("-email and -name are required")
	}

	pw := os.Getenv("ADMIN_PASSWORD")
	if pw == "" {
		fmt.Fprint(os.Stderr, "Admin password (input is visible): ")
		line, err := bufio.NewReader(os.Stdin).ReadString('\n')
		if err != nil && line == "" {
			return fmt.Errorf("read password: %w", err)
		}
		pw = strings.TrimRight(line, "\r\n")
	}

	cfg, err := config.Load()
	if err != nil {
		return err
	}
	slog.SetLogLoggerLevel(slog.LevelWarn)

	ctx := context.Background()
	pool, err := database.NewPool(ctx, cfg.DB)
	if err != nil {
		return err
	}
	defer pool.Close()

	svc, err := auth.NewService(pool, auth.Config{
		SessionTTL: cfg.SessionTTL,
		Password: password.Params{
			MemoryKiB:   cfg.Argon2.MemoryKiB,
			Iterations:  cfg.Argon2.Iterations,
			Parallelism: cfg.Argon2.Parallelism,
		},
	})
	if err != nil {
		return err
	}

	u, err := svc.CreateUserWithRole(ctx, auth.RegisterInput{Name: *name, Email: *email, Password: pw}, authctx.RoleAdmin)
	var fields validation.Errors
	switch {
	case errors.As(err, &fields):
		for field, msg := range fields {
			fmt.Fprintf(os.Stderr, "  %s %s\n", field, msg)
		}
		return errors.New("invalid input")
	case errors.Is(err, users.ErrEmailTaken):
		return fmt.Errorf("a user with email %s already exists", users.NormalizeEmail(*email))
	case err != nil:
		return err
	}

	fmt.Printf("admin created: %s (%s)\n", u.Email, u.ID)
	return nil
}
