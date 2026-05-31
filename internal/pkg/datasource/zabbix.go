package datasource

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"
)

// ZabbixChecker checks Zabbix API health via JSON-RPC.
type ZabbixChecker struct{}

type zabbixRequest struct {
	JSONRPC string      `json:"jsonrpc"`
	Method  string      `json:"method"`
	Params  interface{} `json:"params"`
	Auth    *string     `json:"auth,omitempty"` // nil for unauthenticated calls (apiinfo.version)
	ID      int         `json:"id"`
}

type zabbixResponse struct {
	JSONRPC string          `json:"jsonrpc"`
	Result  json.RawMessage `json:"result"`
	Error   *struct {
		Code    int    `json:"code"`
		Message string `json:"message"`
		Data    string `json:"data"`
	} `json:"error"`
	ID int `json:"id"`
}

// zabbixItem represents a single item returned by item.get.
type zabbixItem struct {
	ItemID    string `json:"itemid"`
	HostID    string `json:"hostid"`
	HostName  string `json:"host"`
	Name      string `json:"name"`
	Key_      string `json:"key_"`
	LastValue string `json:"lastvalue"`
	LastClock string `json:"lastclock"` // Unix timestamp as string
}

// zabbixAuthConfig holds the Zabbix API token or user/password auth config.
// Stored as JSON in DataSource.AuthConfig field.
// Example: {"token": "abc123"} for token auth (Zabbix 5.4+)
// Example: {"username": "Admin", "password": "zabbix"} for user/password auth
type zabbixAuthConfig struct {
	Token    string `json:"token"`
	Username string `json:"username"`
	Password string `json:"password"`
}

// zabbixCachedToken holds a cached Zabbix API session token with its expiry time.
type zabbixCachedToken struct {
	token   string
	expires time.Time
}

// zabbixTokenCache provides thread-safe caching of Zabbix API session tokens
// keyed by apiURL + "|" + username. Prevents re-login on every query.
type zabbixTokenCache struct {
	mu     sync.Mutex
	tokens map[string]zabbixCachedToken
}

var defaultZabbixTokenCache = &zabbixTokenCache{
	tokens: make(map[string]zabbixCachedToken),
}

func (c *zabbixTokenCache) cacheKey(apiURL, username string) string {
	return apiURL + "|" + username
}

func (c *zabbixTokenCache) get(apiURL, username string) (string, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	key := c.cacheKey(apiURL, username)
	if ct, ok := c.tokens[key]; ok && time.Now().Before(ct.expires) {
		return ct.token, true
	}
	return "", false
}

func (c *zabbixTokenCache) put(apiURL, username, token string, ttl time.Duration) {
	c.mu.Lock()
	defer c.mu.Unlock()
	key := c.cacheKey(apiURL, username)
	c.tokens[key] = zabbixCachedToken{
		token:   token,
		expires: time.Now().Add(ttl),
	}
}

// cleanup removes all expired entries from the cache.
func (c *zabbixTokenCache) cleanup() {
	c.mu.Lock()
	defer c.mu.Unlock()
	now := time.Now()
	for key, ct := range c.tokens {
		if now.After(ct.expires) {
			delete(c.tokens, key)
		}
	}
}

// StartCleanup runs periodic cache cleanup until ctx is cancelled.
// Typically called once at startup: go defaultZabbixTokenCache.StartCleanup(ctx, 10*time.Minute)
func (c *zabbixTokenCache) StartCleanup(ctx context.Context, interval time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			c.cleanup()
		}
	}
}

// StartZabbixCacheCleanup starts periodic cleanup of the global Zabbix token cache.
func StartZabbixCacheCleanup(ctx context.Context, interval time.Duration) {
	defaultZabbixTokenCache.StartCleanup(ctx, interval)
}

