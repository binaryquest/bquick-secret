package store

import (
	"context"
	"time"
)

func (s *Store) AllowRateLimit(ctx context.Context, key string, limit int, window time.Duration) (bool, error) {
	if limit <= 0 {
		return true, nil
	}
	now := time.Now().UTC()
	resetsAt := now.Add(window)

	_, err := s.pool.Exec(ctx, `
		INSERT INTO rate_limit_buckets (bucket_key, count, resets_at)
		VALUES ($1, 1, $2)
		ON CONFLICT (bucket_key) DO UPDATE
		SET count = CASE
				WHEN rate_limit_buckets.resets_at <= $3 THEN 1
				ELSE rate_limit_buckets.count + 1
			END,
			resets_at = CASE
				WHEN rate_limit_buckets.resets_at <= $3 THEN $2
				ELSE rate_limit_buckets.resets_at
			END
	`, key, resetsAt, now)
	if err != nil {
		return false, err
	}

	var count int
	var currentReset time.Time
	if err := s.pool.QueryRow(ctx, `SELECT count, resets_at FROM rate_limit_buckets WHERE bucket_key = $1`, key).Scan(&count, &currentReset); err != nil {
		return false, err
	}
	return count <= limit || currentReset.Before(now), nil
}
