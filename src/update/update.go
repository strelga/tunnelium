package update

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"runtime"
	"strings"
)

const (
	owner = "strelga"
	repo  = "tunnelium"
)

// githubRelease is a minimal representation of a GitHub release response.
type githubRelease struct {
	TagName string `json:"tag_name"`
	Assets  []struct {
		Name               string `json:"name"`
		BrowserDownloadURL string `json:"browser_download_url"`
	} `json:"assets"`
}

// Run performs a self-update: checks the latest GitHub release, compares
// versions, downloads the matching binary, and replaces the current executable.
func Run(currentVersion string) error {
	// 1. Get current executable path
	exePath, err := os.Executable()
	if err != nil {
		return fmt.Errorf("cannot determine executable path: %w", err)
	}
	exePath, err = resolveSymlink(exePath)
	if err != nil {
		return fmt.Errorf("cannot resolve symlink: %w", err)
	}

	// 2. Fetch latest release from GitHub
	fmt.Println("Checking for latest release on GitHub...")
	release, err := fetchLatestRelease()
	if err != nil {
		return err
	}

	latestVersion := strings.TrimPrefix(release.TagName, "v")
	currentClean := strings.TrimPrefix(currentVersion, "v")

	if latestVersion == currentClean {
		fmt.Printf("Already on the latest version: v%s\n", currentClean)
		return nil
	}

	fmt.Printf("New version available: v%s (current: v%s)\n", latestVersion, currentClean)

	// 3. Determine target asset name
	assetName := fmt.Sprintf("tunnelium-%s-%s", runtime.GOOS, runtime.GOARCH)

	// 4. Find download URL
	downloadURL := ""
	for _, a := range release.Assets {
		if a.Name == assetName {
			downloadURL = a.BrowserDownloadURL
			break
		}
	}
	if downloadURL == "" {
		return fmt.Errorf("binary %q not found in release v%s", assetName, latestVersion)
	}

	// 5. Download to a temporary file next to the current executable
	fmt.Printf("Downloading %s ...\n", assetName)
	tmpFile := exePath + ".tmp"
	if err := downloadFile(downloadURL, tmpFile); err != nil {
		os.Remove(tmpFile)
		return fmt.Errorf("download failed: %w", err)
	}

	// 6. Preserve file permissions
	info, err := os.Stat(exePath)
	if err != nil {
		os.Remove(tmpFile)
		return fmt.Errorf("cannot stat current binary: %w", err)
	}
	if err := os.Chmod(tmpFile, info.Mode()); err != nil {
		os.Remove(tmpFile)
		return fmt.Errorf("cannot set permissions: %w", err)
	}

	// 7. Atomic replace
	if err := os.Rename(tmpFile, exePath); err != nil {
		os.Remove(tmpFile)
		return fmt.Errorf("cannot replace binary: %w", err)
	}

	fmt.Printf("Updated to v%s\n", latestVersion)
	return nil
}

// fetchLatestRelease queries the GitHub API for the latest release.
func fetchLatestRelease() (*githubRelease, error) {
	url := fmt.Sprintf("https://api.github.com/repos/%s/%s/releases/latest", owner, repo)
	resp, err := http.Get(url)
	if err != nil {
		return nil, fmt.Errorf("GitHub API request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("GitHub API returned %d: %s", resp.StatusCode, string(body))
	}

	var release githubRelease
	if err := json.NewDecoder(resp.Body).Decode(&release); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}
	return &release, nil
}

// downloadFile downloads a URL to a local file path.
func downloadFile(url, dest string) error {
	resp, err := http.Get(url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("HTTP %d", resp.StatusCode)
	}

	f, err := os.Create(dest)
	if err != nil {
		return err
	}
	defer f.Close()

	_, err = io.Copy(f, resp.Body)
	return err
}

// resolveSymlink resolves a symlink to the real path (needed on macOS with Homebrew etc.)
func resolveSymlink(p string) (string, error) {
	fi, err := os.Lstat(p)
	if err != nil {
		return p, err
	}
	if fi.Mode()&os.ModeSymlink != 0 {
		resolved, err := os.Readlink(p)
		if err != nil {
			return p, err
		}
		return resolved, nil
	}
	return p, nil
}
