package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"image"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

func LoadImageFromUrl(url string, ctx context.Context) (image.Image, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, err
	}

	res, err := http.DefaultClient.Do(req)
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

func FetchChapter(chapterID string, ctx context.Context) (MangedexChapterResult, error) {
	var result MangedexChapterResult

	url := "https://api.mangadex.org/at-home/server/" + chapterID
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return result, err
	}

	res, err := http.DefaultClient.Do(req)
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

func FetchMangaChapters(mangaID string, ctx context.Context) (MangadexMangaChapterResponse, error) {
	var result MangadexMangaChapterResponse
	url := "https://api.mangadex.org/manga/" + mangaID + "/feed?translatedLanguage[]=en&limit=396&includes[]=scanlation_group&includes[]=user&order[volume]=desc&order[chapter]=desc&offset=0&contentRating[]=safe&contentRating[]=suggestive&contentRating[]=erotica&contentRating[]=pornographic&includeUnavailable=0"

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return result, err
	}

	res, err := http.DefaultClient.Do(req)
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

func FetchManga(mangaID string, ctx context.Context) (MangadexManga, error) {
	var result MangadexManga

	url := "https://api.mangadex.org/manga/" + mangaID + "?includes[]=cover_art"

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return result, err
	}

	res, err := http.DefaultClient.Do(req)
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

func FetchPopularNewTitles(ctx context.Context, searchValue string) (MangadexMangaCollection, error) {
	var result MangadexMangaCollection

	url := "https://api.mangadex.org/manga?limit=40&offset=0&includes[]=cover_art&contentRating[]=safe&contentRating[]=suggestive&contentRating[]=pornographic&includedTagsMode=AND&excludedTagsMode=OR"

	if searchValue != "" {
		url += "&title=" + searchValue
	}

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return result, err
	}

	res, err := http.DefaultClient.Do(req)
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

var (
)

func MangadexOpIDConnect(username, password string, ctx context.Context) (MangadexAuth, error) {
	var result MangadexAuth

	endpoint := "https://auth.mangadex.org/realms/mangadex/protocol/openid-connect/token"

	form := url.Values{}
	form.Set("grant_type", "password")
	form.Set("username", username)
	form.Set("password", password)
	form.Set("client_id", clientID)
	form.Set("client_secret", clientSecret)

	req, err := http.NewRequestWithContext(ctx, "POST", endpoint, strings.NewReader(form.Encode()))
	if err != nil {
		return result, err
	}

	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return result, err
	}

	defer res.Body.Close()

	if res.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(res.Body)
		return result, fmt.Errorf("mangadex auth failed: status %d, body: %s", res.StatusCode, string(body))
	}

	err = json.NewDecoder(res.Body).Decode(&result)
	result.CreatedAt = time.Now()
	return result, err
}
