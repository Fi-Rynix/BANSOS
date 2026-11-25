package model

type WebResponse struct {
	Success   bool      `json:"success"`
	Code      string    `json:"code"`
	Message   string    `json:"message"`
	Data      any       `json:"data"`
	Fields    map[string]string `json:"fields,omitempty"`
	RequestID string    `json:"request_id,omitempty"`
	Meta      any       `json:"meta,omitempty"`
}

type ErrorResponse struct {
	Success   bool              `json:"success"`
	Code      string            `json:"code"`
	Message   string            `json:"message"`
	Fields    map[string]string `json:"fields,omitempty"`
	RequestID string            `json:"request_id,omitempty"`
}

type Meta struct {
	TotalRows int `json:"total_rows"`
	Limit     int `json:"limit"`
	Offset    int `json:"offset"`
}
