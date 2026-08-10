package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"image"
	"io"
	"net/http"
)

func LoadImageFromUrl(url string) (image.Image, error) {
	res, err := http.Get(url)
	if err != nil {
		return nil, err
	}

	defer res.Body.Close()

	if res.StatusCode != http.StatusOK {
		return nil, errors.New("Failed to fetch")
	}

	data, err := io.ReadAll(res.Body)
	if err != nil {
		return nil, err
	}

	img, _, err := image.Decode(bytes.NewReader(data))
	if err != nil {
		return nil, err
	}

	return img, err
}

func FetchChapter(chapterId string) (MangedexChapterResult, error) {
	var result MangedexChapterResult

	url := "https://api.mangadex.org/at-home/server/" + chapterId
	res, err := http.Get(url)
	if err != nil {
		return result, err
	}

	defer res.Body.Close()

	if res.StatusCode != http.StatusOK {
		return result, errors.New("Failed to fetch")
	}

	err = json.NewDecoder(res.Body).Decode(&result)
	return result, err
}

func FetchMangaChapters(mangaId string) (MangadexMangaChapterResponse, error) {
	var result MangadexMangaChapterResponse
	url := "https://api.mangadex.org/manga/" + mangaId + "/feed?translatedLanguage[]=en&limit=96&includes[]=scanlation_group&includes[]=user&order[volume]=desc&order[chapter]=desc&offset=0&contentRating[]=safe&contentRating[]=suggestive&contentRating[]=erotica&contentRating[]=pornographic&includeUnavailable=0"
	res, err := http.Get(url)
	if err != nil {
		return result, err
	}

	defer res.Body.Close()

	if res.StatusCode != http.StatusOK {
		return result, errors.New("Failed to fetch")
	}

	err = json.NewDecoder(res.Body).Decode(&result)
	if err != nil {
		return result, err
	}

	return result, nil
}

func FetchManga(mangaId string) (MangadexManga, error) {
	var result MangadexManga

	url := "https://api.mangadex.org/manga/" + mangaId + "?includes[]=cover_art"
	res, err := http.Get(url)
	if err != nil {
		return result, err
	}

	defer res.Body.Close()

	if res.StatusCode != http.StatusOK {
		return result, errors.New("Failed to fetch")
	}

	err = json.NewDecoder(res.Body).Decode(&result)
	return result, err
}
