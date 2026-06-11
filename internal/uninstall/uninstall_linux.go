//go:build linux

package uninstall

import (
	"errors"
	"os"
	"os/exec"
)

func Run(purge bool) error {
	_ = exec.Command("systemctl", "disable", "--now", "myrax.service").Run()
	_ = os.Remove("/etc/systemd/system/myrax.service")
	_ = os.Remove("/usr/local/bin/myrax")
	_ = os.Remove("/var/service/myrax")
	_ = os.RemoveAll("/etc/sv/myrax")

	if purge {
		_ = os.RemoveAll("/etc/myrax")
		_ = os.RemoveAll("/var/lib/myrax")
	}

	if err := exec.Command("systemctl", "daemon-reload").Run(); err != nil && !errors.Is(err, os.ErrNotExist) {
		return nil
	}
	return nil
}
