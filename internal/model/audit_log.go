package model

// AuditLog records every write operation performed by a user.
type AuditLog struct {
	BaseModel
	UserID       *uint  `json:"user_id" gorm:"index"`
	Username     string `json:"username" gorm:"size:64;not null;default:''"`
	Action       string `json:"action" gorm:"size:64;not null;index"`
	ResourceType string `json:"resource_type" gorm:"size:64;not null;index"`
	ResourceID   *uint  `json:"resource_id"`
	ResourceName string `json:"resource_name" gorm:"size:256"`
	Detail       string `json:"detail" gorm:"type:text"`
	IP           string `json:"ip" gorm:"size:64"`
	Status       string `json:"status" gorm:"size:16;not null;default:'success'"`
}

func (AuditLog) TableName() string { return "audit_logs" }

// Audit action constants.
const (
	AuditActionCreate  = "create"
	AuditActionUpdate  = "update"
	AuditActionDelete  = "delete"
	AuditActionToggle  = "toggle"
	AuditActionAck     = "acknowledge"
	AuditActionAssign  = "assign"
	AuditActionResolve = "resolve"
	AuditActionClose   = "close"
	AuditActionSilence = "silence"
	AuditActionComment = "comment"
	AuditActionImport  = "import"
)

// Audit resource type constants.
const (
	AuditResourceAlertRule    = "alert_rule"
	AuditResourceAlertEvent   = "alert_event"
	AuditResourceUser         = "user"
	AuditResourceTeam         = "team"
	AuditResourceDatasource   = "datasource"
	AuditResourceNotifyRule   = "notify_rule"
	AuditResourceMuteRule      = "mute_rule"
	AuditResourceSystemSetting = "system_setting"
	AuditResourceUserPassword  = "user_password"
)
