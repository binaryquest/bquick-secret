package store

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

var ErrNotFound = errors.New("not found")

type Store struct {
	pool *pgxpool.Pool
}

type CreateSecretParams struct {
	PublicID               string
	EncryptedPayload       []byte
	IV                     []byte
	Algorithm              string
	Version                int
	ExpiresAt              time.Time
	OneTime                bool
	SenderEmailHash        string
	RecipientEmailProvided bool
	ManualLinkEnabled      bool
	PassphraseEnabled      bool
	NotifySenderOnReveal   bool
	SenderNotifyEmail      string
	RevealTokenHash        string
	DeleteTokenHash        string
	PayloadSizeBytes       int
	WrappedKey             []byte
	WrappingIV             []byte
	KDFSalt                []byte
	KDFIterations          int
	KDFAlgorithm           string
}

type SecretPayload struct {
	EncryptedPayload  []byte `json:"-"`
	IV                []byte `json:"-"`
	Algorithm         string `json:"algorithm"`
	Version           int    `json:"version"`
	OneTime           bool   `json:"oneTime"`
	PassphraseEnabled bool   `json:"passphraseEnabled"`
	WrappedKey        []byte `json:"-"`
	WrappingIV        []byte `json:"-"`
	KDFSalt           []byte `json:"-"`
	KDFIterations     int    `json:"kdfIterations,omitempty"`
	KDFAlgorithm      string `json:"kdfAlgorithm,omitempty"`
}

type DailyStats struct {
	Date                    time.Time `json:"date"`
	SecretsCreatedCount     int64     `json:"secretsCreatedCount"`
	SecretsOpenedCount      int64     `json:"secretsOpenedCount"`
	SecretsExpiredCount     int64     `json:"secretsExpiredCount"`
	SecretsDeletedCount     int64     `json:"secretsDeletedCount"`
	EmailsSentCount         int64     `json:"emailsSentCount"`
	ManualLinksCreatedCount int64     `json:"manualLinksCreatedCount"`
	PassphraseEnabledCount  int64     `json:"passphraseEnabledCount"`
	OneTimeEnabledCount     int64     `json:"oneTimeEnabledCount"`
	FilesUploadedCount      int64     `json:"filesUploadedCount"`
	TotalEncryptedFileBytes int64     `json:"totalEncryptedFileBytes"`
}

func Open(ctx context.Context, databaseURL string) (*Store, error) {
	pool, err := pgxpool.New(ctx, databaseURL)
	if err != nil {
		return nil, err
	}
	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		return nil, err
	}
	store := &Store{pool: pool}
	if err := store.EnsureSchema(ctx); err != nil {
		pool.Close()
		return nil, err
	}
	return store, nil
}

func (s *Store) Close() {
	s.pool.Close()
}

func (s *Store) EnsureSchema(ctx context.Context) error {
	_, err := s.pool.Exec(ctx, schemaSQL)
	return err
}

func (s *Store) CreateSecret(ctx context.Context, params CreateSecretParams) error {
	_, err := s.pool.Exec(ctx, `
		INSERT INTO secrets (
			public_id, encrypted_payload, iv, algorithm, version, expires_at, one_time,
			sender_email_hash, recipient_email_provided, manual_link_enabled, passphrase_enabled,
			notify_sender_on_reveal, sender_notify_email, reveal_token_hash, delete_token_hash, payload_size_bytes,
			wrapped_key, wrapping_iv, kdf_salt, kdf_iterations, kdf_algorithm
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7,
			$8, $9, $10, $11,
			$12, $13, $14, $15, $16,
			$17, $18, $19, $20, $21
		)
	`, params.PublicID, params.EncryptedPayload, params.IV, params.Algorithm, params.Version, params.ExpiresAt, params.OneTime,
		params.SenderEmailHash, params.RecipientEmailProvided, params.ManualLinkEnabled, params.PassphraseEnabled,
		params.NotifySenderOnReveal, nilIfEmptyString(params.SenderNotifyEmail), nilIfEmptyString(params.RevealTokenHash),
		params.DeleteTokenHash, params.PayloadSizeBytes,
		nilIfEmpty(params.WrappedKey), nilIfEmpty(params.WrappingIV),
		nilIfEmpty(params.KDFSalt), nilIfZero(params.KDFIterations), nilIfEmptyString(params.KDFAlgorithm))
	return err
}

