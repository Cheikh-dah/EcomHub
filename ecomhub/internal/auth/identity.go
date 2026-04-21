package auth

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/clerk/clerk-sdk-go/v2"
	clerkuser "github.com/clerk/clerk-sdk-go/v2/user"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// ProviderClerk is stored in user_identities.provider for Clerk users.
const ProviderClerk = "clerk"

// ResolveClerkUser returns internal users.id for this Clerk user id (sub claim),
// creating users + user_identities on first sight (JIT provisioning).
func ResolveClerkUser(ctx context.Context, pool *pgxpool.Pool, clerkUserID string) (uuid.UUID, error) {
	clerkUserID = strings.TrimSpace(clerkUserID)
	if clerkUserID == "" {
		return uuid.Nil, errors.New("empty clerk user id")
	}

	var uid uuid.UUID
	err := pool.QueryRow(ctx,
		`SELECT user_id FROM user_identities WHERE provider = $1 AND provider_subject = $2`,
		ProviderClerk, clerkUserID,
	).Scan(&uid)
	if err == nil {
		return uid, nil
	}
	if !errors.Is(err, pgx.ErrNoRows) {
		return uuid.Nil, err
	}

	cu, err := clerkuser.Get(ctx, clerkUserID)
	if err != nil {
		return uuid.Nil, fmt.Errorf("clerk user lookup: %w", err)
	}

	tx, err := pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return uuid.Nil, err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	var internalID uuid.UUID
	haveUser := false
	emailNorm := primaryEmailNorm(cu)
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
			mail = fmt.Sprintf("%s@users.clerk.uid.invalid", clerkUserID)
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

	provEmail := ""
	if cu != nil {
		provEmail = strings.TrimSpace(primaryEmailRaw(cu))
	}
	_, err = tx.Exec(ctx,
		`INSERT INTO user_identities (user_id, provider, provider_subject, provider_email)
		 VALUES ($1, $2, $3, NULLIF($4, ''))
		 ON CONFLICT (provider, provider_subject) DO NOTHING`,
		internalID, ProviderClerk, clerkUserID, provEmail,
	)
	if err != nil {
		return uuid.Nil, err
	}
	err = tx.QueryRow(ctx,
		`SELECT user_id FROM user_identities WHERE provider = $1 AND provider_subject = $2`,
		ProviderClerk, clerkUserID,
	).Scan(&internalID)
	if err != nil {
		return uuid.Nil, err
	}
	if err := tx.Commit(ctx); err != nil {
		return uuid.Nil, err
	}
	return internalID, nil
}

func primaryEmailNorm(u *clerk.User) string {
	return strings.TrimSpace(strings.ToLower(primaryEmailRaw(u)))
}

func primaryEmailRaw(u *clerk.User) string {
	if u == nil {
		return ""
	}
	var primaryID string
	if u.PrimaryEmailAddressID != nil {
		primaryID = *u.PrimaryEmailAddressID
	}
	for _, ea := range u.EmailAddresses {
		if ea == nil {
			continue
		}
		if primaryID != "" && ea.ID == primaryID {
			return ea.EmailAddress
		}
	}
	if len(u.EmailAddresses) > 0 && u.EmailAddresses[0] != nil {
		return u.EmailAddresses[0].EmailAddress
	}
	return ""
}
