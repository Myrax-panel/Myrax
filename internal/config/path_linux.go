//go:build linux

package config

func defaultConfigPath() string {
	return "/etc/myrax/config.toml"
}
