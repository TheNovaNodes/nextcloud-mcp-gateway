package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"
)

// Config holds runtime configuration parameters for Nextcloud MCP Gateway.
type Config struct {
	NCURL     string
	PublicURL string
	Username  string
	Password  string
	Timeout   time.Duration
}

// LoadFromEnv loads and validates configuration from environment variables.
func LoadFromEnv() (*Config, error) {
	ncURL := os.Getenv("NC_URL")
	if ncURL == "" {
		ncURL = "http://localhost:8080"
	}
	ncURL = strings.TrimRight(ncURL, "/")

	publicURL := os.Getenv("NC_PUBLIC_URL")
	if publicURL == "" {
		publicURL = "https://nextcloud.example.com"
	}
	publicURL = strings.TrimRight(publicURL, "/")

	username := os.Getenv("NC_USER")
	password := os.Getenv("NC_APP_PASSWORD")

	timeoutSec := 30.0
	if rawTimeout := os.Getenv("NC_TIMEOUT"); rawTimeout != "" {
		if val, err := strconv.ParseFloat(rawTimeout, 64); err == nil && val > 0 {
			timeoutSec = val
		}
	}

	return &Config{
		NCURL:     ncURL,
		PublicURL: publicURL,
		Username:  username,
		Password:  password,
		Timeout:   time.Duration(timeoutSec * float64(time.Second)),
	}, nil
}

// WebDAVURL returns the WebDAV base endpoint for user storage.
func (c *Config) WebDAVURL() string {
	if c.Username != "" {
		return fmt.Sprintf("%s/remote.php/dav/files/%s", c.NCURL, c.Username)
	}
	return fmt.Sprintf("%s/remote.php/webdav", c.NCURL)
}

// OCSURL returns the OCS Cloud API base endpoint.
func (c *Config) OCSURL() string {
	return fmt.Sprintf("%s/ocs/v1.php/cloud", c.NCURL)
}

// CalDAVURL returns the CalDAV endpoint for a given calendar.
func (c *Config) CalDAVURL(calendarName string) string {
	user := c.Username
	if user == "" {
		user = "current"
	}
	if calendarName == "" {
		calendarName = "personal"
	}
	return fmt.Sprintf("%s/remote.php/dav/calendars/%s/%s", c.NCURL, user, calendarName)
}

// DeckURL returns the Nextcloud Deck REST API v1.0 base endpoint.
func (c *Config) DeckURL() string {
	return fmt.Sprintf("%s/index.php/apps/deck/api/v1.0", c.NCURL)
}

// StatusURL returns the Nextcloud status.php endpoint.
func (c *Config) StatusURL() string {
	return fmt.Sprintf("%s/status.php", c.NCURL)
}
