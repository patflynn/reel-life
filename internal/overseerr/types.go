package overseerr

type Request struct {
	ID          int          `json:"id"`
	Status      int          `json:"status"`
	Type        string       `json:"type"`
	Media       MediaInfo    `json:"media"`
	RequestedBy UserInfo     `json:"requestedBy"`
	CreatedAt   string       `json:"createdAt"`
}

type MediaInfo struct {
	TMDBID int `json:"tmdbId"`
	TVDBID int `json:"tvdbId"`
	Status int `json:"status"`
}

type UserInfo struct {
	ID          int    `json:"id"`
	DisplayName string `json:"displayName"`
	Email       string `json:"email"`
}

type RequestPage struct {
	PageInfo PageInfo  `json:"pageInfo"`
	Results  []Request `json:"results"`
}

type PageInfo struct {
	Pages   int `json:"pages"`
	Page    int `json:"page"`
	Results int `json:"results"`
}

type RequestCount struct {
	Pending  int `json:"pending"`
	Approved int `json:"approved"`
	Declined int `json:"declined"`
	Total    int `json:"total"`
}

type SearchResults struct {
	Page         int            `json:"page"`
	TotalPages   int            `json:"totalPages"`
	TotalResults int            `json:"totalResults"`
	Results      []SearchResult `json:"results"`
}

type SearchResult struct {
	ID        int    `json:"id"`
	MediaType string `json:"mediaType"`
	Title     string `json:"title"`
	Name      string `json:"name"`
	Overview  string `json:"overview"`
	PosterPath string `json:"posterPath"`
}
