package shared

// PaginationMeta is the standard pagination metadata for list responses.
type PaginationMeta struct {
	CurrentPage int   `json:"current_page"`
	NextPage    *int  `json:"next_page"`
	PrevPage    *int  `json:"prev_page"`
	TotalPages  int   `json:"total_pages"`
	TotalCount  int64 `json:"total_count"`
}