func (s *Store) GetSecretForOpen(ctx context.Context, publicID string) (SecretPayload, error) {
	row := s.pool.QueryRow(ctx, `
		UPDATE secrets
		SET consumed_at = CASE WHEN one_time THEN now() ELSE consumed_at END
		WHERE public_id = $1
			AND deleted_at IS NULL
			AND expires_at > now()
			AND (one_time = false OR consumed_at IS NULL)
		RETURNING encrypted_payload, iv, algorithm, version, one_time, passphrase_enabled,
			wrapped_key, wrapping_iv, kdf_salt, COALESCE(kdf_iterations, 0), COALESCE(kdf_algorithm, '')
	`, publicID)

	var payload SecretPayload
	err := row.Scan(
		&payload.EncryptedPayload,
		&payload.IV,
		&payload.Algorithm,
		&payload.Version,
		&payload.OneTime,
		&payload.PassphraseEnabled,
		&payload.WrappedKey,
		&payload.WrappingIV,
		&payload.KDFSalt,
		&payload.KDFIterations,
		&payload.KDFAlgorithm,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return SecretPayload{}, ErrNotFound
	}
	return payload, err
}

func (s *Store) DeleteSecret(ctx context.Context, publicID, deleteTokenHash string) (bool, error) {
	tag, err := s.pool.Exec(ctx, `
		UPDATE secrets
		SET deleted_at = now()
		WHERE public_id = $1
			AND delete_token_hash = $2
			AND deleted_at IS NULL
	`, publicID, deleteTokenHash)
	if err != nil {
		return false, err
	}
	return tag.RowsAffected() == 1, nil
}

func (s *Store) ClaimRevealNotification(ctx context.Context, publicID, revealTokenHash string) (string, error) {
	row := s.pool.QueryRow(ctx, `
		WITH pending AS (
			SELECT sender_notify_email
			FROM secrets
			WHERE public_id = $1
				AND deleted_at IS NULL
				AND expires_at > now()
				AND notify_sender_on_reveal = true
				AND sender_notify_email IS NOT NULL
				AND sender_notified_at IS NULL
				AND reveal_token_hash = $2
				AND (one_time = false OR consumed_at IS NOT NULL)
			FOR UPDATE
		)
		UPDATE secrets
		SET sender_notified_at = now(), sender_notify_email = NULL
		FROM pending
		WHERE public_id = $1
		RETURNING pending.sender_notify_email
	`, publicID, revealTokenHash)

	var senderEmail string
	err := row.Scan(&senderEmail)
	if errors.Is(err, pgx.ErrNoRows) {
		return "", ErrNotFound
	}
	return senderEmail, err
}

func (s *Store) IncrementStats(ctx context.Context, day time.Time, columns ...string) error {
	if len(columns) == 0 {
		return nil
	}
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	if _, err := tx.Exec(ctx, `
		INSERT INTO daily_stats (stat_date) VALUES ($1)
		ON CONFLICT (stat_date) DO NOTHING
	`, day.UTC().Format("2006-01-02")); err != nil {
		return err
	}

	for _, column := range columns {
		if !allowedStatColumn(column) {
			continue
		}
		if _, err := tx.Exec(ctx, `UPDATE daily_stats SET `+column+` = `+column+` + 1, updated_at = now() WHERE stat_date = $1`, day.UTC().Format("2006-01-02")); err != nil {
			return err
		}
	}
	return tx.Commit(ctx)
}

