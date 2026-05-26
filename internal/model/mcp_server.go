package model

import (
	"encoding/json"
	"fmt"

	"gorm.io/gorm"
)

// MCPServer stores a registered MCP (Model Context Protocol) server
// that AI agents can connect to for external tool discovery and invocation.
type MCPServer struct {
	gorm.Model

	Name        string `json:"name" gorm:"column:name;size:128;not null;uniqueIndex"`
	URL         string `json:"url" gorm:"column:url;size:512;not null"`
	Headers     string `json:"headers" gorm:"column:headers;type:text"`
	Description string `json:"description" gorm:"column:description;size:1024"`
	Enabled     bool   `json:"enabled" gorm:"column:enabled;default:true"`
}

func (MCPServer) TableName() string { return "mcp_servers" }

// GetHeadersMap parses the JSON headers string into a map.
func (m *MCPServer) GetHeadersMap() map[string]string {
	if m.Headers == "" {
		return nil
	}
	var h map[string]string
	if err := json.Unmarshal([]byte(m.Headers), &h); err != nil {
		return nil
	}
	return h
}

// SetHeadersMap serializes a map into the JSON headers string.
func (m *MCPServer) SetHeadersMap(h map[string]string) {
	if len(h) == 0 {
		m.Headers = ""
		return
	}
	b, _ := json.Marshal(h)
	m.Headers = string(b)
}

// Verify validates the MCP server fields.
func (m *MCPServer) Verify() error {
	if m.Name == "" {
		return fmt.Errorf("name is required")
	}
	if m.URL == "" {
		return fmt.Errorf("url is required")
	}
	return nil
}
