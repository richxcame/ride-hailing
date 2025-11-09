package security

import (
	"html"
	"regexp"
	"strings"
	"unicode"
)

var (
	// SQL injection patterns (in addition to using parameterized queries)
	sqlInjectionPatterns = []*regexp.Regexp{
		regexp.MustCompile(`(?i)(union\s+select|insert\s+into|delete\s+from|drop\s+table|update\s+.*set)`),
		regexp.MustCompile(`(?i)(exec\s*\(|execute\s*\(|script\s*>|javascript:)`),
	}

	// XSS patterns
	xssPatterns = []*regexp.Regexp{
		regexp.MustCompile(`(?i)<script[^>]*>.*?</script>`),
		regexp.MustCompile(`(?i)<iframe[^>]*>.*?</iframe>`),
		regexp.MustCompile(`(?i)on\w+\s*=`), // onclick, onload, etc.
		regexp.MustCompile(`(?i)javascript:`),
		regexp.MustCompile(`(?i)<embed[^>]*>`),
		regexp.MustCompile(`(?i)<object[^>]*>`),
	}

	// Allowed HTML tags for rich text (if needed)
	allowedHTMLTags = map[string]bool{
		"b": true, "i": true, "u": true, "em": true, "strong": true,
		"p": true, "br": true, "span": true,
	}
)

// SanitizeString removes potentially dangerous characters and patterns from input
func SanitizeString(input string) string {
	// Trim whitespace
	input = strings.TrimSpace(input)

	// Remove null bytes
	input = strings.ReplaceAll(input, "\x00", "")

	// Remove control characters except newlines and tabs
	input = removeControlCharacters(input)

	return input
}

// SanitizeHTML sanitizes HTML input by encoding special characters
func SanitizeHTML(input string) string {
	// HTML encode the entire string
	return html.EscapeString(input)
}

// SanitizeForSQL sanitizes input for SQL (note: always use parameterized queries!)
// This is a defense-in-depth measure, not a replacement for parameterized queries
func SanitizeForSQL(input string) string {
	input = SanitizeString(input)

	// Check for SQL injection patterns
	for _, pattern := range sqlInjectionPatterns {
		if pattern.MatchString(input) {
			// Log this as a potential attack attempt
			// For now, we'll just strip the dangerous parts
			input = pattern.ReplaceAllString(input, "")
		}
	}

	return input
}

// SanitizeForXSS removes XSS attack vectors
func SanitizeForXSS(input string) string {
	// First, sanitize basic input
	input = SanitizeString(input)

	// Remove XSS patterns
	for _, pattern := range xssPatterns {
		input = pattern.ReplaceAllString(input, "")
	}

	// HTML encode the result
	input = html.EscapeString(input)

	return input
}

// SanitizeEmail sanitizes and normalizes email addresses
func SanitizeEmail(email string) string {
	email = strings.TrimSpace(email)
	email = strings.ToLower(email)

	// Remove any characters that aren't valid in emails
	validEmailChars := regexp.MustCompile(`[^a-z0-9._%+\-@]`)
	email = validEmailChars.ReplaceAllString(email, "")

	return email
}

// SanitizePhone sanitizes and normalizes phone numbers
func SanitizePhone(phone string) string {
	// Remove all non-digit characters except +
	validPhoneChars := regexp.MustCompile(`[^\d+]`)
	phone = validPhoneChars.ReplaceAllString(phone, "")

	return phone
}

// SanitizeAlphanumeric keeps only alphanumeric characters
func SanitizeAlphanumeric(input string) string {
	var result strings.Builder
	for _, r := range input {
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			result.WriteRune(r)
		}
	}
	return result.String()
}

// SanitizeFilename sanitizes filename to prevent directory traversal
func SanitizeFilename(filename string) string {
	// Remove path separators
	filename = strings.ReplaceAll(filename, "/", "")
	filename = strings.ReplaceAll(filename, "\\", "")
	filename = strings.ReplaceAll(filename, "..", "")

	// Remove special characters
	validFilenameChars := regexp.MustCompile(`[^a-zA-Z0-9._\-]`)
	filename = validFilenameChars.ReplaceAllString(filename, "_")

	// Limit length
	if len(filename) > 255 {
		filename = filename[:255]
	}

	return filename
}