func (s *Store) ListDailyStats(ctx context.Context, limit int) ([]DailyStats, error) {
	if limit <= 0 || limit > 90 {
		limit = 30
	}
	rows, err := s.pool.Query(ctx, `
		SELECT stat_date, secrets_created_count, secrets_opened_count, secrets_expired_count,
			secrets_deleted_count, emails_sent_count, manual_links_created_count,
			passphrase_enabled_count, one_time_enabled_count, files_uploaded_count,
			total_encrypted_file_bytes
		FROM daily_stats
		ORDER BY stat_date DESC
		LIMIT $1
	`, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []DailyStats
	for rows.Next() {
		var item DailyStats
		if err := rows.Scan(
			&item.Date,
			&item.SecretsCreatedCount,
			&item.SecretsOpenedCount,
			&item.SecretsExpiredCount,
			&item.SecretsDeletedCount,
			&item.EmailsSentCount,
			&item.ManualLinksCreatedCount,
			&item.PassphraseEnabledCount,
			&item.OneTimeEnabledCount,
			&item.FilesUploadedCount,
			&item.TotalEncryptedFileBytes,
		); err != nil {
			return nil, err
		}
		out = append(out, item)
	}
	return out, rows.Err()
}

func (s *Store) CleanupExpired(ctx context.Context, now time.Time) (int64, int64, error) {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return 0, 0, err
	}
	defer tx.Rollback(ctx)

	expiredTag, err := tx.Exec(ctx, `
		UPDATE secrets
		SET deleted_at = now()
		WHERE expires_at <= $1 AND deleted_at IS NULL
	`, now)
	if err != nil {
		return 0, 0, err
	}
	expired := expiredTag.RowsAffected()
	purgeBefore := now.Add(-24 * time.Hour)

	purgeTag, err := tx.Exec(ctx, `
		DELETE FROM secrets
		WHERE deleted_at IS NOT NULL
			AND deleted_at < $1
	`, purgeBefore)
	if err != nil {
		return 0, 0, err
	}
	consumedPurgeTag, err := tx.Exec(ctx, `
		DELETE FROM secrets
		WHERE one_time = true
			AND consumed_at IS NOT NULL
			AND consumed_at < $1
	`, purgeBefore)
	if err != nil {
		return 0, 0, err
	}
	rateLimitPurgeTag, err := tx.Exec(ctx, `
		DELETE FROM rate_limit_buckets
		WHERE resets_at < $1
	`, purgeBefore)
	if err != nil {
		return 0, 0, err
	}

	for i := int64(0); i < expired; i++ {
		if err := incrementStatsTx(ctx, tx, now, "secrets_expired_count"); err != nil {
			return 0, 0, err
		}
	}

	purged := purgeTag.RowsAffected() + consumedPurgeTag.RowsAffected() + rateLimitPurgeTag.RowsAffected()
	return expired, purged, tx.Commit(ctx)
}

func incrementStatsTx(ctx context.Context, tx pgx.Tx, day time.Time, column string) error {
	if !allowedStatColumn(column) {
		return nil
	}
	if _, err := tx.Exec(ctx, `
		INSERT INTO daily_stats (stat_date) VALUES ($1)
		ON CONFLICT (stat_date) DO NOTHING
	`, day.UTC().Format("2006-01-02")); err != nil {
		return err
	}
	_, err := tx.Exec(ctx, `UPDATE daily_stats SET `+column+` = `+column+` + 1, updated_at = now() WHERE stat_date = $1`, day.UTC().Format("2006-01-02"))
	return err
}

func allowedStatColumn(column string) bool {
	switch column {
	case "secrets_created_count",
		"secrets_opened_count",
		"secrets_expired_count",
		"secrets_deleted_count",
		"emails_sent_count",
		"manual_links_created_count",
		"passphrase_enabled_count",
		"one_time_enabled_count",
		"files_uploaded_count",
		"total_encrypted_file_bytes":
		return true
	default:
		return false
	}
}

func nilIfEmpty(value []byte) any {
	if len(value) == 0 {
		return nil
	}
	return value
}

func nilIfZero(value int) any {
	if value == 0 {
		return nil
	}
	return value
}

func nilIfEmptyString(value string) any {
	if value == "" {
		return nil
	}
	return value
}
