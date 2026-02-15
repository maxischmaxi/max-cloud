package cmd

import (
	"errors"
	"fmt"
	"net"
	"strings"

	"github.com/max-cloud/shared/pkg/api"
)

func formatError(err error) error {
	var apiErr *api.APIError
	if errors.As(err, &apiErr) {
		return formatAPIError(apiErr)
	}

	if isConnError(err) {
		return fmt.Errorf("cannot connect to API server at %s\n\nRun 'maxcloud auth register' first or check --api-url", apiURL)
	}

	return err
}

func isConnError(err error) bool {
	var netErr net.Error
	if errors.As(err, &netErr) {
		return true
	}
	errStr := err.Error()
	return strings.Contains(errStr, "connection refused") ||
		strings.Contains(errStr, "no such host") ||
		strings.Contains(errStr, "connection reset")
}

func formatAPIError(err *api.APIError) error {
	switch err.StatusCode {
	case 401:
		return fmt.Errorf("authentication required\n\nRun 'maxcloud auth register' or set --api-key")
	case 403:
		return fmt.Errorf("access denied: %s", err.Message)
	case 404:
		return fmt.Errorf("not found: %s", err.Message)
	case 409:
		return fmt.Errorf("conflict: %s", err.Message)
	case 400:
		return fmt.Errorf("invalid request: %s", err.Message)
	case 500:
		return fmt.Errorf("server error: %s\n\nIf this persists, contact support", err.Message)
	default:
		return fmt.Errorf("API error: %s", err.Message)
	}
}
