package helpers

import (
	"bytes"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/altereitay/FinalProjectBackend/db"
	"github.com/ledongthuc/pdf"
	"github.com/nguyenthenguyen/docx"
	"io"
	"log"
	_ "mime/multipart"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

type jsonResponse struct {
	Error   bool   `json:"error"`
	Message string `json:"message"`
	Data    any    `json:"data,omitempty"`
}

func ReadJSON(w http.ResponseWriter, r http.Request, data any) error {
	maxBytes := 1048576
	r.Body = http.MaxBytesReader(w, r.Body, int64(maxBytes))

	dec := json.NewDecoder(r.Body)
	err := dec.Decode(data)
	if err != nil {
		return err
	}
	err = dec.Decode(&struct{}{})
	if err != io.EOF {
		return errors.New("Body must have only a single JSON value")
	}
	return nil
}

func WriteJSON(w http.ResponseWriter, status int, data any, headers ...http.Header) error {
	out, err := json.Marshal(data)
	if err != nil {
		return err
	}
	if len(headers) > 0 {
		for key, value := range headers[0] {
			w.Header()[key] = value
		}
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_, err = w.Write(out)
	if err != nil {
		return err
	}
	return nil
}

func ErrorJSON(w http.ResponseWriter, err error, status ...int) error {
	statusCode := http.StatusBadRequest
	if len(status) > 0 {
		statusCode = status[0]
	}
	var payload jsonResponse
	payload.Error = true
	payload.Message = err.Error()

	return WriteJSON(w, statusCode, payload)
}

func isAllowedExt(ext string) bool {
	switch strings.ToLower(ext) {
	case ".pdf", ".doc", ".docx", ".txt":
		return true
	default:
		return false
	}
}

func computeSHA256(content string) string {
	sum := sha256.Sum256([]byte(content))
	return fmt.Sprintf("%x", sum)
}

func extractTitleAndContent(path, ext string) (string, string, error) {
	var fullText string
	switch ext {
	case ".txt":
		data, err := os.ReadFile(path)
		if err != nil {
			return "", "", err
		}
		fullText = string(data)

	case ".pdf":
		f, r, err := pdf.Open(path)
		if err != nil {
			return "", "", err
		}
		defer f.Close()
		var buf bytes.Buffer
		textReader, err := r.GetPlainText()
		if err != nil {
			return "", "", err
		}
		io.Copy(&buf, textReader)
		fullText = buf.String()

	case ".doc":
		cmd := exec.Command("soffice", "--headless", "--convert-to", "docx", path, "--outdir", os.TempDir())
		if err := cmd.Run(); err != nil {
			return "", "", err
		}
		newPath := strings.TrimSuffix(path, ".doc") + ".docx"
		defer os.Remove(newPath)
		fallthrough

	case ".docx":
		r, err := docx.ReadDocxFile(path)
		if err != nil {
			return "", "", err
		}
		defer r.Close()
		fullText = r.Editable().GetContent()
	}

	lines := strings.Split(strings.TrimSpace(fullText), "\n")
	if len(lines) == 0 {
		return "", "", nil
	}
	title := strings.TrimSpace(lines[0])
	content := strings.TrimSpace(strings.Join(lines[1:], "\n"))
	return title, content, nil
}

func HandleFile(w http.ResponseWriter, r *http.Request) error {
	fmt.Println("Handling file")
	err := r.ParseMultipartForm(10 << 20)
	if err != nil {
		log.Println("error in parse multipart form")
		return ErrorJSON(w, err)
	}

	file, handler, err := r.FormFile("file")
	if err != nil {
		log.Println("error in open file from http")
		return ErrorJSON(w, err)
	}

	defer file.Close()

	ext := filepath.Ext(handler.Filename)
	if !isAllowedExt(ext) {
		log.Println("error in non-allowed excitation")
		return ErrorJSON(w, errors.New("extinction not allowed"))
	}

	tmpFile, err := os.CreateTemp("", "*"+ext)
	if err != nil {
		log.Println("error in create temp file")
		return ErrorJSON(w, err)
	}
	defer os.Remove(tmpFile.Name())

	if _, err := io.Copy(tmpFile, file); err != nil {
		log.Println("error in copy to temp file")
		return ErrorJSON(w, err)
	}
	tmpFile.Close()

	// Extract title and content
	title, content, err := extractTitleAndContent(tmpFile.Name(), ext)
	if err != nil {
		log.Println("error in extracting title and content")
		return ErrorJSON(w, err)
	}

	filename := strings.TrimSuffix(handler.Filename, ext) + "-original" + ext
	savePath := filepath.Join("./original", filename)
	dst, err := os.Create(savePath)
	if err != nil {
		return ErrorJSON(w, err)
	}
	defer dst.Close()

	file.Seek(0, io.SeekStart)
	io.Copy(dst, file)

	sha := computeSHA256(content)

	mongoEntry := db.Article{
		Title:    title,
		Original: content,
		Hash:     sha,
	}

	db.InsertNewArticle(mongoEntry)
	return WriteJSON(w, 201, "New Article Received")
}
