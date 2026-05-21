package datasource

import (
	"encoding/json"
	"log"
	"net/http"
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

// applyAuth adds authentication headers to the request based on auth type.
func applyAuth(req *http.Request, authType, authConfig string) {
	if authType == "" || authType == "none" || authConfig == "" {
		return
	}

	switch authType {
	case "basic":
		var cfg basicAuthConfig
		if err := json.Unmarshal([]byte(authConfig), &cfg); err != nil {
			log.Printf("[datasource/auth] WARNING: failed to parse basic auth config, request will proceed without auth: %v", err)
		} else {
			req.SetBasicAuth(cfg.Username, cfg.Password)
		}
	case "bearer":
		var cfg bearerAuthConfig
		if err := json.Unmarshal([]byte(authConfig), &cfg); err != nil {
			log.Printf("[datasource/auth] WARNING: failed to parse bearer auth config, request will proceed without auth: %v", err)
		} else {
			req.Header.Set("Authorization", "Bearer "+cfg.Token)
		}
	case "api_key":
		var cfg apiKeyAuthConfig
		if err := json.Unmarshal([]byte(authConfig), &cfg); err != nil {
			log.Printf("[datasource/auth] WARNING: failed to parse api_key auth config, request will proceed without auth: %v", err)
		} else {
			headerName := cfg.Header
			if headerName == "" {
				headerName = "X-API-Key"
			}
			req.Header.Set(headerName, cfg.Value)
		}
	}
}
