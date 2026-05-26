package model

import (
	"encoding/json"
	"time"

	"gorm.io/gorm"
)

// AISkill represents a structured skill package (SKILL.md + files).
type AISkill struct {
	ID          uint           `json:"id" gorm:"primaryKey"`
	Name        string         `json:"name" gorm:"size:128;not null;uniqueIndex"`
	Description string         `json:"description" gorm:"size:4096"`
	Instructions string        `json:"instructions" gorm:"type:text"` // SKILL.md body (markdown)
	License     string         `json:"license" gorm:"size:255"`
	Compatibility string       `json:"compatibility" gorm:"size:255"`
	AllowedTools string        `json:"allowed_tools" gorm:"size:4096"` // space-separated tool whitelist
	Metadata    string         `json:"metadata" gorm:"type:text"` // JSON map[string]string
	Enabled     bool           `json:"enabled" gorm:"default:true;index"`
	CreatedBy   string         `json:"created_by" gorm:"size:64"`
	UpdatedBy   string         `json:"updated_by" gorm:"size:64"`
	CreatedAt   time.Time      `json:"created_at"`
	UpdatedAt   time.Time      `json:"updated_at"`
	DeletedAt   gorm.DeletedAt `json:"-" gorm:"index"`

	// Runtime fields (not persisted)
	Files    []*AISkillFile `json:"files,omitempty" gorm:"-"`
	Builtin  bool           `json:"builtin" gorm:"-"` // true if CreatedBy == "system"
}

func (AISkill) TableName() string { return "ai_skills" }

// GetMetadataMap parses the JSON metadata field.
func (s *AISkill) GetMetadataMap() map[string]string {
	if s.Metadata == "" {
		return nil
	}
	var m map[string]string
	if err := json.Unmarshal([]byte(s.Metadata), &m); err != nil {
		return nil
	}
	return m
}

// SetMetadataMap serializes a map to JSON metadata.
func (s *AISkill) SetMetadataMap(m map[string]string) error {
	if m == nil {
		s.Metadata = ""
		return nil
	}
	b, err := json.Marshal(m)
	if err != nil {
		return err
	}
	s.Metadata = string(b)
	return nil
}

// AISkillFile represents a file within a skill package.
type AISkillFile struct {
	ID        uint      `json:"id" gorm:"primaryKey"`
	SkillID   uint      `json:"skill_id" gorm:"not null;index"`
	Name      string    `json:"name" gorm:"size:512;not null"` // relative path, e.g. "SKILL.md", "scripts/foo.sh"
	Content   string    `json:"content" gorm:"type:mediumtext"`
	Size      int64     `json:"size"` // auto-calculated from len(Content)
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

func (AISkillFile) TableName() string { return "ai_skill_files" }
