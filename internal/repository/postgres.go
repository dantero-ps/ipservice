package repository

import (
	"context"
	"database/sql"
	"errors"
	"net"

	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq"
	"go.uber.org/zap"

	"ipservice/internal/model"
)

type PostgresRepository struct {
	db     *sqlx.DB
	logger *zap.Logger
}

func NewPostgresRepository(db *sqlx.DB, logger *zap.Logger) *PostgresRepository {
	return &PostgresRepository{
		db:     db,
		logger: logger,
	}
}

func (r *PostgresRepository) SaveIPRanges(ctx context.Context, ranges []model.IPRange) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	query := `
        INSERT INTO ip_ranges (network, country_code, ip_version)
        VALUES ($1, $2, $3)
        ON CONFLICT (network)
        DO UPDATE SET 
            country_code = EXCLUDED.country_code,
            ip_version = EXCLUDED.ip_version
    `

	stmt, err := tx.PrepareContext(ctx, query)
	if err != nil {
		return err
	}
	defer stmt.Close()

	for _, ipRange := range ranges {
		_, err = stmt.ExecContext(ctx,
			ipRange.Network.String(),
			ipRange.CountryCode,
			ipRange.Version)
		if err != nil {
			r.logger.Error("failed to insert IP range",
				zap.String("network", ipRange.Network.String()),
				zap.Error(err))
			return err
		}
	}

	return tx.Commit()
}

func (r *PostgresRepository) FindCountryForIP(ctx context.Context, ip net.IP) (string, error) {
	query := `
        SELECT country_code 
        FROM ip_ranges 
        WHERE network >> $1
        ORDER BY network ASC 
        LIMIT 1
    `

	var countryCode string
	err := r.db.GetContext(ctx, &countryCode, query, ip.String())
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return "ZZ", nil // ZZ for unknown/not found
		}

		r.logger.Error("failed to find country for IP",
			zap.String("ip", ip.String()),
			zap.Error(err))
		return "", err
	}

	return countryCode, nil
}

func (r *PostgresRepository) ClearIPRanges(ctx context.Context) error {
	_, err := r.db.ExecContext(ctx, "TRUNCATE TABLE ip_ranges")
	return err
}

func (r *PostgresRepository) GetRangesCount(ctx context.Context) (int64, error) {
	var count int64
	err := r.db.GetContext(ctx, &count, "SELECT count(*) FROM ip_ranges LIMIT 1")
	return count, err
}
