package storage

import (
	"coinStatApp/internal/domain/model"
	"coinStatApp/internal/domain/repository"
	"context"
	"fmt"
	"time"

	"github.com/ClickHouse/clickhouse-go/v2"
	"github.com/ClickHouse/clickhouse-go/v2/lib/driver"
)

// ClickHouseRepository implements both StatisticsPersistence and EventPersistence interfaces
// using ClickHouse as the backend database. It provides durable, analytical storage for
// both aggregated statistics and individual swap events.
type ClickHouseRepository struct {
	conn driver.Conn
}

type ClickHouseConfig struct {
	Addr     string
	Username string
	Password string
	Timeout  int
}

func NewClickHouseRepository(cfg ClickHouseConfig) (*ClickHouseRepository, error) {
	conn, err := clickhouse.Open(&clickhouse.Options{
		Addr: []string{cfg.Addr},
		Auth: clickhouse.Auth{
			Database: "default",
			Username: cfg.Username,
			Password: cfg.Password,
		},
		DialTimeout: time.Duration(cfg.Timeout) * time.Second,
		Compression: &clickhouse.Compression{
			Method: clickhouse.CompressionLZ4,
		},
	})
	if err != nil {
		return nil, err
	}

	// Check the connection
	if err := conn.Ping(context.Background()); err != nil {
		return nil, fmt.Errorf("failed to ping ClickHouse: %w", err)
	}

	// Ensure tables exist
	if err := createTablesIfNotExist(conn); err != nil {
		return nil, fmt.Errorf("failed to create tables: %w", err)
	}

	return &ClickHouseRepository{conn: conn}, nil
}

// Ensure ClickHouseRepository implements both required interfaces
var _ repository.StatisticsPersistence = (*ClickHouseRepository)(nil)
var _ repository.EventPersistence = (*ClickHouseRepository)(nil)

func createTablesIfNotExist(conn driver.Conn) error {
	// Create swap events table
	err := conn.Exec(context.Background(), `
		CREATE TABLE IF NOT EXISTS swap_events (
			id String,
			token String,
			amount Float64,
			usd Float64,
			side String,
			who String,
			timestamp DateTime,
			processed_at DateTime DEFAULT now()
		) ENGINE = MergeTree()
		ORDER BY (token, timestamp)
	`)
	if err != nil {
		return err
	}

	// Create statistics table
	err = conn.Exec(context.Background(), `
		CREATE TABLE IF NOT EXISTS token_statistics (
			token String,
			volume_5m Float64,
			volume_1h Float64,
			volume_24h Float64,
			tx_count_5m UInt32,
			tx_count_1h UInt32,
			tx_count_24h UInt32,
			updated_at DateTime DEFAULT now()
		) ENGINE = ReplacingMergeTree()
		ORDER BY (token, updated_at)
	`)

	return err
}

// SwapStorage interface implementation

// SaveSwap saves a swap event to ClickHouse
func (r *ClickHouseRepository) SaveSwap(ctx context.Context, swap *model.Swap) error {
	query := `
		INSERT INTO swap_events (
			id, token, amount, usd, side, who, timestamp
		) VALUES (
			?, ?, ?, ?, ?, ?, ?
		)
	`

	return r.conn.AsyncInsert(ctx, query, false,
		swap.ID,
		swap.Token,
		swap.Amount,
		swap.USD,
		swap.Side,
		swap.Who,
		swap.Timestamp,
	)
}

// GetSwapsSince retrieves all swap events after the given timestamp
func (r *ClickHouseRepository) GetSwapsSince(ctx context.Context, since int64) ([]*model.Swap, error) {
	query := `
		SELECT id, token, amount, usd, side, who, timestamp
		FROM swap_events
		WHERE timestamp >= fromUnixTimestamp(?)
		ORDER BY timestamp
	`

	rows, err := r.conn.Query(ctx, query, since)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var results []*model.Swap
	for rows.Next() {
		swap := new(model.Swap)
		if err := rows.Scan(
			&swap.ID,
			&swap.Token,
			&swap.Amount,
			&swap.USD,
			&swap.Side,
			&swap.Who,
			&swap.Timestamp,
		); err != nil {
			return nil, err
		}
		results = append(results, swap)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return results, nil
}

// StatisticsStorage interface implementation

// SaveStatistics saves token statistics to ClickHouse
func (r *ClickHouseRepository) SaveStatistics(ctx context.Context, stats *model.Statistics) error {
	query := `
		INSERT INTO token_statistics (
			token, volume_5m, volume_1h, volume_24h, 
			tx_count_5m, tx_count_1h, tx_count_24h
		) VALUES (
			?, ?, ?, ?, ?, ?, ?
		)
	`

	err := r.conn.AsyncInsert(ctx, query, false,
		stats.Token,
		stats.Volume5Min,
		stats.Volume1H,
		stats.Volume24H,
		stats.Count5Min,
		stats.Count1H,
		stats.Count24H,
	)

	return err
}

// GetStatistics retrieves statistics for a specific token
func (r *ClickHouseRepository) GetStatistics(ctx context.Context, token string) (*model.Statistics, error) {
	query := `
		SELECT 
			token, volume_5m, volume_1h, volume_24h, 
			tx_count_5m, tx_count_1h, tx_count_24h
		FROM token_statistics
		WHERE token = ?
		ORDER BY updated_at DESC
		LIMIT 1
	`

	var stats model.Statistics
	row := r.conn.QueryRow(ctx, query, token)
	err := row.Scan(
		&stats.Token,
		&stats.Volume5Min,
		&stats.Volume1H,
		&stats.Volume24H,
		&stats.Count5Min,
		&stats.Count1H,
		&stats.Count24H,
	)

	if err != nil {
		return nil, err
	}

	return &stats, nil
}

// GetAllStatistics retrieves statistics for all tokens
func (r *ClickHouseRepository) GetAllStatistics(ctx context.Context) ([]*model.Statistics, error) {
	query := `
		WITH latest_stats AS (
			SELECT 
				token,
				MAX(updated_at) as latest_update
			FROM token_statistics
			GROUP BY token
		)
		SELECT 
			ts.token, ts.volume_5m, ts.volume_1h, ts.volume_24h, 
			ts.tx_count_5m, ts.tx_count_1h, ts.tx_count_24h
		FROM token_statistics ts
		INNER JOIN latest_stats ls 
		ON ts.token = ls.token AND ts.updated_at = ls.latest_update
		ORDER BY ts.token
	`

	rows, err := r.conn.Query(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var results []*model.Statistics
	for rows.Next() {
		stats := new(model.Statistics)
		if err := rows.Scan(
			&stats.Token,
			&stats.Volume5Min,
			&stats.Volume1H,
			&stats.Volume24H,
			&stats.Count5Min,
			&stats.Count1H,
			&stats.Count24H,
		); err != nil {
			return nil, err
		}
		results = append(results, stats)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return results, nil
}
