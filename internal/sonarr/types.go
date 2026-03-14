package sonarr

type Series struct {
	ID               int    `json:"id"`
	Title            string `json:"title"`
	Year             int    `json:"year"`
	TVDBID           int    `json:"tvdbId"`
	Status           string `json:"status"`
	Overview         string `json:"overview"`
	Monitored        bool   `json:"monitored"`
	SeasonCount      int    `json:"seasonCount"`
	RootFolderPath   string `json:"rootFolderPath"`
	QualityProfileID int    `json:"qualityProfileId"`
}

type AddSeriesRequest struct {
	Title            string `json:"title"`
	TVDBID           int    `json:"tvdbId"`
	QualityProfileID int    `json:"qualityProfileId"`
	RootFolderPath   string `json:"rootFolderPath"`
	Monitored        bool   `json:"monitored"`
	SeasonFolder     bool   `json:"seasonFolder"`
}

type QueuePage struct {
	Page         int         `json:"page"`
	PageSize     int         `json:"pageSize"`
	TotalRecords int         `json:"totalRecords"`
	Records      []QueueItem `json:"records"`
}

type QueueItem struct {
	ID                    int     `json:"id"`
	SeriesID              int     `json:"seriesId"`
	Title                 string  `json:"title"`
	Status                string  `json:"status"`
	TrackedDownloadStatus string  `json:"trackedDownloadStatus"`
	TrackedDownloadState  string  `json:"trackedDownloadState"`
	Sizeleft              float64 `json:"sizeleft"`
	Size                  float64 `json:"size"`
}

type HistoryPage struct {
	Page         int             `json:"page"`
	PageSize     int             `json:"pageSize"`
	TotalRecords int             `json:"totalRecords"`
	Records      []HistoryRecord `json:"records"`
}

type HistoryRecord struct {
	ID          int    `json:"id"`
	SeriesID    int    `json:"seriesId"`
	EpisodeID   int    `json:"episodeId"`
	SourceTitle string `json:"sourceTitle"`
	EventType   string `json:"eventType"`
	Date        string `json:"date"`
}

type HealthCheck struct {
	Source  string `json:"source"`
	Type    string `json:"type"`
	Message string `json:"message"`
}
