package slug

import (
	"fmt"

	"github.com/gosimple/slug"
)

func init() {
	// Use Bulgarian transliteration for Cyrillic (covers Russian)
	// Max 60 chars for URL-friendly slugs
	slug.MaxLength = 60
}

// Slugify converts a title to a URL-friendly base slug.
// Supports Latin, Cyrillic (Russian/Kazakh), and other scripts via transliteration.
// Example: "Spring Hackathon 2026!" -> "spring-hackathon-2026"
// Example: "Встреча студентов" -> "vstrecha-studentov"
func Slugify(title string) string {
	return slug.MakeLang(title, "bg")
}

// WithSuffix appends a numeric suffix to a base slug.
// Example: WithSuffix("tech-workshop", 2) -> "tech-workshop-2"
func WithSuffix(base string, n int) string {
	if n <= 1 {
		return base
	}
	return fmt.Sprintf("%s-%d", base, n)
}
