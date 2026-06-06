package model

import (
	"database/sql/driver"
	"encoding/json"
	"fmt"
)

// Team represents a group of users responsible for specific services.
type Team struct {
	BaseModel
	Name        string     `json:"name" gorm:"uniqueIndex;size:128;not null"`
	Description string     `json:"description" gorm:"size:512"`
	Labels      JSONLabels `json:"labels" gorm:"type:json"`
	Members     []User     `json:"members,omitempty" gorm:"many2many:team_members;"`
}

func (Team) TableName() string {
	return "teams"
}

// TeamMember is the join table for team-user relationship with role info.
type TeamMember struct {
	TeamID uint   `json:"team_id" gorm:"primaryKey"`
	UserID uint   `json:"user_id" gorm:"primaryKey"`
	Role   string `json:"role" gorm:"size:32;default:member"` // lead, member
	User   *User  `json:"user,omitempty" gorm:"foreignKey:UserID"`
}

func (TeamMember) TableName() string {
	return "team_members"
}

// JSONLabels is a custom type for storing labels as JSON in MySQL.
type JSONLabels map[string]string

func (j JSONLabels) Value() (driver.Value, error) {
	if j == nil {
		return "{}", nil
	}
	b, err := json.Marshal(j)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal labels: %w", err)
	}
	return string(b), nil
}

func (j *JSONLabels) Scan(value interface{}) error {
	if value == nil {
		*j = make(JSONLabels)
		return nil
	}
	var bytes []byte
	switch v := value.(type) {
	case string:
		bytes = []byte(v)
	case []byte:
		bytes = v
	default:
		return fmt.Errorf("unsupported type for JSONLabels: %T", value)
	}
	return json.Unmarshal(bytes, j)
}
