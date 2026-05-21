package model

// KnowledgeSource defines where a knowledge document originated.
type KnowledgeSource string

const (
	KBSourceSOP             KnowledgeSource = "sop"
	KBSourceIncidentCase    KnowledgeSource = "incident_case"
	KBSourceRunbook         KnowledgeSource = "runbook"
	KBSourceTemplateExample KnowledgeSource = "template_example"
	KBSourceWiki            KnowledgeSource = "wiki"
)

// KnowledgeDocument is a searchable knowledge base entry (SOP, incident case, runbook, etc.).
type KnowledgeDocument struct {
	BaseModel
	Source       KnowledgeSource `json:"source" gorm:"size:50;not null"`
	Title        string          `json:"title" gorm:"size:255;not null"`
	Content      string          `json:"content" gorm:"type:mediumtext;not null"`
	Summary      string          `json:"summary" gorm:"type:text"`
	Tags         JSONLabels      `json:"tags" gorm:"type:json"`
	SourceRef    string          `json:"source_ref" gorm:"size:255"`
	OwnerID      *uint           `json:"owner_id" gorm:"index"`
	Status       string          `json:"status" gorm:"size:20;default:active;index"`
	ViewCount    int             `json:"view_count" gorm:"default:0"`
	HelpfulCount int             `json:"helpful_count" gorm:"default:0"`
}

func (KnowledgeDocument) TableName() string { return "knowledge_documents" }
