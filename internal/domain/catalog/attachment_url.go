package catalog

// AttachmentURL is a lightweight DTO that catalog API responses embed
// when returning gallery images for studios and rooms.
// It is populated by joining the attachments + uploads tables; never stored directly.
type AttachmentURL struct {
	ID           int64  `json:"id"`
	URL          string `json:"url"`
	OriginalName string `json:"original_name"`
	MimeType     string `json:"mime_type"`
	SortOrder    int    `json:"sort_order"`
	Caption      string `json:"caption,omitempty"`
}
