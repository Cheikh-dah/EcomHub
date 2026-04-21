package auth

import (
	"context"
	"strings"
	"time"

	clerkjwt "github.com/clerk/clerk-sdk-go/v2/jwt"
)

// VerifyClerkSessionJWT verifies a Clerk session JWT (RS256 via JWKS) and returns
// the Clerk user id from the standard `sub` claim and a suggested cookie MaxAge from `exp`.
func VerifyClerkSessionJWT(ctx context.Context, tokenString string, authorizedOrigins []string) (clerkUserID string, cookieMaxAge int, err error) {
	cookieMaxAge = 3600
	tokenString = strings.TrimSpace(tokenString)
	if tokenString == "" {
		return "", cookieMaxAge, errEmptyToken
	}

	var partyHandler clerkjwt.AuthorizedPartyHandler
	if len(authorizedOrigins) > 0 {
		allowed := make(map[string]struct{}, len(authorizedOrigins))
		for _, o := range authorizedOrigins {
			if s := strings.TrimSpace(o); s != "" {
				allowed[s] = struct{}{}
			}
		}
		partyHandler = func(azp string) bool {
			if azp == "" {
				return true
			}
			_, ok := allowed[azp]
			return ok
		}
	}

	claims, err := clerkjwt.Verify(ctx, &clerkjwt.VerifyParams{
		Token:                  tokenString,
		AuthorizedPartyHandler: partyHandler,
	})
	if err != nil {
		return "", cookieMaxAge, err
	}
	clerkUserID = strings.TrimSpace(claims.Subject)
	if clerkUserID == "" {
		return "", cookieMaxAge, errMissingSub
	}
	if claims.Expiry != nil && *claims.Expiry > 0 {
		sec := int(time.Until(time.Unix(*claims.Expiry, 0)).Seconds())
		if sec < 60 {
			sec = 60
		}
		const maxCookie = 86400 * 7
		if sec > maxCookie {
			sec = maxCookie
		}
		cookieMaxAge = sec
	}
	return clerkUserID, cookieMaxAge, nil
}

var (
	errEmptyToken = errStr("empty token")
	errMissingSub = errStr("jwt missing sub")
)

type errStr string

func (e errStr) Error() string { return string(e) }
