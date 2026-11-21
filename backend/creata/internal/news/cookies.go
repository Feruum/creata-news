package news

import (
	"errors"
	"golang.org/x/net/publicsuffix"
	"net"
	"net/url"
	"strings"
)

func validateCookieDomain(c *Config) error {
	c.CookieDomain = strings.TrimPrefix(strings.ToLower(strings.TrimSpace(c.CookieDomain)), ".")
	hosts := []string{}
	for _, raw := range append([]string{c.APIURL, c.PublicURL}, c.Origins...) {
		if strings.TrimSpace(raw) == "" {
			continue
		}
		u, e := url.Parse(strings.TrimSpace(raw))
		if e != nil || u.Hostname() == "" || u.User != nil {
			return errors.New("API URL, public URL and origins must be valid absolute URLs")
		}
		hosts = append(hosts, strings.ToLower(u.Hostname()))
	}
	if c.CookieDomain != "" {
		suffix, _ := publicsuffix.PublicSuffix(c.CookieDomain)
		if suffix == c.CookieDomain || net.ParseIP(c.CookieDomain) != nil || strings.ContainsAny(c.CookieDomain, "/:\\ ") || !strings.Contains(c.CookieDomain, ".") {
			return errors.New("cookie domain must be a registrable shared parent domain")
		}
		for _, host := range hosts {
			if host != c.CookieDomain && !strings.HasSuffix(host, "."+c.CookieDomain) {
				return errors.New("cookie domain must contain the API and every configured frontend hostname")
			}
		}
	}
	oidc := c.GoogleClientID != "" || c.TelegramClientID != ""
	if c.Environment == "production" && oidc {
		api, e := url.Parse(c.APIURL)
		if e != nil || api.Scheme != "https" || api.Hostname() == "" {
			return errors.New("production OAuth requires an HTTPS API URL")
		}
		for _, host := range hosts {
			if host != strings.ToLower(api.Hostname()) && c.CookieDomain == "" {
				return errors.New("cross-host production OAuth requires CREATA_COOKIE_DOMAIN")
			}
		}
	}
	return nil
}
