package datasource

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
)

type basicAuthConfig struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

type bearerAuthConfig struct {
	Token string `json:"token"`
}

type apiKeyAuthConfig struct {
	Header string `json:"header"`
	Value  string `json:"value"`
}

// blockedAPIKeyHeaders lists HTTP headers that must not be set by user-supplied
// api_key auth configs. These headers can interfere with request routing, cause
// security issues, or lead to request smuggling.
var blockedAPIKeyHeaders = map[string]bool{
	"host":           true,
	"content-length": true,
	"content-type":   true,
	"cookie":         true,
	"set-cookie":     true,
	"origin":         true,
	"referer":        true,
}

// applyAuth adds authentication headers to the request based on auth type.
// Returns an error if the auth config cannot be parsed or the api_key header
// name is blocked. Callers must check the returned error.
func applyAuth(req *http.Request, authType, authConfig string) error {
	if authType == "" || authType == "none" || authConfig == "" {
		return nil
	}

	switch authType {
	case "basic":
		var cfg basicAuthConfig
		if err := json.Unmarshal([]byte(authConfig), &cfg); err != nil {
			return fmt.Errorf("invalid basic auth config: %w", err)
		}
		req.SetBasicAuth(cfg.Username, cfg.Password)

	case "bearer":
		var cfg bearerAuthConfig
		if err := json.Unmarshal([]byte(authConfig), &cfg); err != nil {
			return fmt.Errorf("invalid bearer auth config: %w", err)
		}
		req.Header.Set("Authorization", "Bearer "+cfg.Token)

	case "api_key":
		var cfg apiKeyAuthConfig
		if err := json.Unmarshal([]byte(authConfig), &cfg); err != nil {
			return fmt.Errorf("invalid api_key auth config: %w", err)
		}
		headerName := cfg.Header
		if headerName == "" {
			headerName = "X-API-Key"
		}
		if blockedAPIKeyHeaders[strings.ToLower(headerName)] {
			return fmt.Errorf("api_key header %q is not allowed", headerName)
		}
		req.Header.Set(headerName, cfg.Value)
	}

	return nil
}
