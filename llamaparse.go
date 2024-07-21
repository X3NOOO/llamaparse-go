package llamaparse

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"mime/multipart"
	"net/http"
	"os"
	"time"
)

type LlamaParseMode string

const (
	MARKDOWN LlamaParseMode = "markdown"
	TEXT     LlamaParseMode = "text"
	JSON     LlamaParseMode = "json"

	BASE_URL                       = "https://api.cloud.llamaindex.ai"
	DEFAULT_MAX_TIMEOUT_SECONDS    = 2000
	DEFAULT_CHECK_INTERVAL_SECONDS = 1
)

var (
	ErrNoAPIKey       = errors.New("LlamaCloud API key is required")
	ErrEmptyFile      = errors.New("the file cannot be empty")
	ErrParsingFailed  = errors.New("parsing the file failed")
	ErrTimeoutReached = errors.New("timeout reached while parsing the file")

	// sos: https://github.com/run-llama/llama_parse/blob/7515fe5f3ef6757a1859274c1148a56b26254357/llama_parse/utils.py#L102C1-L193C2 + utils/extension_to_mime.py
	SUPPORTED_MIME_TYPES = []string{"application/pdf", "image/cgm", "application/msword", "application/vnd.openxmlformats-officedocument.wordprocessingml.document", "application/vnd.ms-word.document.macroEnabled.12", "text/vnd.graphviz", "application/vnd.ms-word.template.macroEnabled.12", "application/vnd.lotus-wordpro", "application/vnd.apple.pages", "application/vnd.powerbuilder6", "application/vnd.ms-powerpoint", "application/vnd.ms-powerpoint.presentation.macroEnabled.12", "application/vnd.openxmlformats-officedocument.presentationml.presentation", "application/vnd.ms-powerpoint", "application/vnd.ms-powerpoint.template.macroEnabled.12", "application/vnd.openxmlformats-officedocument.presentationml.template", "application/rtf", "application/sdp", "application/vnd.sun.xml.impress.template", "application/vnd.sun.xml.impress", "application/vnd.sun.xml.writer", "application/vnd.sun.xml.writer.template", "application/vnd.sun.xml.writer.global", "text/plain", "application/vnd.wordperfect", "application/vnd.ms-works", "text/xml", "application/epub+zip", "image/jpeg", "image/jpeg", "image/png", "image/gif", "image/bmp", "image/svg+xml", "image/tiff", "image/webp", "text/html", "text/html", "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet", "application/vnd.ms-excel", "application/vnd.ms-excel.sheet.macroEnabled.12", "application/vnd.ms-excel.sheet.binary.macroEnabled.12", "application/vnd.ms-excel", "text/csv", "application/vnd.apple.numbers", "application/vnd.oasis.opendocument.spreadsheet", "application/vnd.dbf", "application/vnd.lotus-1-2-3", "application/vnd.lotus-1-2-3", "application/vnd.lotus-1-2-3", "application/vnd.ms-works", "application/vnd.lotus-1-2-3", "text/tab-separated-values"}
)

func createMultipartRequest(file []byte, language *string) (*bytes.Buffer, string, error) {
	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)

	part, err := writer.CreateFormFile("file", "uploadfile")
	if err != nil {
		return nil, "", err
	}

	_, err = part.Write(file)
	if err != nil {
		return nil, "", err
	}

	if language != nil {
		err = writer.WriteField("language", *language)
		if err != nil {
			return nil, "", err
		}
	}

	contentType := writer.FormDataContentType()
	err = writer.Close()
	if err != nil {
		return nil, "", err
	}

	return body, contentType, nil
}

