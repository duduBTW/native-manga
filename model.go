package main

import "errors"

// Browse
type MangadexMangaCollection struct {
	Result   string              `json:"result"`
	Response string              `json:"response"`
	Data     []MangadexMangaData `json:"data"`
	Limit    int                 `json:"limit"`
	Offset   int                 `json:"offset"`
	Total    int                 `json:"total"`
}

// Manga
type MangadexManga struct {
	Result   string            `json:"result"`
	Response string            `json:"response"`
	Data     MangadexMangaData `json:"data"`
}

func (m *MangadexManga) CoverArtImageUrl() (string, error) {
	return m.Data.CoverArtImageUrl()
}

type MangadexMangaData struct {
	Id            string                      `json:"id"`
	Type          string                      `json:"type"`
	Attributes    MangadexMangaAttributes     `json:"attributes"`
	Relationships []MangadexMangaRelationship `json:"relationships"`
}

func (d *MangadexMangaData) CoverArtImageUrl() (string, error) {
	baseUrl := "https://mangadex.org/covers/"
	coverArtFileName := ""
	for _, relationship := range d.Relationships {
		if relationship.Type == "cover_art" && relationship.Attributes != nil {
			coverArtFileName = relationship.Attributes.FileName
		}
	}
	if coverArtFileName == "" {
		return "", errors.New("Cover art not found")
	}
	return baseUrl + d.Id + "/" + coverArtFileName, nil
}

type MangadexMangaAttributes struct {
	Title       map[string]string `json:"title"`
	Description map[string]string `json:"description"`
}

type MangadexMangaRelationship struct {
	Id         string                      `json:"id"`
	Type       string                      `json:"type"`
	Related    string                      `json:"related,omitempty"`
	Attributes *MangadexCoverArtAttributes `json:"attributes,omitempty"`
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
	Result   string                     `json:"result"`
	Response string                     `json:"response"`
	Data     []MangadexMangaChapterData `json:"data"`
	Limit    int                        `json:"limit"`
	Offset   int                        `json:"offset"`
	Total    int                        `json:"total"`
}

type MangadexMangaChapterData struct {
	Id         string                         `json:"id"`
	Type       string                         `json:"type"`
	Attributes MangadexMangaChapterAttributes `json:"attributes"`
}

type MangadexMangaChapterAttributes struct {
	Volume  string `json:"volume"`
	Chapter string `json:"chapter"`
	Title   string `json:"title"`
}

// Chapter
type MangedexChapter struct {
	Hash      string   `json:"hash"`
	Data      []string `json:"data"`
	DataSaver []string `json:"dataSaver"`
}
type MangedexChapterResult struct {
	Result  string `json:"result"`
	BaseUrl string `json:"baseUrl"`
	Chapter MangedexChapter
}
