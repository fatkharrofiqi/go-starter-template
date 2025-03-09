package dto

// WebResponse is a generic API response structure
type WebResponse[T any] struct {
	Data   T             `json:"data"`             // Holds the main response data
	Paging *PageMetadata `json:"paging,omitempty"` // Pagination details (if applicable)
}

// PageMetadata contains pagination details
type PageMetadata struct {
	Page      int   `json:"page"`       // Current page number
	Size      int   `json:"size"`       // Number of items per page
	TotalItem int64 `json:"total_item"` // Total number of items
	TotalPage int64 `json:"total_page"` // Total pages available
}