func getJobResult(apiKey string, baseUrl string, jobID string, mode LlamaParseMode, timeout time.Duration, checkInterval time.Duration) (string, error) {
	client := &http.Client{Timeout: timeout}
	headers := map[string]string{
		"Authorization": "Bearer " + apiKey,
	}
	statusURL := fmt.Sprintf("%s/api/parsing/job/%s", baseUrl, jobID)
	resultURL := fmt.Sprintf("%s/api/parsing/job/%s/result/%s", baseUrl, jobID, mode)

	start := time.Now()
	for {
		if time.Since(start) > timeout {
			return "", ErrTimeoutReached
		}

		time.Sleep(checkInterval)

		req, err := http.NewRequest("GET", statusURL, nil)
		if err != nil {
			return "", err
		}
		for key, value := range headers {
			req.Header.Set(key, value)
		}

		resp, err := client.Do(req)
		if err != nil {
			return "", err
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			continue
		}

		var statusResponse map[string]interface{}
		err = json.NewDecoder(resp.Body).Decode(&statusResponse)
		if err != nil {
			return "", err
		}

		status, ok := statusResponse["status"].(string)
		if !ok || status != "SUCCESS" {
			continue
		}

		req, err = http.NewRequest("GET", resultURL, nil)
		if err != nil {
			return "", err
		}
		for key, value := range headers {
			req.Header.Set(key, value)
		}

		resp, err = client.Do(req)
		if err != nil {
			return "", err
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			return "", ErrParsingFailed
		}

		var resultResponse map[string]interface{}
		err = json.NewDecoder(resp.Body).Decode(&resultResponse)
		if err != nil {
			return "", err
		}

		result, ok := resultResponse[string(mode)].(string)
		if !ok {
			return "", ErrParsingFailed
		}

		return result, nil
	}
}

/*
Parse a file using the LlamaParse API.

Args:

	file: The file to parse.
	mode: The output format (markdown, text, json).
	apiKeyOptional: The LlamaCloud API key. If not provided, it will be read from the LLAMA_CLOUD_API_KEY environment variable.
	languageOptional: The language of the file. If not provided, it will be detected automatically.
	timeoutSecondsOptional: The maximum time to wait for the parsing to finish. Default is 2000 seconds.
	checkIntervalSecondsOptional: The interval between checking the parsing status. Default is 1 second.

Returns:

	The parsed file.
*/
func Parse(file []byte, mode LlamaParseMode, apiKeyOptional *string, languageOptional *string, timeoutSecondsOptional *int, checkIntervalSecondsOptional *int) (string, error) {
	if len(file) == 0 {
		return "", ErrEmptyFile
	}

	var apiKey string

	if apiKeyOptional != nil {
		apiKey = *apiKeyOptional
	} else {
		apiKey = os.Getenv("LLAMA_CLOUD_API_KEY")
		if apiKey == "" {
			return "", ErrNoAPIKey
		}
	}

	url := fmt.Sprintf("%s/api/parsing/upload", BASE_URL)

	body, contentType, err := createMultipartRequest(file, languageOptional)
	if err != nil {
		return "", err
	}

	req, err := http.NewRequest("POST", url, body)
	if err != nil {
		return "", err
	}

	req.Header.Set("Authorization", "Bearer "+apiKey)
	req.Header.Set("Content-Type", contentType)

	var timeoutSeconds int

	if timeoutSecondsOptional != nil {
		timeoutSeconds = *timeoutSecondsOptional
	} else {
		timeoutSeconds = DEFAULT_MAX_TIMEOUT_SECONDS
	}

	client := &http.Client{Timeout: time.Duration(timeoutSeconds) * time.Second}

	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", ErrParsingFailed
	}

	var response map[string]interface{}
	err = json.NewDecoder(resp.Body).Decode(&response)
	if err != nil {
		return "", err
	}

	jobID, ok := response["id"].(string)
	if !ok {
		return "", ErrParsingFailed
	}

	var checkIntervalSeconds int
	if checkIntervalSecondsOptional != nil {
		checkIntervalSeconds = *checkIntervalSecondsOptional
	} else {
		checkIntervalSeconds = DEFAULT_CHECK_INTERVAL_SECONDS
	}

	result, err := getJobResult(apiKey, BASE_URL, jobID, mode, time.Duration(timeoutSeconds)*time.Second, time.Duration(checkIntervalSeconds)*time.Second)
	if err != nil {
		return "", err
	}

	return result, nil
}