// SanitizeURL sanitizes URL input
func SanitizeURL(url string) string {
	url = strings.TrimSpace(url)

	// Only allow http and https protocols
	if !strings.HasPrefix(url, "http://") && !strings.HasPrefix(url, "https://") {
		return ""
	}

	// Check for javascript: protocol
	if strings.Contains(strings.ToLower(url), "javascript:") {
		return ""
	}

	return url
}

// StripHTMLTags removes all HTML tags from input
func StripHTMLTags(input string) string {
	htmlTagsRegex := regexp.MustCompile(`<[^>]*>`)
	return htmlTagsRegex.ReplaceAllString(input, "")
}

// StripNonAllowedHTMLTags removes HTML tags except those in the allowed list
func StripNonAllowedHTMLTags(input string) string {
	// This is a simple implementation - for production use a proper HTML sanitizer library
	htmlTagRegex := regexp.MustCompile(`<(/?)(\w+)([^>]*)>`)

	return htmlTagRegex.ReplaceAllStringFunc(input, func(tag string) string {
		matches := htmlTagRegex.FindStringSubmatch(tag)
		if len(matches) > 2 {
			tagName := strings.ToLower(matches[2])
			if allowedHTMLTags[tagName] {
				// Keep allowed tags but strip attributes
				if matches[1] == "/" {
					return "</" + tagName + ">"
				}
				return "<" + tagName + ">"
			}
		}
		return ""
	})
}

// removeControlCharacters removes control characters except newlines and tabs
func removeControlCharacters(input string) string {
	var result strings.Builder
	for _, r := range input {
		// Keep printable characters, newlines, and tabs
		if unicode.IsPrint(r) || r == '\n' || r == '\t' {
			result.WriteRune(r)
		}
	}
	return result.String()
}

// TruncateString truncates a string to a maximum length
func TruncateString(input string, maxLength int) string {
	if len(input) <= maxLength {
		return input
	}
	return input[:maxLength]
}

// NormalizeWhitespace normalizes whitespace in a string
func NormalizeWhitespace(input string) string {
	// Replace multiple spaces with single space
	whitespaceRegex := regexp.MustCompile(`\s+`)
	input = whitespaceRegex.ReplaceAllString(input, " ")

	// Trim leading/trailing whitespace
	return strings.TrimSpace(input)
}

// ContainsSQLInjection checks if input contains potential SQL injection patterns
func ContainsSQLInjection(input string) bool {
	for _, pattern := range sqlInjectionPatterns {
		if pattern.MatchString(input) {
			return true
		}
	}
	return false
}

// ContainsXSS checks if input contains potential XSS patterns
func ContainsXSS(input string) bool {
	for _, pattern := range xssPatterns {
		if pattern.MatchString(input) {
			return true
		}
	}
	return false
}

// SanitizeInput is a general-purpose sanitizer for user input
func SanitizeInput(input string, maxLength int) string {
	input = SanitizeString(input)
	input = SanitizeForXSS(input)
	input = SanitizeForSQL(input)
	input = NormalizeWhitespace(input)
	if maxLength > 0 {
		input = TruncateString(input, maxLength)
	}
	return input
}

// SanitizeUserInput sanitizes common user input fields
type UserInput struct {
	Email       string
	Phone       string
	Name        string
	Description string
	URL         string
}

// Sanitize sanitizes all fields in UserInput
func (u *UserInput) Sanitize() {
	u.Email = SanitizeEmail(u.Email)
	u.Phone = SanitizePhone(u.Phone)
	u.Name = SanitizeInput(u.Name, 100)
	u.Description = SanitizeInput(u.Description, 1000)
	u.URL = SanitizeURL(u.URL)
}
