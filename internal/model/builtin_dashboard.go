package model

// BuiltinDashboard represents a built-in dashboard template that can be
// imported into a user's dashboard collection. Modeled after Nightingale's
// BuiltinPayload with type/component/cate taxonomy.
type BuiltinDashboard struct {
	BaseModel
	Name      string `json:"name" gorm:"size:256;not null;default:''"`
	Ident     string `json:"ident" gorm:"size:128;not null;default:'';uniqueIndex"`
	Category  string `json:"category" gorm:"size:64;not null;default:'';index"`
	Component string `json:"component" gorm:"size:64;not null;default:'';index"`
	Tags      string `json:"tags" gorm:"size:512;default:''"`
	Config    string `json:"config" gorm:"type:longtext"`
	Version   int    `json:"version" gorm:"not null;default:1"`
	BuiltIn   bool   `json:"built_in" gorm:"not null;default:true"`
	CreateBy  string `json:"create_by" gorm:"size:64;default:''"`
}

func (BuiltinDashboard) TableName() string {
	return "builtin_dashboards"
}
