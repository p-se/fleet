package bundlereader

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"encoding/pem"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"sync"
	"testing"

	ggcrregistry "github.com/google/go-containerregistry/pkg/registry"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"helm.sh/helm/v4/pkg/registry"

	"github.com/rancher/fleet/internal/httputils"
)

const (
	ociTestUsername = "user"
	ociTestPassword = "pass"
)

// ociTestChart returns a minimal packaged chart named foo, version 1.0.0.
func ociTestChart(t *testing.T) []byte {
	t.Helper()

	chartYAML := []byte("apiVersion: v2\nname: foo\nversion: 1.0.0\n")

	var buf bytes.Buffer
	gz := gzip.NewWriter(&buf)
	tw := tar.NewWriter(gz)
	require.NoError(t, tw.WriteHeader(&tar.Header{Name: "foo/Chart.yaml", Mode: 0o644, Size: int64(len(chartYAML))}))
	_, err := tw.Write(chartYAML)
	require.NoError(t, err)
	require.NoError(t, tw.Close())
	require.NoError(t, gz.Close())

	return buf.Bytes()
}

func TestDownloadOCIChartSendsFleetUserAgent(t *testing.T) {
	caBundle := func(srv *httptest.Server) []byte {
		return pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: srv.Certificate().Raw})
	}

	tests := []struct {
		name         string
		tls          bool
		requiresAuth bool
		auth         func(srv *httptest.Server) Auth
	}{
		{
			name: "plain HTTP without credentials",
			auth: func(*httptest.Server) Auth { return Auth{BasicHTTP: true} },
		},
		{
			name:         "plain HTTP with credentials",
			requiresAuth: true,
			auth: func(*httptest.Server) Auth {
				return Auth{BasicHTTP: true, Username: ociTestUsername, Password: ociTestPassword}
			},
		},
		{
			name:         "TLS skipping verification with credentials",
			tls:          true,
			requiresAuth: true,
			auth: func(*httptest.Server) Auth {
				return Auth{InsecureSkipVerify: true, Username: ociTestUsername, Password: ociTestPassword}
			},
		},
		{
			name: "TLS with CA bundle without credentials",
			tls:  true,
			auth: func(srv *httptest.Server) Auth { return Auth{CABundle: caBundle(srv)} },
		},
		{
			name:         "TLS with CA bundle and credentials",
			tls:          true,
			requiresAuth: true,
			auth: func(srv *httptest.Server) Auth {
				return Auth{CABundle: caBundle(srv), Username: ociTestUsername, Password: ociTestPassword}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var (
				mu         sync.Mutex
				recording  bool
				userAgents []string
			)

			reg := ggcrregistry.New()
			handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				mu.Lock()
				if recording {
					userAgents = append(userAgents, strings.Join(r.Header.Values("User-Agent"), ","))
				}
				mu.Unlock()

				if tt.requiresAuth {
					username, password, ok := r.BasicAuth()
					if !ok || username != ociTestUsername || password != ociTestPassword {
						w.Header().Set("WWW-Authenticate", `Basic realm="test"`)
						w.WriteHeader(http.StatusUnauthorized)
						return
					}
				}
				reg.ServeHTTP(w, r)
			})

			var srv *httptest.Server
			if tt.tls {
				srv = httptest.NewTLSServer(handler)
			} else {
				srv = httptest.NewServer(handler)
			}
			defer srv.Close()

			host := strings.TrimPrefix(strings.TrimPrefix(srv.URL, "https://"), "http://")

			// Push the chart with a client of its own; only the requests of
			// the download are of interest.
			pushOptions := []registry.ClientOption{
				registry.ClientOptHTTPClient(srv.Client()),
				registry.ClientOptCredentialsFile(filepath.Join(t.TempDir(), "creds.json")),
				registry.ClientOptBasicAuth(ociTestUsername, ociTestPassword),
			}
			if !tt.tls {
				pushOptions = append(pushOptions, registry.ClientOptPlainHTTP())
			}
			pushClient, err := registry.NewClient(pushOptions...)
			require.NoError(t, err)
			_, err = pushClient.Push(ociTestChart(t), host+"/charts/foo:1.0.0")
			require.NoError(t, err)

			mu.Lock()
			recording = true
			mu.Unlock()

			saved, err := downloadOCIChart("oci://"+host+"/charts/foo", "1.0.0", t.TempDir(), tt.auth(srv))
			require.NoError(t, err)
			assert.FileExists(t, saved)

			mu.Lock()
			defer mu.Unlock()
			require.NotEmpty(t, userAgents)
			for _, userAgent := range userAgents {
				assert.Equal(t, httputils.UserAgent(), userAgent)
			}
		})
	}
}
