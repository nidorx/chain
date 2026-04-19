// Copyright 2022 Alex Rodin. All rights reserved.
// Based on the httprouter package, Copyright 2013 Julien Schmidt.

package chain

import (
	"errors"
	"fmt"
	"net/http"
	"net/url"
)

// Common error variables for standardized error handling
var (
	// ErrInvalidRoutePath is returned when a route path doesn't start with '/'
	ErrInvalidRoutePath = errors.New("route path must begin with '/'")

	// ErrEmptyRoutePath is returned when a route path is empty
	ErrEmptyRoutePath = errors.New("route path cannot be empty")

	// ErrInvalidRouteMethod is returned when a route method is empty or invalid
	ErrInvalidRouteMethod = errors.New("route method must not be empty")

	// ErrInvalidRouteHandler is returned when a route handler is nil or invalid
	ErrInvalidRouteHandler = errors.New("invalid route handler")

	// ErrRequestBodyTooLarge is returned when the request body exceeds the maximum allowed size
	ErrRequestBodyTooLarge = errors.New("request body too large")

	// ErrInvalidContentType is returned when the Content-Type header is invalid or unsupported
	ErrInvalidContentType = errors.New("invalid or unsupported Content-Type")

	// ErrInvalidHeader is returned when a header value is malformed
	ErrInvalidHeader = errors.New("invalid header format")

	// ErrMissingParameter is returned when a required route parameter is missing
	ErrMissingParameter = errors.New("missing required parameter")

	// ErrInvalidParameter is returned when a route parameter has an invalid value
	ErrInvalidParameter = errors.New("invalid parameter value")

	// ErrInvalidMiddleware is returned when an invalid middleware type is provided
	ErrInvalidMiddleware = errors.New("invalid middleware type")

	// ErrWildcardConflict is returned when two wildcard routes conflict
	ErrWildcardConflict = errors.New("wildcard route conflicts with existing route")

	// ErrEmptyParameterName is returned when a route parameter name is empty
	ErrEmptyParameterName = errors.New("route parameter name cannot be empty")

	// ErrDuplicateWildcard is returned when multiple wildcards exist in a path segment
	ErrDuplicateWildcard = errors.New("only one wildcard per path segment is allowed")

	// ErrWildcardNotAtEnd is returned when a wildcard is not at the end of the path
	ErrWildcardNotAtEnd = errors.New("catch-all routes are only allowed at the end of the path")

	// ErrInvalidParameterName is returned when a parameter name contains invalid characters
	ErrInvalidParameterName = errors.New("parameter name cannot contain wildcards or other parameters")
)

// RouteValidationError represents a validation error for route configuration
type RouteValidationError struct {
	// Field is the field that failed validation
	Field string
	// Value is the invalid value
	Value string
	// Message is a human-readable error message
	Message string
}

func (e *RouteValidationError) Error() string {
	if e.Message != "" {
		return e.Message
	}
	return fmt.Sprintf("route validation error: field '%s' has invalid value '%s'", e.Field, e.Value)
}

// Is implements errors.Is for RouteValidationError
func (e *RouteValidationError) Is(target error) bool {
	_, ok := target.(*RouteValidationError)
	return ok
}

// NewRouteValidationError creates a new RouteValidationError
func NewRouteValidationError(field string, value string, message string) *RouteValidationError {
	return &RouteValidationError{
		Field:   field,
		Value:   value,
		Message: message,
	}
}

// ValidateRoutePath validates a route path and returns an error if invalid
func ValidateRoutePath(path string) error {
	if path == "" {
		return fmt.Errorf("%w: path is empty", ErrEmptyRoutePath)
	}

	if path[0] != '/' {
		return fmt.Errorf("%w: path must begin with '/', got '%s'", ErrInvalidRoutePath, path)
	}

	// Check for double slashes (except at the beginning)
	if len(path) > 2 && path[1] == '/' {
		return fmt.Errorf("%w: path contains double slash: '%s'", ErrInvalidRoutePath, path)
	}

	// Check for path traversal attempts
	if pathContainsPathTraversal(path) {
		return fmt.Errorf("%w: path contains path traversal attempt: '%s'", ErrInvalidRoutePath, path)
	}

	return nil
}

// pathContainsPathTraversal checks if a path contains path traversal attempts
func pathContainsPathTraversal(path string) bool {
	// Check for ../ or ..\ sequences
	for i := 0; i < len(path)-2; i++ {
		if path[i:i+3] == "../" || path[i:i+3] == "..\\" {
			return true
		}
	}
	// Check for trailing ..
	if len(path) >= 2 && path[len(path)-2:] == ".." {
		return true
	}
	return false
}

