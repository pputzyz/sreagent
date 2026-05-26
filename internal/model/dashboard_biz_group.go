package model

// DashboardBizGroup links a dashboard to a business group with a permission flag.
type DashboardBizGroup struct {
	BaseModel
	DashboardID uint   `gorm:"uniqueIndex:idx_did_bgid" json:"dashboard_id"`
	BizGroupID  uint   `gorm:"uniqueIndex:idx_did_bgid" json:"biz_group_id"`
	PermFlag    string `gorm:"size:4;default:'ro'" json:"perm_flag"` // ro or rw
}

func (DashboardBizGroup) TableName() string {
	return "dashboard_biz_groups"
}
