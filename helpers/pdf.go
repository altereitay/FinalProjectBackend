package helpers

import (
	"bytes"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"os"

	"github.com/miku/grobidclient/tei"
)

func ExtractTitleAndBody(pdfPath string) (string, string, error) {
	const grobidURL = "http://localhost:8070/api/processFulltextDocument"

	file, err := os.Open(pdfPath)
	if err != nil {
		return "", "", fmt.Errorf("open PDF: %w", err)
	}
	defer file.Close()

	var body bytes.Buffer
	w := multipart.NewWriter(&body)
	part, err := w.CreateFormFile("input", pdfPath)
	if err != nil {
		return "", "", err
	}
	if _, err := io.Copy(part, file); err != nil {
		return "", "", err
	}
	w.Close()

	req, _ := http.NewRequest("POST", grobidURL, &body)
	req.Header.Set("Content-Type", w.FormDataContentType())

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		msg, _ := io.ReadAll(resp.Body)
		return "", "", fmt.Errorf("grobid HTTP %d: %s", resp.StatusCode, msg)
	}

	doc, err := tei.ParseDocument(resp.Body)
	if err != nil {
		return "", "", fmt.Errorf("parse TEI: %w", err)
	}

	title := string(doc.Header.Title)
	text := string(doc.Abstract) + "\n" + string(doc.Body)
	return title, text, nil
}
