package model

// MemberBlockState is the canonical directional block state exposed to clients.
type MemberBlockState struct {
	Name        string  `json:"name,omitempty"`
	PhotoURL    *string `json:"photoUrl,omitempty"`
	Cohort      string  `json:"cohort,omitempty"`
	Department  string  `json:"department,omitempty"`
	UserSeq     int     `json:"userSeq"`
	BlockedByMe bool    `json:"blockedByMe"`
	UpdatedAt   *string `json:"updatedAt"`
}

// MemberBlockListResponse is the canonical non-paginated block collection.
type MemberBlockListResponse struct {
	Items []MemberBlockState `json:"items"`
}