// CheckHealth calls apiinfo.version via JSON-RPC to verify the Zabbix API is reachable
// and optionally tests auth credentials if configured (basic/token).
func (c *ZabbixChecker) CheckHealth(ctx context.Context, endpoint, authType, authConfig string) HealthResult {
	apiURL := strings.TrimRight(endpoint, "/") + "/api_jsonrpc.php"

	reqBody := zabbixRequest{
		JSONRPC: "2.0",
		Method:  "apiinfo.version",
		Params:  []interface{}{},
		ID:      1,
	}

	bodyBytes, _ := json.Marshal(reqBody)

	start := time.Now()
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, apiURL, bytes.NewReader(bodyBytes))
	if err != nil {
		return HealthResult{Healthy: false, LatencyMs: -1,
			Message: fmt.Sprintf("failed to create request: %v", err)}
	}
	req.Header.Set("Content-Type", "application/json-rpc")

	resp, err := httpClient.Do(req)
	latency := time.Since(start).Milliseconds()
	if err != nil {
		return HealthResult{Healthy: false, LatencyMs: latency,
			Message: fmt.Sprintf("API unreachable: %v", err)}
	}
	defer func() { _ = resp.Body.Close() }()

	body, _ := io.ReadAll(io.LimitReader(resp.Body, 10<<20))

	var zResp zabbixResponse
	if err := json.Unmarshal(body, &zResp); err != nil {
		return HealthResult{Healthy: false, LatencyMs: latency,
			Message: fmt.Sprintf("invalid JSON-RPC response: %v", err)}
	}
	if zResp.Error != nil {
		return HealthResult{Healthy: false, LatencyMs: latency,
			Message: fmt.Sprintf("Zabbix API error: %s — %s", zResp.Error.Message, zResp.Error.Data)}
	}

	// zResp.Result contains the version string for apiinfo.version (JSON-encoded string)
	var version string
	_ = json.Unmarshal(zResp.Result, &version)
	msg := "Zabbix API is healthy"
	if version != "" {
		msg = fmt.Sprintf("Zabbix %s API is healthy", version)
	}

	// If auth credentials are provided, also verify they actually work
	if authType == "basic" && authConfig != "" {
		var cfg struct {
			Username string `json:"username"`
			Password string `json:"password"`
		}
		if err := json.Unmarshal([]byte(authConfig), &cfg); err == nil && cfg.Username != "" {
			if _, authErr := zabbixAPIToken(ctx, apiURL, cfg.Username, cfg.Password); authErr != nil {
				return HealthResult{Healthy: false, LatencyMs: latency,
					Message: fmt.Sprintf("Zabbix auth failed: %v", authErr), Version: version}
			}
			msg += " (credentials verified)"
		}
	} else if authType == "bearer" && authConfig != "" {
		// Verify bearer token by making an authenticated API call (user.get with limit 1).
		// If the token is invalid/expired, Zabbix returns a JSON-RPC auth error.
		var cfg zabbixAuthConfig
		if err := json.Unmarshal([]byte(authConfig), &cfg); err == nil && cfg.Token != "" {
			if authErr := zabbixVerifyToken(ctx, apiURL, cfg.Token); authErr != nil {
				return HealthResult{Healthy: false, LatencyMs: latency,
					Message: fmt.Sprintf("Zabbix token auth failed: %v", authErr), Version: version}
			}
			msg += " (token verified)"
		}
	}

	return HealthResult{Healthy: true, LatencyMs: latency, Message: msg, Version: version}
}

