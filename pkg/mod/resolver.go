package mod

import (
	"archive/tar"
	"compress/gzip"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// Resolver handles fetching and caching dependencies
type Resolver struct {
	cacheDir string
	client   *http.Client
}

// NewResolver creates a new dependency resolver
func NewResolver() (*Resolver, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return nil, fmt.Errorf("getting home directory: %w", err)
	}

	cacheDir := filepath.Join(homeDir, ".glyph", "cache")
	if err := os.MkdirAll(cacheDir, 0755); err != nil {
		return nil, fmt.Errorf("creating cache directory: %w", err)
	}

	return &Resolver{
		cacheDir: cacheDir,
		client: &http.Client{
			Timeout: 30 * time.Second,
		},
	}, nil
}

// GitHubRelease represents a GitHub release
type GitHubRelease struct {
	TagName string `json:"tag_name"`
	Assets  []struct {
		Name string `json:"name"`
		URL  string `json:"browser_download_url"`
	} `json:"assets"`
}

// Resolve fetches a package and caches it locally
func (r *Resolver) Resolve(req Require) (string, error) {
	// Parse GitHub path: github.com/user/repo
	parts := strings.Split(req.Path, "/")
	if len(parts) < 3 {
		return "", fmt.Errorf("invalid module path: %s (expected github.com/user/repo)", req.Path)
	}

	owner := parts[1]
	repo := parts[2]

	// Check cache first
	cachePath := r.cachePath(req)
	if _, err := os.Stat(filepath.Join(cachePath, ".downloaded")); err == nil {
		return cachePath, nil
	}

	// Fetch from GitHub
	if err := r.fetchFromGitHub(owner, repo, req.Version, cachePath); err != nil {
		return "", fmt.Errorf("fetching %s: %w", req.Path, err)
	}

	return cachePath, nil
}

// cachePath returns the local cache path for a dependency
func (r *Resolver) cachePath(req Require) string {
	return filepath.Join(r.cacheDir, req.Path, req.Version.String())
}

// fetchFromGitHub downloads a package from GitHub
func (r *Resolver) fetchFromGitHub(owner, repo string, version Version, dest string) error {
	// Create destination directory
	if err := os.MkdirAll(dest, 0755); err != nil {
		return fmt.Errorf("creating cache directory: %w", err)
	}

	// GitHub API URL for release info
	apiURL := fmt.Sprintf("https://api.github.com/repos/%s/%s/releases/tags/%s", owner, repo, version)

	resp, err := r.client.Get(apiURL)
	if err != nil {
		return fmt.Errorf("fetching release info: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == 404 {
		return fmt.Errorf("release %s not found for %s/%s", version, owner, repo)
	}

	if resp.StatusCode != 200 {
		return fmt.Errorf("GitHub API error: %s", resp.Status)
	}

	var release GitHubRelease
	if err := json.NewDecoder(resp.Body).Decode(&release); err != nil {
		return fmt.Errorf("parsing release info: %w", err)
	}

	// Download source archive if available
	archiveURL := fmt.Sprintf("https://github.com/%s/%s/archive/refs/tags/%s.tar.gz", owner, repo, version)

	// Download and extract
	tmpFile := filepath.Join(dest, "source.tar.gz")
	if err := r.downloadFile(archiveURL, tmpFile); err != nil {
		return fmt.Errorf("downloading archive: %w", err)
	}
	defer os.Remove(tmpFile)

	// Extract tar.gz
	if err := r.extractTarGz(tmpFile, dest); err != nil {
		return fmt.Errorf("extracting archive: %w", err)
	}

	return nil
}

// downloadFile downloads a URL to a file
func (r *Resolver) downloadFile(url, dest string) error {
	resp, err := r.client.Get(url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return fmt.Errorf("HTTP %s", resp.Status)
	}

	out, err := os.Create(dest)
	if err != nil {
		return err
	}
	defer out.Close()

	_, err = io.Copy(out, resp.Body)
	return err
}

// extractTarGz extracts a tar.gz file
func (r *Resolver) extractTarGz(src, dest string) error {
	f, err := os.Open(src)
	if err != nil {
		return err
	}
	defer f.Close()

	gzr, err := gzip.NewReader(f)
	if err != nil {
		return err
	}
	defer gzr.Close()

	tr := tar.NewReader(gzr)

	for {
		header, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}

		// The path in the tarball is usually repo-version/file
		// We want to strip the first component
		parts := strings.Split(header.Name, "/")
		if len(parts) <= 1 {
			continue
		}
		targetPath := filepath.Join(dest, filepath.Join(parts[1:]...))

		switch header.Typeflag {
		case tar.TypeDir:
			if err := os.MkdirAll(targetPath, 0755); err != nil {
				return err
			}
		case tar.TypeReg:
			// Ensure parent directory exists
			if err := os.MkdirAll(filepath.Dir(targetPath), 0755); err != nil {
				return err
			}
			
			outFile, err := os.Create(targetPath)
			if err != nil {
				return err
			}
			if _, err := io.Copy(outFile, tr); err != nil {
				outFile.Close()
				return err
			}
			outFile.Close()
		}
	}

	// Create a placeholder file indicating the package was downloaded successfully
	placeholder := filepath.Join(dest, ".downloaded")
	return os.WriteFile(placeholder, []byte(time.Now().Format(time.RFC3339)), 0644)
}

// GetLatestVersion fetches the latest release version for a package
func (r *Resolver) GetLatestVersion(path string) (Version, error) {
	parts := strings.Split(path, "/")
	if len(parts) < 3 {
		return Version{}, fmt.Errorf("invalid module path: %s", path)
	}

	owner := parts[1]
	repo := parts[2]

	apiURL := fmt.Sprintf("https://api.github.com/repos/%s/%s/releases/latest", owner, repo)

	resp, err := r.client.Get(apiURL)
	if err != nil {
		return Version{}, fmt.Errorf("fetching latest release: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == 404 {
		return Version{}, fmt.Errorf("no releases found for %s", path)
	}

	if resp.StatusCode != 200 {
		return Version{}, fmt.Errorf("GitHub API error: %s", resp.Status)
	}

	var release GitHubRelease
	if err := json.NewDecoder(resp.Body).Decode(&release); err != nil {
		return Version{}, fmt.Errorf("parsing release info: %w", err)
	}

	return ParseVersion(release.TagName)
}

// ClearCache removes all cached packages
func (r *Resolver) ClearCache() error {
	return os.RemoveAll(r.cacheDir)
}

// CacheDir returns the cache directory path
func (r *Resolver) CacheDir() string {
	return r.cacheDir
}
