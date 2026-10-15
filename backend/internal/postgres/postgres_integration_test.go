//go:build integration

package postgres_test

import (
	"context"
	"fmt"
	"os"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"
	tcpostgres "github.com/testcontainers/testcontainers-go/modules/postgres"

	"github.com/AlisherGulomzoda/family-expenses-ai-bot/backend/internal/expense/expensetest"
	"github.com/AlisherGulomzoda/family-expenses-ai-bot/backend/internal/postgres"
)

// Integration tests run against a real PostgreSQL in Docker:
//
//	go test -tags integration ./internal/postgres/...

var (
	pool        *pgxpool.Pool
	databaseURL string
)

func TestMain(m *testing.M) {
	os.Exit(runMain(m))
}

func runMain(m *testing.M) int {
	ctx := context.Background()
	container, err := tcpostgres.Run(ctx, "postgres:17-alpine",
		tcpostgres.WithDatabase("expenses"),
		tcpostgres.WithUsername("expenses"),
		tcpostgres.WithPassword("expenses"),
		tcpostgres.BasicWaitStrategies(),
	)
	if err != nil {
		fmt.Fprintln(os.Stderr, "start postgres:", err)
		return 1
	}
	defer func() { _ = container.Terminate(ctx) }()

	databaseURL, err = container.ConnectionString(ctx, "sslmode=disable")
	if err != nil {
		fmt.Fprintln(os.Stderr, "connection string:", err)
		return 1
	}
	if err := migrateUp(ctx); err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}
	pool, err = postgres.Connect(ctx, databaseURL)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}
	defer pool.Close()

	return m.Run()
}

func migrateUp(ctx context.Context) error {
	db, err := postgres.OpenSQL(databaseURL)
	if err != nil {
		return err
	}
	defer db.Close()
	migrator, err := postgres.NewMigrator(db)
	if err != nil {
		return err
	}
	if _, err := migrator.Up(ctx); err != nil {
		return fmt.Errorf("migrate up: %w", err)
	}
	return nil
}

func truncate(t *testing.T) {
	t.Helper()
	_, err := pool.Exec(context.Background(), "TRUNCATE chat_messages, expenses, users RESTART IDENTITY CASCADE")
	if err != nil {
		t.Fatalf("truncate: %v", err)
	}
}

func TestRepositoryContract(t *testing.T) {
	expensetest.RunRepositoryContract(t, func(t *testing.T) expensetest.Repos {
		truncate(t)
		return expensetest.Repos{
			Expenses: postgres.NewExpenseRepo(pool),
			Users:    postgres.NewUserRepo(pool),
		}
	})
}

func TestMigrationsRoundTrip(t *testing.T) {
	ctx := context.Background()
	db, err := postgres.OpenSQL(databaseURL)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	migrator, err := postgres.NewMigrator(db)
	if err != nil {
		t.Fatal(err)
	}

	if _, err := migrator.DownTo(ctx, 0); err != nil {
		t.Fatalf("reset: %v", err)
	}
	var tables int
	if err := db.QueryRowContext(ctx,
		`SELECT count(*) FROM information_schema.tables
		 WHERE table_schema = 'public' AND table_name IN ('users', 'expenses', 'chat_messages')`,
	).Scan(&tables); err != nil {
		t.Fatal(err)
	}
	if tables != 0 {
		t.Errorf("%d tables left after reset", tables)
	}

	if _, err := migrator.Up(ctx); err != nil {
		t.Fatalf("up after reset: %v", err)
	}
	version, err := migrator.GetDBVersion(ctx)
	if err != nil || version != 3 {
		t.Errorf("schema version = %d, %v; want 3", version, err)
	}
}
