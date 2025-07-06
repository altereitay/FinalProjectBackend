package helpers

import (
	"bytes"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	_ "mime/multipart"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/altereitay/FinalProjectBackend/db"
	"github.com/ledongthuc/pdf"
	"github.com/nguyenthenguyen/docx"
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

func ReadTxt(path string) (string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	fullText := string(data)
	return fullText, nil
}

func readPDF(path string) (string, error) {
	f, r, err := pdf.Open(path)
	if err != nil {
		return "", err
	}
	defer f.Close()
	var buf bytes.Buffer
	sentences, err := r.GetStyledTexts()
	if err != nil {
		return "", err
	}

	for _, sentence := range sentences {
		cleanText := strings.ReplaceAll(sentence.S, "\uFFFD", "")
		buf.WriteString(cleanText)
		buf.WriteString("\n")
	}

	return buf.String(), nil
}

func readDocx(path string) (string, error) {
	r, err := docx.ReadDocxFile(path)
	if err != nil {
		return "", err
	}
	defer r.Close()

	// Extract the raw XML
	content := r.Editable().GetContent()

	// Extract text between <w:t>...</w:t>
	var buf bytes.Buffer
	tokens := strings.Split(content, "<w:t")
	for _, token := range tokens[1:] {
		parts := strings.SplitN(token, ">", 2)
		if len(parts) < 2 {
			continue
		}
		end := strings.Index(parts[1], "</w:t>")
		if end >= 0 {
			buf.WriteString(parts[1][:end])
			buf.WriteString("\n")
		}
	}

	return buf.String(), nil
}

func extractTitleAndContent(path, ext string) (string, string, error) {
	var fullText string
	var err error
	switch ext {
	case ".txt":
		fullText, err = ReadTxt(path)
		if err != nil {
			return "", "", nil
		}

	case ".pdf":
		fullText, err = readPDF(path)
		if err != nil {
			return "", "", nil
		}

	case ".docx":
		fullText, err = readDocx(path)
		if err != nil {
			return "", "", nil
		}
	}

	lines := strings.Split(strings.TrimSpace(fullText), "\n")
	var filteredLines []string
	for _, line := range lines {
		if strings.TrimSpace(line) != "" {
			filteredLines = append(filteredLines, strings.TrimSpace(line))
		}
	}
	if len(filteredLines) == 0 {
		return "", "", nil
	}
	title := filteredLines[0]
	content := strings.Join(filteredLines[1:], "\n")
	return title, content, nil
}

func HandleFile(w http.ResponseWriter, r *http.Request) error {
	err := r.ParseMultipartForm(10 << 20)
	if err != nil {
		log.Printf("error in parsing multi part form: %v", err)
		return ErrorJSON(w, err)
	}

	file, handler, err := r.FormFile("file")
	if err != nil {
		log.Printf("error in getting file from FormData: %v", err)
		return ErrorJSON(w, err)
	}

	defer file.Close()

	ext := filepath.Ext(handler.Filename)
	if !isAllowedExt(ext) {
		return ErrorJSON(w, errors.New("extinction not allowed"))
	}

	tmpFile, err := os.CreateTemp("", "*"+ext)
	if err != nil {
		log.Printf("error in create temp file: %v", err)
		return ErrorJSON(w, err)
	}
	defer os.Remove(tmpFile.Name())

	if _, err := io.Copy(tmpFile, file); err != nil {
		log.Printf("error in io copy: %v", err)
		return ErrorJSON(w, err)
	}
	tmpFile.Close()

	// Extract title and content
	title, content, err := extractTitleAndContent(tmpFile.Name(), ext)
	if err != nil {
		log.Printf("error in extracting title and content: %v", err)
		return ErrorJSON(w, err)
	}

	sha := computeSHA256(content)

	articleExists := db.CheckIfExists(sha)
	if articleExists {
		log.Println("Article already exists")
		payload := jsonResponse{
			Error:   false,
			Message: "Article already exists",
		}
		return WriteJSON(w, 400, payload)
	}

	fileName := "/home/sceuser/articles/" + sha + "-original.txt"
	fileData := title + "\n" + content
	fileCopy, err := os.Create(fileName)
	if err != nil {
		log.Printf("error in creating file: %v", err)
		return ErrorJSON(w, err)
	}
	defer fileCopy.Close()
	_, err = fileCopy.WriteString(fileData)
	if err != nil {
		log.Printf("error in wtrie to file: %v", err)
		return ErrorJSON(w, err)
	}

	mongoEntry := db.Article{
		Title:    title,
		Original: content,
		Hash:     sha,
	}

	err = db.InsertNewArticle(mongoEntry)
	if err != nil {
		log.Printf("error in inserting new article: %v", err)
		return ErrorJSON(w, err)
	}

	data := SimplifiedJSON{
		Hash:   sha,
		Name:   fileName,
		Status: "new",
	}

	dataString, err := json.Marshal(data)

	if err != nil {
		log.Printf("error in marshalling for mqtt: %v", err)
		return ErrorJSON(w, err)
	}

	Publish(SIMPLIFY_TOPIC, dataString)

	payload := jsonResponse{
		Error:   false,
		Message: "New Article Received",
	}

	return WriteJSON(w, 201, payload)
}
