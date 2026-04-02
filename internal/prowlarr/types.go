package prowlarr

type Indexer struct {
	ID       int      `json:"id"`
	Name     string   `json:"name"`
	Enable   bool     `json:"enable"`
	Protocol string   `json:"protocol"`
	Priority int      `json:"priority"`
	Fields   []Field  `json:"fields"`
	Tags     []int    `json:"tags"`
}

type Field struct {
	Name  string `json:"name"`
	Value any    `json:"value"`
}

type IndexerStats struct {
	Indexers []IndexerStatEntry `json:"indexers"`
}

type IndexerStatEntry struct {
	IndexerID            int     `json:"indexerId"`
	IndexerName          string  `json:"indexerName"`
	AverageResponseTime  float64 `json:"averageResponseTime"`
	NumberOfQueries      int     `json:"numberOfQueries"`
	NumberOfGrabs        int     `json:"numberOfGrabs"`
	NumberOfFailedQueries int    `json:"numberOfFailedQueries"`
}

type HealthCheck struct {
	Source  string `json:"source"`
	Type    string `json:"type"`
	Message string `json:"message"`
}

type SearchResult struct {
	GUID        string     `json:"guid"`
	IndexerID   int        `json:"indexerId"`
	Title       string     `json:"title"`
	Size        int64      `json:"size"`
	PublishDate string     `json:"publishDate"`
	DownloadURL string     `json:"downloadUrl"`
	Categories  []Category `json:"categories"`
}

type Category struct {
	ID   int    `json:"id"`
	Name string `json:"name"`
}
