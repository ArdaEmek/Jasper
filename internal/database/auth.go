package database

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"net/http"
	"net/url"
	"slices"
	"strconv"
	"strings"
	"time"
)

type AuthError struct {
	Code    string
	Message string
}

// precomputed SHA256 hash of empty string: SHA256("")
const emptyPayloadHash = "e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855"

func (s *service) ValidatePresignedUrl(ctx context.Context, r *http.Request) (*User, *ApiKey, *AuthError) {
	query := r.URL.Query()
	creds := query.Get("X-Amz-Credential")
	dateStr := query.Get("X-Amz-Date")
	expiresStr := query.Get("X-Amz-Expires")
	signature := query.Get("X-Amz-Signature")
	signedHeaders := query.Get("X-Amz-SignedHeaders")
	algorithm := query.Get("X-Amz-Algorithm")

	if creds == "" || dateStr == "" || expiresStr == "" || signature == "" {
		return nil, nil, &AuthError{"AccessDenied", "Missing required query parameters"}
	}

	if algorithm != "AWS4-HMAC-SHA256" {
		return nil, nil, &AuthError{"AccessDenied", "Invalid signature algorithm"}
	}

	// Parse timestamp and check expiry
	startTime, err := time.Parse("20060102T150405Z", dateStr)
	if err != nil {
		return nil, nil, &AuthError{"AccessDenied", "Invalid X-Amz-Date format"}
	}

	expSecs, err := strconv.Atoi(expiresStr)
	if err != nil {
		return nil, nil, &AuthError{"AccessDenied", "Invalid X-Amz-Expires value"}
	}

	if expSecs > 604800 { // 7 days max
		return nil, nil, &AuthError{"AccessDenied", "Expiration time exceeds maximum allowed"}
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

	// Extract access key and fetch user/apikey
	accessKey := strings.Split(creds, "/")[0]
	user, apiKey, err := s.GetUserByApiKey(ctx, accessKey)
	if err != nil || user == nil || apiKey == nil {
		return nil, nil, &AuthError{"InvalidAccessKeyId", "The AWS Access Key Id you provided does not exist"}
	}

	ok := s.verifySigV4(r, signature, apiKey.SecretKey, dateStr, signedHeaders, creds)
	if !ok {
		return nil, nil, &AuthError{"SignatureDoesNotMatch", "The request signature that the server calculated does not match the signature that you provided"}
	}

	// Validate permission scope
	scope := query.Get("X-Amz-Scope")
	if scope != "" {
		if !s.validateScopePermission(scope, r.Method) {
			return nil, nil, &AuthError{"AccessDenied", "Request method not allowed for this credential scope"}
		}
	}

	return user, apiKey, nil
}

// verifySigV4 creates signature from parameters and checks if it matches the signature given by the client
func (s *service) verifySigV4(r *http.Request, providedSignature, secretKey, dateStr, signedHeaders, credentialScope string) bool {
	query := r.URL.Query()
	method := r.Method
	canonicalURI := r.URL.EscapedPath()

	// Build canonical query, excluding X-Amz-Signature
	var queryPairs []string
	for key, values := range query {
		if strings.EqualFold(key, "X-Amz-Signature") {
			continue
		}
		for _, value := range values {
			queryPairs = append(queryPairs, fmt.Sprintf("%s=%s", url.QueryEscape(key), url.QueryEscape(value)))
		}
	}
	slices.Sort(queryPairs)
	canonicalQueryString := strings.Join(queryPairs, "&")

	// X-Amz-Content-Sha256 contains the actual payload hash or "UNSIGNED-PAYLOAD"
	payloadHash := r.Header.Get("X-Amz-Content-Sha256")
	if payloadHash == "" {
		payloadHash = query.Get("X-Amz-Content-Sha256")
	}
	if payloadHash == "" {
		payloadHash = emptyPayloadHash
	}

	// Build canonical headers from signed headers
	canonicalHeaders := buildCanonicalHeaders(r, signedHeaders)
	canonicalRequest := fmt.Sprintf("%s\n%s\n%s\n%s\n%s\n%s",
		method,
		canonicalURI,
		canonicalQueryString,
		canonicalHeaders,
		signedHeaders,
		payloadHash,
	)
	hashedCanonicalRequest := hashSHA256(canonicalRequest)

	// Extract date and region
	parts := strings.Split(credentialScope, "/")
	if len(parts) < 5 {
		return false
	}

	// dateStr format: YYYYMMDDTHHmmss
	if len(dateStr) < 8 {
		return false
	}
	dateOnly := dateStr[:8] // Extract YYYYMMDD
	region := parts[2]
	stringToSign := fmt.Sprintf("AWS4-HMAC-SHA256\n%s\n%s/%s/s3/aws4_request\n%s",
		dateStr,
		dateOnly,
		region,
		hashedCanonicalRequest,
	)

	kSecret := "AWS4" + secretKey
	kDate := hmacSHA256(kSecret, dateOnly)
	kRegion := hmacSHA256(kDate, region)
	kService := hmacSHA256(kRegion, "s3")
	kSigning := hmacSHA256(kService, "aws4_request")

	computedSignature := hmacHexSHA256(kSigning, stringToSign)
	return computedSignature == providedSignature
}

func buildCanonicalHeaders(r *http.Request, signedHeaders string) string {
	headers := strings.Split(signedHeaders, ";")

	var headerLines []string
	for _, headerName := range headers {
		headerName = strings.TrimSpace(headerName)
		lowerHeaderName := strings.ToLower(headerName)

		// IMPORTANT: Uses host variable provided by quic-go, it'll ignore Host header provided by reverse proxy.
		var headerValue string
		if lowerHeaderName == "host" {
			headerValue = r.Host
		} else {
			headerValue = r.Header.Get(headerName)
		}

		headerLine := fmt.Sprintf("%s:%s", lowerHeaderName, strings.TrimSpace(headerValue))
		headerLines = append(headerLines, headerLine)
	}
	slices.Sort(headerLines)

	return strings.Join(headerLines, "\n") + "\n"
}

// Scopes: read, write, read_write
func (s *service) validateScopePermission(scope, method string) bool {
	switch scope {
	case "read":
		return method == http.MethodGet || method == http.MethodHead
	case "write":
		return method == http.MethodPut || method == http.MethodPost || method == http.MethodDelete
	case "read_write":
		return true
	default:
		return false
	}
}

func hashSHA256(data string) string {
	hash := sha256.Sum256([]byte(data))
	return hex.EncodeToString(hash[:])
}

func hmacSHA256(key, data string) string {
	h := hmac.New(sha256.New, []byte(key))
	h.Write([]byte(data))
	return string(h.Sum(nil))
}
func hmacHexSHA256(key, data string) string {
	h := hmac.New(sha256.New, []byte(key))
	h.Write([]byte(data))
	return hex.EncodeToString(h.Sum(nil))
}

func extractAuthParam(authHeader, key string) string {
	parts := strings.Split(authHeader, ",")
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if strings.HasPrefix(p, key+"=") {
			return strings.TrimPrefix(p, key+"=")
		}
	}

	return ""
}