// zabbixAPIToken retrieves an API session token by logging in with username/password.
// Returns the token string or an error. Used internally when token auth is not available.
//
// NOTE: Zabbix 7.0+ deprecates JSON-RPC user.login in favor of HTTP Authorization header.
// Users on Zabbix 7.0+ should configure API token authentication ({"token":"..."}) in the
// datasource auth_config instead of username/password, which avoids this deprecated code path.
func zabbixAPIToken(ctx context.Context, apiURL, username, password string) (string, error) {
	reqBody := zabbixRequest{
		JSONRPC: "2.0",
		Method:  "user.login",
		Params:  map[string]string{"username": username, "password": password},
		ID:      1,
	}

	bodyBytes, err := json.Marshal(reqBody)
	if err != nil {
		return "", fmt.Errorf("failed to marshal login request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, apiURL, bytes.NewReader(bodyBytes))
	if err != nil {
		return "", fmt.Errorf("failed to create login request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json-rpc")

	resp, err := httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("login request failed: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	body, err := io.ReadAll(io.LimitReader(resp.Body, 10<<20))
	if err != nil {
		return "", fmt.Errorf("failed to read login response: %w", err)
	}

	var zResp zabbixResponse
	if err := json.Unmarshal(body, &zResp); err != nil {
		return "", fmt.Errorf("failed to parse login response: %w", err)
	}

	if zResp.Error != nil {
		return "", fmt.Errorf("zabbix login error: %s - %s", zResp.Error.Message, zResp.Error.Data)
	}

	var token string
	if err := json.Unmarshal(zResp.Result, &token); err != nil {
		return "", fmt.Errorf("failed to parse login token: %w", err)
	}
	return token, nil
}

// zabbixVerifyToken verifies a Zabbix API bearer token by making an authenticated
// API call (user.get with limit 1). Returns nil if the token is valid, or an error
// if the token is invalid/expired (Zabbix returns a JSON-RPC auth error).
func zabbixVerifyToken(ctx context.Context, apiURL, token string) error {
	reqBody := zabbixRequest{
		JSONRPC: "2.0",
		Method:  "user.get",
		Params: map[string]interface{}{
			"output": []string{"userid"},
			"limit":  1,
		},
		Auth: &token,
		ID:   3,
	}

	bodyBytes, err := json.Marshal(reqBody)
	if err != nil {
		return fmt.Errorf("failed to marshal user.get request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, apiURL, bytes.NewReader(bodyBytes))
	if err != nil {
		return fmt.Errorf("failed to create user.get request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json-rpc")

	resp, err := httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("user.get request failed: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	body, err := io.ReadAll(io.LimitReader(resp.Body, 10<<20))
	if err != nil {
		return fmt.Errorf("failed to read user.get response: %w", err)
	}

	var zResp zabbixResponse
	if err := json.Unmarshal(body, &zResp); err != nil {
		return fmt.Errorf("failed to parse user.get response: %w", err)
	}

	if zResp.Error != nil {
		return fmt.Errorf("token rejected: %s — %s", zResp.Error.Message, zResp.Error.Data)
	}

	return nil
}

// ZabbixInstantQuery queries Zabbix items by key pattern and returns QueryResults.
// expression is a Zabbix item key pattern (e.g. "system.cpu.util", "vm.memory.*").
// authConfig JSON should contain either {"token":"..."} or {"username":"...","password":"..."}.
func ZabbixInstantQuery(ctx context.Context, endpoint, authType, authConfig, expression string) ([]QueryResult, error) {
	apiURL := strings.TrimRight(endpoint, "/") + "/api_jsonrpc.php"

	// Parse auth config
	var cfg zabbixAuthConfig
	if authConfig != "" {
		_ = json.Unmarshal([]byte(authConfig), &cfg)
	}

	// Resolve API token (with cache for username/password auth)
	token := cfg.Token
	if token == "" && cfg.Username != "" {
		if cached, ok := defaultZabbixTokenCache.get(apiURL, cfg.Username); ok {
			token = cached
		} else {
			var err error
			token, err = zabbixAPIToken(ctx, apiURL, cfg.Username, cfg.Password)
			if err != nil {
				return nil, fmt.Errorf("zabbix authentication failed: %w", err)
			}
			// Cache for 3 hours (Zabbix sessions default to 4 hours)
			defaultZabbixTokenCache.put(apiURL, cfg.Username, token, 3*time.Hour)
		}
	}

	// Build item.get params — search by key pattern
	params := map[string]interface{}{
		"output":                 []string{"itemid", "hostid", "host", "name", "key_", "lastvalue", "lastclock"},
		"search":                 map[string]string{"key_": expression},
		"searchWildcardsEnabled": true,
		"sortfield":              "name",
		"limit":                  1000,
	}

	reqBody := zabbixRequest{
		JSONRPC: "2.0",
		Method:  "item.get",
		Params:  params,
		ID:      2,
	}
	if token != "" {
		reqBody.Auth = &token
	}

	bodyBytes, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal item.get request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, apiURL, bytes.NewReader(bodyBytes))
	if err != nil {
		return nil, fmt.Errorf("failed to create item.get request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json-rpc")

	resp, err := httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("item.get request failed: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	body, err := io.ReadAll(io.LimitReader(resp.Body, 10<<20))
	if err != nil {
		return nil, fmt.Errorf("failed to read item.get response: %w", err)
	}

	var zResp zabbixResponse
	if err := json.Unmarshal(body, &zResp); err != nil {
		return nil, fmt.Errorf("failed to parse item.get response: %w", err)
	}

	if zResp.Error != nil {
		return nil, fmt.Errorf("zabbix item.get error: %s - %s", zResp.Error.Message, zResp.Error.Data)
	}

	var items []zabbixItem
	if err := json.Unmarshal(zResp.Result, &items); err != nil {
		return nil, fmt.Errorf("failed to parse items: %w", err)
	}

	results := make([]QueryResult, 0, len(items))
	for _, item := range items {
		val, err := strconv.ParseFloat(item.LastValue, 64)
		if err != nil {
			// Skip non-numeric items (e.g. string values)
			continue
		}

		ts := time.Now()
		if item.LastClock != "" {
			if clockSec, err := strconv.ParseInt(item.LastClock, 10, 64); err == nil {
				ts = time.Unix(clockSec, 0)
			}
		}

		results = append(results, QueryResult{
			MetricName: item.Key_,
			Labels: map[string]string{
				"__name__":  item.Key_,
				"host":      item.HostName,
				"itemid":    item.ItemID,
				"item_key":  item.Key_,
				"item_name": item.Name,
			},
			Values: []DataPoint{{Timestamp: ts, Value: val}},
		})
	}

	return results, nil
}
