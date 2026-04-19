package auth

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// ProviderSupabase is the value stored in user_identities.provider for Supabase Auth.
const ProviderSupabase = "supabase"

// ResolveSupabaseUser returns the internal users.id for this Supabase auth subject,
// creating users + user_identities on first sight (JIT provisioning).
func ResolveSupabaseUser(ctx context.Context, pool *pgxpool.Pool, subject uuid.UUID, email string) (uuid.UUID, error) {
	subStr := subject.String()
	var uid uuid.UUID
	err := pool.QueryRow(ctx,
		`SELECT user_id FROM user_identities WHERE provider = $1 AND provider_subject = $2`,
		ProviderSupabase, subStr,
	).Scan(&uid)
	if err == nil {
		return uid, nil
	}
	if !errors.Is(err, pgx.ErrNoRows) {
		return uuid.Nil, err
	}

	tx, err := pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return uuid.Nil, err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	var internalID uuid.UUID
	haveUser := false
	emailNorm := strings.TrimSpace(strings.ToLower(email))
	if emailNorm != "" {
		err = tx.QueryRow(ctx, `SELECT id FROM users WHERE email = $1`, emailNorm).Scan(&internalID)
		if err == nil {
			haveUser = true
		} else if !errors.Is(err, pgx.ErrNoRows) {
			return uuid.Nil, err
		}
	}
	if !haveUser {
		mail := emailNorm
		if mail == "" {
			mail = fmt.Sprintf("%s@users.supabase.uid.invalid", subStr)
		}
		err = tx.QueryRow(ctx,
			`INSERT INTO users (email, password_hash) VALUES ($1, NULL) RETURNING id`,
			mail,
		).Scan(&internalID)
		if err != nil {
			low := strings.ToLower(err.Error())
			if strings.Contains(low, "unique") || strings.Contains(low, "duplicate") {
				if err2 := tx.QueryRow(ctx, `SELECT id FROM users WHERE email = $1`, mail).Scan(&internalID); err2 != nil {
					return uuid.Nil, err2
				}
			} else {
				return uuid.Nil, err
			}
		}
	}

	pe := strings.TrimSpace(email)
	_, err = tx.Exec(ctx,
		`INSERT INTO user_identities (user_id, provider, provider_subject, provider_email)
		 VALUES ($1, $2, $3, NULLIF($4, ''))
		 ON CONFLICT (provider, provider_subject) DO NOTHING`,
		internalID, ProviderSupabase, subStr, pe,
	)
	if err != nil {
		return uuid.Nil, err
	}
	err = tx.QueryRow(ctx,
		`SELECT user_id FROM user_identities WHERE provider = $1 AND provider_subject = $2`,
		ProviderSupabase, subStr,
	).Scan(&internalID)
	if err != nil {
		return uuid.Nil, err
	}
	if err := tx.Commit(ctx); err != nil {
		return uuid.Nil, err
	}
	return internalID, nil
}
