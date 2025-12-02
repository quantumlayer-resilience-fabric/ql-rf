// Package database provides PostgreSQL connection management.
package database

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/quantumlayerhq/ql-rf/pkg/config"
)

// DB wraps a PostgreSQL connection pool.
type DB struct {
	Pool *pgxpool.Pool
}

// New creates a new database connection pool.
func New(ctx context.Context, cfg config.DatabaseConfig) (*DB, error) {
	poolConfig, err := pgxpool.ParseConfig(cfg.URL)
	if err != nil {
		return nil, fmt.Errorf("failed to parse database URL: %w", err)
	}

	// Configure pool settings
	poolConfig.MaxConns = int32(cfg.MaxOpenConns)
	poolConfig.MinConns = int32(cfg.MaxIdleConns)
	poolConfig.MaxConnLifetime = cfg.ConnMaxLifetime
	poolConfig.MaxConnIdleTime = 5 * time.Minute
	poolConfig.HealthCheckPeriod = 1 * time.Minute

	pool, err := pgxpool.NewWithConfig(ctx, poolConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to create connection pool: %w", err)
	}

	// Verify connection
	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	return &DB{Pool: pool}, nil
}

// Close closes the database connection pool.
func (db *DB) Close() {
	if db.Pool != nil {
		db.Pool.Close()
	}
}

// Health checks the database connection health.
func (db *DB) Health(ctx context.Context) error {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	if err := db.Pool.Ping(ctx); err != nil {
		return fmt.Errorf("database ping failed: %w", err)
	}

	return nil
}

// Stats returns connection pool statistics.
func (db *DB) Stats() *pgxpool.Stat {
	return db.Pool.Stat()
}

// Exec executes a query without returning any rows.
func (db *DB) Exec(ctx context.Context, sql string, args ...any) error {
	_, err := db.Pool.Exec(ctx, sql, args...)
	if err != nil {
		return fmt.Errorf("exec failed: %w", err)
	}
	return nil
}

// QueryRow executes a query that returns at most one row.
func (db *DB) QueryRow(ctx context.Context, sql string, args ...any) pgx.Row {
	return db.Pool.QueryRow(ctx, sql, args...)
}

// Query executes a query that returns rows.
func (db *DB) Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error) {
	rows, err := db.Pool.Query(ctx, sql, args...)
	if err != nil {
		return nil, fmt.Errorf("query failed: %w", err)
	}
	return rows, nil
}

// BeginTx starts a transaction with the given options.
func (db *DB) BeginTx(ctx context.Context) (pgx.Tx, error) {
	tx, err := db.Pool.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to begin transaction: %w", err)
	}
	return tx, nil
}

// WithTx executes a function within a transaction.
// If the function returns an error, the transaction is rolled back.
// Otherwise, the transaction is committed.
func (db *DB) WithTx(ctx context.Context, fn func(tx pgx.Tx) error) error {
	tx, err := db.BeginTx(ctx)
	if err != nil {
		return err
	}

	defer func() {
		if p := recover(); p != nil {
			_ = tx.Rollback(ctx)
			panic(p)
		}
	}()

	if err := fn(tx); err != nil {
		if rbErr := tx.Rollback(ctx); rbErr != nil {
			return fmt.Errorf("tx error: %v, rollback error: %w", err, rbErr)
		}
		return err
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	return nil
}

// =============================================================================
// Row-Level Security (RLS) Support
// =============================================================================

// TenantConn represents a database connection with RLS context set.
// It wraps a connection and sets the organization context for RLS policies.
type TenantConn struct {
	conn  *pgxpool.Conn
	orgID uuid.UUID
	db    *DB
}

// AcquireTenantConn acquires a connection from the pool and sets the RLS context.
// The caller must call Release() when done with the connection.
func (db *DB) AcquireTenantConn(ctx context.Context, orgID uuid.UUID) (*TenantConn, error) {
	conn, err := db.Pool.Acquire(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to acquire connection: %w", err)
	}

	// Set the organization context for RLS
	// Using SET LOCAL ensures the setting is transaction-scoped
	_, err = conn.Exec(ctx, "SET LOCAL app.current_org_id = $1", orgID.String())
	if err != nil {
		conn.Release()
		return nil, fmt.Errorf("failed to set RLS context: %w", err)
	}

	return &TenantConn{
		conn:  conn,
		orgID: orgID,
		db:    db,
	}, nil
}

// Release releases the connection back to the pool.
func (tc *TenantConn) Release() {
	if tc.conn != nil {
		tc.conn.Release()
	}
}

// OrgID returns the organization ID for this connection.
func (tc *TenantConn) OrgID() uuid.UUID {
	return tc.orgID
}

// Exec executes a query without returning any rows.
func (tc *TenantConn) Exec(ctx context.Context, sql string, args ...any) error {
	_, err := tc.conn.Exec(ctx, sql, args...)
	if err != nil {
		return fmt.Errorf("exec failed: %w", err)
	}
	return nil
}

// QueryRow executes a query that returns at most one row.
func (tc *TenantConn) QueryRow(ctx context.Context, sql string, args ...any) pgx.Row {
	return tc.conn.QueryRow(ctx, sql, args...)
}

// Query executes a query that returns rows.
func (tc *TenantConn) Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error) {
	rows, err := tc.conn.Query(ctx, sql, args...)
	if err != nil {
		return nil, fmt.Errorf("query failed: %w", err)
	}
	return rows, nil
}

// BeginTx starts a transaction with the RLS context already set.
func (tc *TenantConn) BeginTx(ctx context.Context) (pgx.Tx, error) {
	tx, err := tc.conn.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to begin transaction: %w", err)
	}

	// Ensure RLS context is set within the transaction
	_, err = tx.Exec(ctx, "SET LOCAL app.current_org_id = $1", tc.orgID.String())
	if err != nil {
		_ = tx.Rollback(ctx)
		return nil, fmt.Errorf("failed to set RLS context in transaction: %w", err)
	}

	return tx, nil
}

// WithTx executes a function within a transaction with RLS context.
func (tc *TenantConn) WithTx(ctx context.Context, fn func(tx pgx.Tx) error) error {
	tx, err := tc.BeginTx(ctx)
	if err != nil {
		return err
	}

	defer func() {
		if p := recover(); p != nil {
			_ = tx.Rollback(ctx)
			panic(p)
		}
	}()

	if err := fn(tx); err != nil {
		if rbErr := tx.Rollback(ctx); rbErr != nil {
			return fmt.Errorf("tx error: %v, rollback error: %w", err, rbErr)
		}
		return err
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	return nil
}

// WithTenantContext executes a function with a tenant-scoped connection.
// This is a convenience method that handles acquiring and releasing the connection.
func (db *DB) WithTenantContext(ctx context.Context, orgID uuid.UUID, fn func(tc *TenantConn) error) error {
	tc, err := db.AcquireTenantConn(ctx, orgID)
	if err != nil {
		return err
	}
	defer tc.Release()

	return fn(tc)
}
