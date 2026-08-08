package main

// Manga
type MangadexManga struct {
	Result   string        `json:"result"`
	Response string        `json:"response"`
	Data     MangadexMangaData `json:"data"`
}

type MangadexMangaData struct {
	Id            string                          `json:"id"`
	Type          string                          `json:"type"`
	Attributes    MangadexMangaAttributes         `json:"attributes"`
	Relationships []MangadexMangaRelationship     `json:"relationships"`
}

type MangadexMangaAttributes struct {
	Title map[string]string `json:"title"`
	Description map[string]string `json:"description"`
}

type MangadexMangaRelationship struct {
	Id         string                        `json:"id"`
	Type       string                        `json:"type"`
	Related    string                        `json:"related,omitempty"`
	Attributes *MangadexCoverArtAttributes   `json:"attributes,omitempty"`
}

type MangadexCoverArtAttributes struct {
	Description string `json:"description"`
	Volume      string `json:"volume"`
	FileName    string `json:"fileName"`
	Locale      string `json:"locale"`
	CreatedAt   string `json:"createdAt"`
	UpdatedAt   string `json:"updatedAt"`
	Version     int    `json:"version"`
}

// Manga Chapter
type MangadexMangaChapterResponse struct {
	Result   string            `json:"result"`
	Response string            `json:"response"`
	Data     []MangadexMangaChapterData `json:"data"`
	Limit    int               `json:"limit"`
	Offset   int               `json:"offset"`
	Total    int               `json:"total"`
}

type MangadexMangaChapterData struct {
	ID         string                    `json:"id"`
	Type       string                    `json:"type"`
	Attributes MangadexMangaChapterAttributes `json:"attributes"`
}

type MangadexMangaChapterAttributes struct {
	Volume  string `json:"volume"`
	Chapter string `json:"chapter"`
	Title   string `json:"title"`
}

// Chapter
type MangedexChapter struct {
	Hash string `json:"hash"`
	Data []string `json:"data"`
	DataSaver []string `json:"dataSaver"`
}
type MangedexChapterResult struct {
	Result string `json:"result"`
	BaseUrl string `json:"baseUrl"`
	Chapter MangedexChapter
}

