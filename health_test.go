package main

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestDetermineStatus(t *testing.T) {
	tests := []struct {
		name       string
		statusCode int
		err        error
		url        string
		expected   string
	}{
		{
			name:       "Status 200 is Online",
			statusCode: 200,
			err:        nil,
			url:        "http://api.com",
			expected:   "Online",
		},
		{
			name:       "Status 500 is Offline",
			statusCode: 500,
			err:        nil,
			url:        "http://api.com",
			expected:   "Offline",
		},
		{
			name:       "Status 404 is Offline",
			statusCode: 404,
			err:        nil,
			url:        "http://api.com",
			expected:   "Offline",
		},
		{
			name:       "Connection Error is Offline",
			statusCode: 0,
			err:        errors.New("connection refused"),
			url:        "http://api.com",
			expected:   "Offline",
		},
		{
			name:       "Vercel 401 is Online (Auth required but alive)",
			statusCode: 401,
			err:        nil,
			url:        "https://myapp.vercel.app",
			expected:   "Online",
		},
		{
			name:       "Non-Vercel 401 is Offline",
			statusCode: 401,
			err:        nil,
			url:        "http://other-api.com",
			expected:   "Offline",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := determineStatus(tt.statusCode, tt.err, tt.url)
			assert.Equal(t, tt.expected, result)
		})
	}
}
