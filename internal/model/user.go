package model

// Role defines user roles within the platform.
type Role string

const (
	RoleAdmin        Role = "admin"
	RoleTeamLead     Role = "team_lead"
	RoleMember       Role = "member"
	RoleViewer       Role = "viewer"
	RoleGlobalViewer Role = "global_viewer" // can view all alerts but not manage
)

// IsValid returns true if the role is a recognized value.
func (r Role) IsValid() bool {
	switch r {
	case RoleAdmin, RoleTeamLead, RoleMember, RoleViewer, RoleGlobalViewer:
		return true
	}
	return false
}

// UserType distinguishes human users from non-human entities.
type UserType string

const (
	UserTypeHuman   UserType = "human"
	UserTypeBot     UserType = "bot"     // Lark group bot / virtual receiver
	UserTypeChannel UserType = "channel" // alert channel entity
)

// User represents a platform user.
type User struct {
	BaseModel
	Username     string   `json:"username" gorm:"uniqueIndex;size:64;not null"`
	Password     string   `json:"-" gorm:"size:256;not null"`
	DisplayName  string   `json:"display_name" gorm:"size:128"`
	Email        string   `json:"email" gorm:"size:256"`
	Phone        string   `json:"phone" gorm:"size:32"`
	LarkUserID   string   `json:"lark_user_id" gorm:"size:64;index"`
	Avatar       string   `json:"avatar" gorm:"size:512"`
	Role         Role     `json:"role" gorm:"size:32;not null;default:member"`
	IsActive     bool     `json:"is_active"` // create handlers set true explicitly; DB column keeps DEFAULT 1 for seeds
	UserType     UserType `json:"user_type" gorm:"size:32;default:human;index"`
	NotifyTarget string   `json:"notify_target" gorm:"type:text"`               // JSON: for bot type: {"lark_webhook":"https://..."}, for channel: {"media_id":1}
	OIDCSubject  string   `json:"oidc_subject,omitempty" gorm:"size:256;index"` // OIDC subject identifier (sub claim)
	Teams        []Team   `json:"teams,omitempty" gorm:"many2many:team_members;"`
}

func (User) TableName() string {
	return "users"
}
