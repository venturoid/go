package stdresp

// Meta represents pagination metadata, with optional fields using omitempty
type Meta struct {
	Page                int         `json:"page"`
	Limit               int         `json:"limit"`
	LastPage            *int        `json:"lastPage,omitempty"`            // Optional
	LastPaginationValue interface{} `json:"lastPaginationValue,omitempty"` // Optional
	Count               *int        `json:"count,omitempty"`               // Optional
}

// NewMeta initializes a Meta object with optional fields omitted
func NewMeta(page, limit int) Meta {
	if page <= 0 {
		page = 1
	}
	if limit <= 0 {
		limit = 10
	}
	return Meta{
		Page:  page,
		Limit: limit,
	}
}

// NewMetaWithLastPage adds lastPage to Meta
func NewMetaWithLastPage(page, limit, lastPage int) Meta {
	meta := NewMeta(page, limit)
	meta.LastPage = &lastPage // Using a pointer to allow omitempty behavior
	return meta
}

// NewMetaWithLastPaginationValue adds lastPaginationValue to Meta
func NewMetaWithLastPaginationValue(lastPaginationValue interface{}) Meta {
	return Meta{
		LastPaginationValue: lastPaginationValue,
	}
}

// NewMetaWithCount adds count to Meta
func NewMetaWithCount(page, limit, count int) Meta {
	meta := NewMeta(page, limit)
	meta.Count = &count // Using a pointer to allow omitempty behavior
	return meta
}