func (s *service) ValidateHeaderAuth(ctx context.Context, r *http.Request) (*User, *ApiKey, *AuthError) {
	authHeader := r.Header.Get("Authorization")
	if authHeader == "" {
		return nil, nil, &AuthError{"InvalidArgument", "Missing Authorization header"}
	}

	dateStr := r.Header.Get("X-Amz-Date")
	if dateStr == "" {
		dateStr = r.Header.Get("Date")
	}
	if dateStr == "" {
		return nil, nil, &AuthError{"InvalidArgument", "Missing date header"}
	}

	requestTime, err := time.Parse("20060102T150405Z", dateStr)
	if err != nil {
		return nil, nil, &AuthError{"InvalidArgument", "Invalid date format"}
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
		return nil, nil, &AuthError{"InvalidArgument", "Malformed Authorization header"}
	}

	credentialScope := strings.Split(parts[1], ",")[0]
	accessKey := strings.Split(credentialScope, "/")[0]

	user, apiKey, err := s.GetUserByApiKey(ctx, accessKey)
	if err != nil || user == nil {
		return nil, nil, &AuthError{"InvalidAccessKeyId", "The AWS Access Key Id you provided does not exist"}
	}

	signedHeaders := extractAuthParam(parts[1], "SignedHeaders")
	signature := extractAuthParam(parts[1], "Signature")

	ok := s.verifySigV4(r, signature, apiKey.SecretKey, dateStr, signedHeaders, credentialScope)
	if !ok {
		return nil, nil, &AuthError{"SignatureDoesNotMatch", "The request signature that the server calculated does not match the signature that you provided"}
	}

	return user, apiKey, nil
}
