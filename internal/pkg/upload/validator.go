package upload

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"mime"
	"net/http"
	"path/filepath"
	"strings"
)

const maxSniffBytes = 512 // http.DetectContentType reads at most 512 bytes

// AllowedType describes one allowed upload type with its file extension(s) and MIME prefix.
type AllowedType struct {
	Extensions []string // e.g. [".yaml", ".yml"]
	MIMEPrefix string   // e.g. "text/yaml", "application/x-yaml", "text/plain"
}

// ValidateUpload checks that an uploaded file's extension and detected MIME type
// are in the allowed set. It reads up to maxSniffBytes from r and returns the
// consumed bytes wrapped in a new reader for downstream consumption.
//
// allowedExts is the set of permitted file extensions (e.g. ".yaml", ".json").
// allowedMIMEs is the set of permitted MIME prefixes (e.g. "text/", "application/json").
func ValidateUpload(filename string, r io.Reader, allowedExts []string, allowedMIMEs []string) (io.Reader, error) {
	ext := strings.ToLower(filepath.Ext(filename))

	extAllowed := false
	for _, e := range allowedExts {
		if ext == e {
			extAllowed = true
			break
		}
	}
	if !extAllowed {
		return nil, fmt.Errorf("file extension %q is not allowed", ext)
	}

	// Read a sniff buffer to detect content type.
	buf := make([]byte, maxSniffBytes)
	n, err := io.ReadFull(r, buf)
	if err != nil && err != io.EOF && err != io.ErrUnexpectedEOF {
		return nil, fmt.Errorf("failed to read file header: %w", err)
	}
	buf = buf[:n]

	detected := http.DetectContentType(buf)
	// http.DetectContentType returns "type/subtype; params".
	mediaType, _, _ := mime.ParseMediaType(detected)
	mediaType = strings.ToLower(mediaType)

	mimeAllowed := false
	for _, m := range allowedMIMEs {
		if strings.HasPrefix(mediaType, m) || mediaType == m {
			mimeAllowed = true
			break
		}
	}
	if !mimeAllowed {
		return nil, fmt.Errorf("file content type %q is not allowed (detected: %s)", mediaType, detected)
	}

	// Also validate that YAML/JSON content is actually parseable.
	ext = strings.ToLower(filepath.Ext(filename))
	if ext == ".json" {
		if !json.Valid(buf) {
			// Read the rest to give the full body for the caller to decide.
			rest, _ := io.ReadAll(r)
			full := append(buf, rest...)
			if !json.Valid(full) {
				return nil, fmt.Errorf("file has .json extension but contains invalid JSON")
			}
			return bytes.NewReader(full), nil
		}
	}

	// Reassemble: sniff buffer + remaining reader.
	return io.MultiReader(bytes.NewReader(buf), r), nil
}

// ValidateYAMLUpload is a convenience wrapper for YAML file uploads.
func ValidateYAMLUpload(filename string, r io.Reader) (io.Reader, error) {
	return ValidateUpload(filename, r,
		[]string{".yaml", ".yml"},
		[]string{"text/", "application/x-yaml", "application/yaml", "application/octet-stream"},
	)
}

// ValidateJSONUpload is a convenience wrapper for JSON file uploads.
func ValidateJSONUpload(filename string, r io.Reader) (io.Reader, error) {
	return ValidateUpload(filename, r,
		[]string{".json"},
		[]string{"application/json", "text/", "application/octet-stream"},
	)
}
