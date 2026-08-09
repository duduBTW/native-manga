package main

import (
	"net/http"
	"image"
	"errors"
	"encoding/json"
	"io"
	"bytes"
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
