package services_test

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"sort"
	"strings"
	"testing"

	"github.com/studyverse/ems-backend/internal/roomid"
)

var (
	mapRoomIDPattern  = regexp.MustCompile(`'([A-Za-z0-9_.]+)'`)
	seedRoomIDPattern = regexp.MustCompile(`\('([^']+)'\s*,`)
)

func TestMapRoomCatalogHasProductionMigrationCoverage(t *testing.T) {
	mapRoomIDs, err := readMapRoomIDs()
	if err != nil {
		t.Fatalf("read map room ids: %v", err)
	}

	migratedRoomIDs, err := readMigratedRoomIDs()
	if err != nil {
		t.Fatalf("read migrated room ids: %v", err)
	}

	missing := make([]string, 0)
	for roomID := range mapRoomIDs {
		if _, exists := migratedRoomIDs[roomID]; !exists {
			missing = append(missing, roomID)
		}
	}

	if len(missing) > 0 {
		sort.Strings(missing)
		t.Fatalf("map catalog has %d ids missing in production migrations: %s", len(missing), strings.Join(missing, ", "))
	}
}

func readMapRoomIDs() (map[string]struct{}, error) {
	mapCatalogPath := projectPath("..", "map", "src", "features", "events", "components", "map-room-catalog.ts")
	content, err := os.ReadFile(mapCatalogPath)
	if err != nil {
		return nil, err
	}

	const marker = "] as const"
	rawCatalog := string(content)
	markerIndex := strings.Index(rawCatalog, marker)
	if markerIndex == -1 {
		return nil, fmt.Errorf("marker %q not found in map room catalog", marker)
	}

	matches := mapRoomIDPattern.FindAllStringSubmatch(rawCatalog[:markerIndex], -1)
	roomIDs := make(map[string]struct{}, len(matches))
	for _, match := range matches {
		if len(match) < 2 {
			continue
		}

		normalizedRoomID := roomid.Normalize(match[1])
		if normalizedRoomID == "" {
			continue
		}

		roomIDs[normalizedRoomID] = struct{}{}
	}

	return roomIDs, nil
}

func readMigratedRoomIDs() (map[string]struct{}, error) {
	migrationsPath := projectPath("..", "..", "packages", "db", "migrations")
	content, err := os.ReadFile(filepath.Join(migrationsPath, "meta", "_journal.json"))
	if err != nil {
		return nil, err
	}

	var journal struct {
		Entries []struct {
			Tag string `json:"tag"`
		} `json:"entries"`
	}
	if err := json.Unmarshal(content, &journal); err != nil {
		return nil, fmt.Errorf("decode migration journal: %w", err)
	}

	roomIDs := make(map[string]struct{})
	for _, entry := range journal.Entries {
		content, err := os.ReadFile(filepath.Join(migrationsPath, entry.Tag+".sql"))
		if err != nil {
			return nil, fmt.Errorf("read migration %s: %w", entry.Tag, err)
		}
		for _, match := range seedRoomIDPattern.FindAllStringSubmatch(string(content), -1) {
			normalizedRoomID := roomid.Normalize(match[1])
			if normalizedRoomID != "" {
				roomIDs[normalizedRoomID] = struct{}{}
			}
		}
	}

	return roomIDs, nil
}

func projectPath(parts ...string) string {
	_, thisFilePath, _, _ := runtime.Caller(0)
	basePath := filepath.Join(filepath.Dir(thisFilePath), "..", "..")
	pathParts := append([]string{basePath}, parts...)
	return filepath.Clean(filepath.Join(pathParts...))
}
