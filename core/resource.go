package core

type ResourceID int64

type ResourceType string

type ResourceStatus string

type Resource struct {
	ID          ResourceID     `json:"id"`
	SiteID      int64          `json:"site_id"`
	ParentID    *ResourceID    `json:"parent_id,omitempty"`
	Type        ResourceType   `json:"type"`
	Template    string         `json:"template"`
	Title       string         `json:"title"`
	Alias       string         `json:"alias"`
	Path        string         `json:"path"`
	Sort        int            `json:"sort"`
	IsPublished bool           `json:"is_published"`
	Settings    map[string]any `json:"settings"`
	SEO         map[string]any `json:"seo"`
}
