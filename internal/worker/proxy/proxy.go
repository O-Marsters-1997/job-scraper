package proxy

import (
	"fmt"
	"net/url"
	"os"
	"strconv"
)

const (
	residentialEnvKey = "DECODO_PROXY_URL"
)

// ValidateResidential checks DECODO_PROXY_URL, which every residential Source needs.
func ValidateResidential() error {
	_, err := proxyURL(residentialEnvKey)
	return err
}

func proxyURL(envKey string) (*url.URL, error) {
	raw := os.Getenv(envKey)
	if raw == "" {
		return nil, fmt.Errorf("proxy: %s is required", envKey)
	}
	u, err := url.Parse(raw)
	if err != nil || u.Scheme != "http" || u.Hostname() == "" || u.User == nil {
		return nil, fmt.Errorf("proxy: invalid %s", envKey)
	}
	port, err := strconv.Atoi(u.Port())
	if err != nil || port < 1 || port > 65535 || u.Path != "" || u.RawQuery != "" || u.Fragment != "" {
		return nil, fmt.Errorf("proxy: invalid %s", envKey)
	}
	if password, ok := u.User.Password(); !ok || password == "" || u.User.Username() == "" {
		return nil, fmt.Errorf("proxy: %s requires username and password", envKey)
	}
	return u, nil
}