// ValidateRouteMethod validates a route method and returns an error if invalid
func ValidateRouteMethod(method string) error {
	if method == "" {
		return fmt.Errorf("%w: method is empty", ErrInvalidRouteMethod)
	}

	// Validate against standard HTTP methods
	validMethods := map[string]bool{
		http.MethodGet:     true,
		http.MethodHead:    true,
		http.MethodPost:    true,
		http.MethodPut:     true,
		http.MethodPatch:   true,
		http.MethodDelete:  true,
		http.MethodConnect: true,
		http.MethodOptions: true,
		http.MethodTrace:   true,
	}

	if !validMethods[method] {
		return fmt.Errorf("%w: method '%s' is not a valid HTTP method", ErrInvalidRouteMethod, method)
	}

	return nil
}

// ValidateRouteHandler validates a route handler and returns an error if invalid
func ValidateRouteHandler(handler any) error {
	if handler == nil {
		return fmt.Errorf("%w: handler is nil", ErrInvalidRouteHandler)
	}

	// Check if handler is one of the supported types
	if _, err := Handler(handler); err != nil {
		return fmt.Errorf("%w: %v", ErrInvalidRouteHandler, err)
	}

	return nil
}

// ValidateQueryParameter validates a query parameter value
func ValidateQueryParameter(name string, value string, maxLength int) error {
	if len(value) > maxLength {
		return fmt.Errorf("%w: parameter '%s' exceeds maximum length of %d characters", ErrInvalidParameter, name, maxLength)
	}

	// Check for null bytes
	if containsNullByte(value) {
		return fmt.Errorf("%w: parameter '%s' contains null byte", ErrInvalidParameter, name)
	}

	return nil
}

// ValidateHeaderValue validates a header value
func ValidateHeaderValue(name string, value string, maxLength int) error {
	if len(value) > maxLength {
		return fmt.Errorf("%w: header '%s' exceeds maximum length of %d characters", ErrInvalidHeader, name, maxLength)
	}

	// Check for newlines (header injection prevention)
	if containsNewline(value) {
		return fmt.Errorf("%w: header '%s' contains newline characters", ErrInvalidHeader, name)
	}

	// Check for null bytes
	if containsNullByte(value) {
		return fmt.Errorf("%w: header '%s' contains null byte", ErrInvalidHeader, name)
	}

	return nil
}

// containsNullByte checks if a string contains null bytes
func containsNullByte(s string) bool {
	for i := 0; i < len(s); i++ {
		if s[i] == 0 {
			return true
		}
	}
	return false
}

// containsNewline checks if a string contains newline characters
func containsNewline(s string) bool {
	for i := 0; i < len(s); i++ {
		if s[i] == '\n' || s[i] == '\r' {
			return true
		}
	}
	return false
}

// DefaultMaxQueryParameterLength is the default maximum length for query parameters
const DefaultMaxQueryParameterLength = 1024

// DefaultMaxHeaderLength is the default maximum length for header values
const DefaultMaxHeaderLength = 4096

// DefaultMaxRequestBodySize is the default maximum request body size (10MB)
const DefaultMaxRequestBodySize = 10 << 20

// ValidateRequestBodySize validates the request body size
func ValidateRequestBodySize(contentLength int64, maxSize int64) error {
	if maxSize <= 0 {
		maxSize = DefaultMaxRequestBodySize
	}

	if contentLength > maxSize {
		return fmt.Errorf("%w: size %d exceeds maximum allowed size %d", ErrRequestBodyTooLarge, contentLength, maxSize)
	}

	return nil
}

// SanitizePath cleans a URL path and removes potentially dangerous characters
func SanitizePath(path string) string {
	// Remove null bytes
	path = removeNullBytes(path)

	// Normalize path
	parsed, err := url.Parse(path)
	if err != nil {
		return "/"
	}

	return parsed.Path
}

// removeNullBytes removes null bytes from a string
func removeNullBytes(s string) string {
	result := make([]byte, 0, len(s))
	for i := 0; i < len(s); i++ {
		if s[i] != 0 {
			result = append(result, s[i])
		}
	}
	return string(result)
}

// SanitizeHeaderValue removes potentially dangerous characters from header values
func SanitizeHeaderValue(value string) string {
	// Remove newlines (header injection prevention)
	value = removeNewlines(value)

	// Remove null bytes
	value = removeNullBytes(value)

	return value
}

// removeNewlines removes newline characters from a string
func removeNewlines(s string) string {
	result := make([]byte, 0, len(s))
	for i := 0; i < len(s); i++ {
		if s[i] != '\n' && s[i] != '\r' {
			result = append(result, s[i])
		}
	}
	return string(result)
}
