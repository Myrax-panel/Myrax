//go:build linux

package actions

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"time"

	"myrax/internal/version"
)

func Reboot() error {
	return exec.Command("systemctl", "reboot").Start()
}

func Shutdown() error {
	return exec.Command("systemctl", "poweroff").Start()
}

func Reload() error {
	return exec.Command("sh", "-c", "sleep 0.2; systemctl restart myrax.service").Start()
}

func restartService() error {
	if output, err := exec.Command("systemctl", "restart", "myrax.service").CombinedOutput(); err != nil {
		return fmt.Errorf("restart myrax.service failed: %w: %s", err, string(output))
	}
	return nil
}

func UpdateLatest() error {
	arch := runtime.GOARCH
	switch arch {
	case "amd64", "arm64":
	default:
		return fmt.Errorf("unsupported architecture: %s", arch)
	}

	executable, err := os.Executable()
	if err != nil || executable == "" {
		executable = "/usr/local/bin/myrax"
	}
	executable, err = filepath.EvalSymlinks(executable)
	if err != nil {
		executable = "/usr/local/bin/myrax"
	}

	temp, err := os.CreateTemp("", "myrax-update-*")
	if err != nil {
		return err
	}
	tempPath := temp.Name()
	defer os.Remove(tempPath)

	url := version.GitHubLatestDownloadURL(arch)
	client := http.Client{Timeout: 90 * time.Second}
	response, err := client.Get(url)
	if err != nil {
		temp.Close()
		return err
	}
	defer response.Body.Close()
	if response.StatusCode < 200 || response.StatusCode >= 300 {
		temp.Close()
		return fmt.Errorf("download failed: %s", response.Status)
	}
	if _, err := io.Copy(temp, response.Body); err != nil {
		temp.Close()
		return err
	}
	if err := temp.Close(); err != nil {
		return err
	}
	if err := os.Chmod(tempPath, 0755); err != nil {
		return err
	}

	nextPath := executable + ".new"
	if err := copyFile(tempPath, nextPath, 0755); err != nil {
		return err
	}
	if err := os.Rename(nextPath, executable); err != nil {
		_ = os.Remove(nextPath)
		return err
	}
	return restartService()
}

func UpdateLatestDetached() error {
	executable, err := os.Executable()
	if err != nil || executable == "" {
		executable = "/usr/local/bin/myrax"
	}
	executable, err = filepath.EvalSymlinks(executable)
	if err != nil {
		executable = "/usr/local/bin/myrax"
	}
	unit := fmt.Sprintf("myrax-update-%d", time.Now().Unix())
	command := exec.Command(
		"systemd-run",
		"--quiet",
		"--collect",
		"--unit", unit,
		"--property", "Type=oneshot",
		"--property", "User=root",
		"--property", "Group=root",
		executable,
		"update",
	)
	if output, err := command.CombinedOutput(); err != nil {
		return fmt.Errorf("start update job failed: %w: %s", err, string(output))
	}
	return nil
}

func copyFile(source string, target string, mode os.FileMode) error {
	in, err := os.Open(source)
	if err != nil {
		return err
	}
	defer in.Close()
	out, err := os.OpenFile(target, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, mode)
	if err != nil {
		return err
	}
	defer out.Close()
	_, err = io.Copy(out, in)
	return err
}
