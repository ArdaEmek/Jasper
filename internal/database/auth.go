package database

import (
	"context"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
)

type AuthError struct {
	Code    string
	Message string
}

func (s *service) ValidatePresignedUrl(ctx context.Context, query url.Values) (*User, *ApiKey, *AuthError) {
	creds := query.Get("X-Amz-Credential")
	dateStr := query.Get("X-Amz-Date")
	expiresStr := query.Get("X-Amz-Expires")

	if creds == "" || dateStr == "" || expiresStr == "" {
		return nil, nil, &AuthError{"AccessDenied", "Missing required query parameters"}
	}

	startTime, err := time.Parse("20060102T150405Z", dateStr)
	if err != nil {
		return nil, nil, &AuthError{"AccessDenied", "Invalid X-Amz-Date format"}
	}

	expSecs, err := strconv.Atoi(expiresStr)
	if err != nil {
		return nil, nil, &AuthError{"AccessDenied", "Invalid X-Amz-Expires value"}
	}

	now := time.Now().UTC()
	expTime := startTime.Add(time.Duration(expSecs) * time.Second)

	if now.After(expTime) {
		return nil, nil, &AuthError{"AccessDenied", "Request has expired"}
	}

	// Clock skew check
	if now.Before(startTime.Add(-5 * time.Minute)) {
		return nil, nil, &AuthError{"RequestTimeTooSkewed", "Request time is too far in the future"}
	}

	accessKey := strings.Split(creds, "/")[0]
	user, apiKey, err := s.GetUserByApiKey(ctx, accessKey)
	if err != nil || user == nil {
		return nil, nil, &AuthError{"InvalidAccessKeyId", "The AWS Access Key Id you provided does not exist"}
	}

	return user, apiKey, nil
}

func (s *service) ValidateHeaderAuth(ctx context.Context, r *http.Request) (*User, *ApiKey, *AuthError) {
	authHeader := r.Header.Get("Authorization")
	if authHeader == "" {
		return nil, nil, &AuthError{"AccessDenied", "Missing Authorization header"}
	}

	dateStr := r.Header.Get("X-Amz-Date")
	if dateStr == "" {
		dateStr = r.Header.Get("Date")
	}
	if dateStr == "" {
		return nil, nil, &AuthError{"AccessDenied", "Missing date header"}
	}

	requestTime, err := time.Parse("20060102T150405Z", dateStr)
	if err != nil {
		return nil, nil, &AuthError{"AccessDenied", "Invalid date format"}
	}

	now := time.Now().UTC()
	diff := now.Sub(requestTime)
	if diff < 0 {
		diff = -diff
	}

	// Clock skew check
	if diff > 5*time.Minute {
		return nil, nil, &AuthError{"RequestTimeTooSkewed", "The difference between the request time and the current time is too large."}
	}

	parts := strings.Split(authHeader, "Credential=")
	if len(parts) < 2 {
		return nil, nil, &AuthError{"AccessDenied", "Malformed Authorization header"}
	}

	credentialScope := strings.Split(parts[1], ",")[0]
	accessKey := strings.Split(credentialScope, "/")[0]

	user, apiKey, err := s.GetUserByApiKey(ctx, accessKey)
	if err != nil || user == nil {
		return nil, nil, &AuthError{"InvalidAccessKeyId", "The AWS Access Key Id you provided does not exist"}
	}

	return user, apiKey, nil
}
