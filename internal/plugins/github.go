package plugins

import (
	"archive/zip"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"strings"
	"time"
)

type githubSource struct {
	Owner   string
	Repo    string
	Branch  string
	SubPath string
}

func looksLikeGitHub(source string) bool {
	return strings.Contains(strings.ToLower(source), "github.com/")
}

func downloadGitHub(source string, tempDir string) (string, error) {
	parsed, err := parseGitHubSource(source)
	if err != nil {
		return "", err
	}
	branches := []string{parsed.Branch}
	if parsed.Branch == "" {
		branches = []string{"main", "master"}
	}

	var lastErr error
	for _, branch := range branches {
		archivePath := filepath.Join(tempDir, fmt.Sprintf("%s-%s.zip", parsed.Repo, branch))
		if err := downloadFile(githubZipURL(parsed.Owner, parsed.Repo, branch), archivePath); err != nil {
			lastErr = err
			continue
		}
		extractDir := filepath.Join(tempDir, fmt.Sprintf("extract-%s", branch))
		if err := unzip(archivePath, extractDir); err != nil {
			lastErr = err
			continue
		}
		root, err := singleChildDir(extractDir)
		if err != nil {
			lastErr = err
			continue
		}
		pluginDir := filepath.Join(root, filepath.FromSlash(parsed.SubPath))
		if parsed.SubPath == "" {
			pluginDir = root
		}
		if exists(pluginDir) {
			return pluginDir, nil
		}
		lastErr = fmt.Errorf("plugin subpath not found: %s", parsed.SubPath)
	}
	return "", lastErr
}

func parseGitHubSource(source string) (githubSource, error) {
	if !strings.Contains(source, "://") {
		source = "https://" + source
	}
	u, err := url.Parse(source)
	if err != nil {
		return githubSource{}, err
	}
	if strings.ToLower(u.Host) != "github.com" {
		return githubSource{}, fmt.Errorf("unsupported GitHub host: %s", u.Host)
	}
	segments := strings.Split(strings.Trim(u.Path, "/"), "/")
	if len(segments) < 2 {
		return githubSource{}, fmt.Errorf("GitHub URL must include owner and repo")
	}
	result := githubSource{
		Owner: segments[0],
		Repo:  strings.TrimSuffix(segments[1], ".git"),
	}
	rest := segments[2:]
	if len(rest) >= 2 && rest[0] == "tree" {
		result.Branch = rest[1]
		rest = rest[2:]
	}
	result.SubPath = path.Join(rest...)
	return result, nil
}

func githubZipURL(owner string, repo string, branch string) string {
	return fmt.Sprintf("https://codeload.github.com/%s/%s/zip/refs/heads/%s", owner, repo, branch)
}

func downloadFile(source string, target string) error {
	client := http.Client{Timeout: 45 * time.Second}
	response, err := client.Get(source)
	if err != nil {
		return err
	}
	defer response.Body.Close()
	if response.StatusCode < 200 || response.StatusCode >= 300 {
		return fmt.Errorf("download failed: %s", response.Status)
	}
	out, err := os.Create(target)
	if err != nil {
		return err
	}
	defer out.Close()
	_, err = io.Copy(out, response.Body)
	return err
}

func unzip(source string, target string) error {
	reader, err := zip.OpenReader(source)
	if err != nil {
		return err
	}
	defer reader.Close()

	for _, file := range reader.File {
		dest := filepath.Join(target, file.Name)
		if !strings.HasPrefix(dest, filepath.Clean(target)+string(os.PathSeparator)) {
			return fmt.Errorf("zip path escapes target: %s", file.Name)
		}
		if file.FileInfo().IsDir() {
			if err := os.MkdirAll(dest, 0750); err != nil {
				return err
			}
			continue
		}
		if err := os.MkdirAll(filepath.Dir(dest), 0750); err != nil {
			return err
		}
		in, err := file.Open()
		if err != nil {
			return err
		}
		out, err := os.OpenFile(dest, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, file.Mode())
		if err != nil {
			in.Close()
			return err
		}
		_, copyErr := io.Copy(out, in)
		closeErr := out.Close()
		in.Close()
		if copyErr != nil {
			return copyErr
		}
		if closeErr != nil {
			return closeErr
		}
	}
	return nil
}

func singleChildDir(root string) (string, error) {
	entries, err := os.ReadDir(root)
	if err != nil {
		return "", err
	}
	for _, entry := range entries {
		if entry.IsDir() {
			return filepath.Join(root, entry.Name()), nil
		}
	}
	return "", fmt.Errorf("archive root not found")
}
