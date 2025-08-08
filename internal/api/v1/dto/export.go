package dto

// ExportRequest represents parameters for export requests
type ExportRequest struct {
	Format    string `form:"format" json:"format" binding:"required,oneof=csv json xlsx"`
	User      string `form:"user" json:"user"`
	StartDate string `form:"startDate" json:"startDate"`
	EndDate   string `form:"endDate" json:"endDate"`
	Limit     int    `form:"limit" json:"limit"`
}