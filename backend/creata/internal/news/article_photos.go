package news

import (
	"fmt"
	"net/url"
	"path"
	"strings"
	"unicode/utf8"
)

var articlePhotoLicenses = map[string]string{
	"/licenses/by/3.0":       "CC BY 3.0",
	"/licenses/by/4.0":       "CC BY 4.0",
	"/licenses/by-sa/3.0":    "CC BY-SA 3.0",
	"/licenses/by-sa/4.0":    "CC BY-SA 4.0",
	"/publicdomain/zero/1.0": "CC0 1.0",
}

func articlePhotoURL(raw string) *url.URL {
	if len(raw) > 2048 || strings.TrimSpace(raw) != raw || strings.ContainsRune(raw, '\\') {
		return nil
	}
	parsed, err := parsePublicURL(raw)
	if err != nil || parsed.Scheme != "https" || parsed.Port() != "" || parsed.RawQuery != "" || parsed.ForceQuery || parsed.Fragment != "" {
		return nil
	}
	return parsed
}

func validateArticlePhotos(value any) error {
	photos, ok := value.([]any)
	if !ok || len(photos) > 3 {
		return fmt.Errorf("photos must be an array of up to three images")
	}
	seen := make(map[string]bool, len(photos))
	for index, value := range photos {
		photo := object(value)
		image := articlePhotoURL(str(photo["url"]))
		source := articlePhotoURL(str(photo["sourceUrl"]))
		license := articlePhotoURL(str(photo["licenseUrl"]))
		if len(photo) != 6 || image == nil || source == nil || license == nil {
			return fmt.Errorf("photos[%d] must include two Wikimedia links, a license, a caption and a credit", index)
		}
		if !has([]string{"upload.wikimedia.org", "thumb.wikimedia.org"}, image.Hostname()) || !strings.HasPrefix(image.Path, "/wikipedia/commons/") {
			return fmt.Errorf("photos[%d] must use a Wikimedia Commons image URL", index)
		}
		if source.Hostname() != "commons.wikimedia.org" || !strings.HasPrefix(source.Path, "/wiki/File:") || source.Path == "/wiki/File:" {
			return fmt.Errorf("photos[%d] must link to the Wikimedia Commons file page", index)
		}
		imageName := path.Base(image.Path)
		if strings.HasPrefix(image.Path, "/wikipedia/commons/thumb/") {
			imageName = path.Base(path.Dir(image.Path))
		}
		key := strings.ReplaceAll(strings.TrimPrefix(source.Path, "/wiki/File:"), "_", " ")
		if strings.ReplaceAll(imageName, "_", " ") != key {
			return fmt.Errorf("photos[%d] must credit the same file as the image", index)
		}
		rights := articlePhotoLicenses[strings.TrimSuffix(license.Path, "/")]
		if license.Hostname() != "creativecommons.org" || rights == "" || rights != str(photo["rights"]) {
			return fmt.Errorf("photos[%d] must have a matching Creative Commons license", index)
		}
		if !nonempty(photo["caption"]) || !nonempty(photo["credit"]) || utf8.RuneCountInString(str(photo["caption"])) > 500 || utf8.RuneCountInString(str(photo["credit"])) > 500 {
			return fmt.Errorf("photos[%d] must have a caption and a credit of up to 500 characters", index)
		}
		if seen[key] {
			return fmt.Errorf("photos must contain distinct Wikimedia Commons files")
		}
		seen[key] = true
	}
	return nil
}

func copyArticlePhotos(target, source Doc) {
	if photos, ok := source["photos"]; ok && validateArticlePhotos(photos) == nil {
		target["photos"] = photos
	}
}
