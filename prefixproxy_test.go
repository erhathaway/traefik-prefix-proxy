package prefixproxy

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewPrefixProxy(t *testing.T) {
	testCases := []struct {
		desc         string
		prefix       string
		expectsError bool
	}{
		{
			desc:   "Works with a non empty prefix",
			prefix: "api",
		},
		{
			desc:         "Fails if prefix is empty",
			prefix:       "",
			expectsError: true,
		},
	}

	for _, test := range testCases {
		t.Run(test.desc, func(t *testing.T) {
			t.Parallel()

			next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {})
			config := &Config{Prefix: test.prefix}

			_, err := New(context.Background(), next, config, "foo-prefix-proxy")
			if test.expectsError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestPrefixProxy(t *testing.T) {
	testCases := []struct {
		desc             string
		prefix           string
		path             string
		expectedPath     string
		expectedRawPath  string
		location         string
		expectedLocation string
	}{
		{
			desc:             "Remove prefix",
			prefix:           "api",
			path:             "/api/users",
			expectedPath:     "/users",
			location:         "/users/123",
			expectedLocation: "/api/users/123",
		},
		{
			desc:             "Remove prefix with trailing slash",
			prefix:           "api",
			path:             "/api/users/",
			expectedPath:     "/users/",
			location:         "/users/123",
			expectedLocation: "/api/users/123",
		},
		{
			desc:             "No prefix to remove",
			prefix:           "api",
			path:             "/users",
			expectedPath:     "/users",
			location:         "/users/123",
			expectedLocation: "/users/123",
		},
		{
			desc:             "Root path",
			prefix:           "api",
			path:             "/api",
			expectedPath:     "/",
			location:         "/",
			expectedLocation: "/api",
		},
		{
			desc:             "Works with a raw path",
			prefix:           "api",
			path:             "/api/users%2Fprofiles",
			expectedPath:     "/users/profiles",
			expectedRawPath:  "/users%2Fprofiles",
			location:         "/users/profiles/123",
			expectedLocation: "/api/users/profiles/123",
		},
	}

	for _, test := range testCases {
		t.Run(test.desc, func(t *testing.T) {
			t.Parallel()

			var actualPath, actualRawPath string

			next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				actualPath = r.URL.Path
				actualRawPath = r.URL.RawPath
				w.Header().Set("Location", test.location)
				w.WriteHeader(http.StatusFound) // Add this line to ensure WriteHeader is called
			})

			req, err := http.NewRequest(http.MethodGet, "http://localhost"+test.path, nil)
			require.NoError(t, err)

			config := &Config{Prefix: test.prefix}
			handler, err := New(context.Background(), next, config, "foo-prefix-proxy")
			require.NoError(t, err)

			recorder := httptest.NewRecorder()
			handler.ServeHTTP(recorder, req)

			assert.Equal(t, test.expectedPath, actualPath)
			if test.expectedRawPath != "" {
				assert.Equal(t, test.expectedRawPath, actualRawPath)
			}
			assert.Equal(t, test.expectedLocation, recorder.Header().Get("Location"))
		})
	}
}
