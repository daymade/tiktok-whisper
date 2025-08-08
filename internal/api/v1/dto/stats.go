package dto

// UserStats represents user statistics
type UserStats struct {
	User             string `json:"user"`
	TotalTranscripts int    `json:"totalTranscripts"`
	GeminiEmbeddings int    `json:"geminiEmbeddings"`
	OpenAIEmbeddings int    `json:"openaiEmbeddings"`
}

// SystemStats represents system-wide statistics
type SystemStats struct {
	TotalTranscripts  int         `json:"totalTranscripts"`
	GeminiEmbeddings  int         `json:"geminiEmbeddings"`
	OpenAIEmbeddings  int         `json:"openaiEmbeddings"`
	PendingProcessing int         `json:"pendingProcessing"`
	TopUsers          []UserStats `json:"topUsers"`
}

// StatsRequest represents parameters for statistics requests
type StatsRequest struct {
	User      string `form:"user" json:"user"`
	StartDate string `form:"startDate" json:"startDate"`
	EndDate   string `form:"endDate" json:"endDate"`
}