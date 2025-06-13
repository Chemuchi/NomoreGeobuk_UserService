package config

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"os"
)

type imgbbResponse struct {
	Data struct {
		URL string `json:"url"`
	} `json:"data"`
	Success bool `json:"success"`
	Status  int  `json:"status"`
}

func UploadToImgBB(r io.Reader, filename string) (string, error) {
	key := os.Getenv("IMGBB_KEY")
	if key == "" {
		return "", fmt.Errorf("IMGBB_KEY not set")
	}

	var buf bytes.Buffer
	w := multipart.NewWriter(&buf)

	// file part
	fw, err := w.CreateFormFile("image", filename)
	if err != nil {
		return "", err
	}
	if _, err = io.Copy(fw, r); err != nil {
		return "", err
	}

	// optional expiration
	if exp := os.Getenv("IMGBB_EXPIRATION"); exp != "" {
		_ = w.WriteField("expiration", exp)
	}
	w.Close()

	req, err := http.NewRequest("POST",
		"https://api.imgbb.com/1/upload?key="+key,
		&buf)
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", w.FormDataContentType())

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	var res imgbbResponse
	if err := json.NewDecoder(resp.Body).Decode(&res); err != nil {
		return "", err
	}
	if !res.Success {
		return "", fmt.Errorf("imgbb upload failed, status %d", res.Status)
	}
	return res.Data.URL, nil
}
