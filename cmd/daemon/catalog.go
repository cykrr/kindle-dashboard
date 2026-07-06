package main

import (
	"encoding/json"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"sync"
)

// CatalogItem is one launchable shortcut, exposed to clients.
type CatalogItem struct {
	ID       string `json:"id"`
	Name     string `json:"name"`
	Category string `json:"category"`
	IconURL  string `json:"icon_url"`
}

var (
	catalogMu   sync.RWMutex
	catalogByID = map[string]string{} // id -> shortcut full path
)

var idSanitize = regexp.MustCompile(`[^a-z0-9]+`)

// makeID turns a shortcut display name into a URL-safe stable id.
func makeID(name string) string {
	return strings.Trim(idSanitize.ReplaceAllString(strings.ToLower(name), "-"), "-")
}

func scanCatalog() []CatalogItem {
	items := []CatalogItem{}
	byID := map[string]string{}

	err := filepath.WalkDir(cfg.ShortcutsDir, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return nil // skip errors
		}
		if d.IsDir() {
			return nil
		}
		
		ext := strings.ToLower(filepath.Ext(d.Name()))
		if ext != ".lnk" && ext != ".url" && ext != ".ps1" && ext != ".bat" && ext != ".cmd" && ext != ".exe" {
			return nil
		}
		
		name := strings.TrimSuffix(d.Name(), filepath.Ext(d.Name()))
		id := makeID(name)
		if id == "" {
			return nil
		}
		
		// Determine category from relative path
		category := "root"
		rel, err := filepath.Rel(cfg.ShortcutsDir, filepath.Dir(path))
		if err == nil && rel != "." && rel != "" {
			// Use the first folder name as category, or the whole relative path
			category = strings.ReplaceAll(rel, string(filepath.Separator), " / ")
		}

		if _, exists := byID[id]; !exists {
			byID[id] = path
			items = append(items, CatalogItem{
				ID:       id,
				Name:     name,
				Category: category,
				IconURL:  "/icon/" + id + ".png",
			})
		}
		return nil
	})

	if err != nil {
		log.Printf("catalog: error scanning %s: %v", cfg.ShortcutsDir, err)
	}

	catalogMu.Lock()
	catalogByID = byID
	catalogMu.Unlock()

	sort.Slice(items, func(i, j int) bool {
		if items[i].Category != items[j].Category {
			return items[i].Category < items[j].Category
		}
		return items[i].Name < items[j].Name
	})
	return items
}

// pathForID returns the shortcut path for an id, rescanning once if unknown
// (so a freshly-dropped shortcut resolves without a restart).
func pathForID(id string) string {
	catalogMu.RLock()
	src := catalogByID[id]
	catalogMu.RUnlock()
	if src == "" {
		scanCatalog()
		catalogMu.RLock()
		src = catalogByID[id]
		catalogMu.RUnlock()
	}
	return src
}

func executeLaunch(id string) error {
	if id == "" {
		return unknownActionError("launch: missing target")
	}
	src := pathForID(id)
	if src == "" {
		return unknownActionError("launch: unknown target " + id)
	}
	
	ext := strings.ToLower(filepath.Ext(src))
	if ext == ".ps1" {
		return runPowerShell(false, "& '"+psEscape(src)+"'")
	} else if ext == ".bat" || ext == ".cmd" {
		return runPowerShell(false, "Start-Process -FilePath '"+psEscape(src)+"' -WindowStyle Hidden")
	}
	return runPowerShell(false, "Invoke-Item -LiteralPath '"+psEscape(src)+"'")
}

// handleCatalog serves GET /catalog as JSON (rescans on each call).
func handleCatalog(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Content-Type", "application/json")
	if r.URL.Query().Get("key") != cfg.APIKey {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}
	data, _ := json.Marshal(scanCatalog())
	w.Write(data)
}

// handleIcon serves GET /icon/<id>.png, extracting on demand and caching to
// disk. The cache is refreshed when the source shortcut is newer.
func handleIcon(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	if r.URL.Query().Get("key") != cfg.APIKey {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	id := strings.TrimSuffix(strings.TrimPrefix(r.URL.Path, "/icon/"), ".png")
	src := pathForID(id)
	if src == "" {
		http.NotFound(w, r)
		return
	}

	cachePath := filepath.Join(cfg.IconCacheDir, id+".png")
	if iconCacheStale(cachePath, src) {
		if err := os.MkdirAll(cfg.IconCacheDir, 0755); err != nil {
			http.Error(w, "cache dir: "+err.Error(), http.StatusInternalServerError)
			return
		}
		if err := extractIconPNG(src, cachePath); err != nil {
			log.Printf("icon extract %s: %v", id, err)
			http.Error(w, "extract failed", http.StatusInternalServerError)
			return
		}
	}
	w.Header().Set("Content-Type", "image/png")
	http.ServeFile(w, r, cachePath)
}

// iconCacheStale reports whether the cached PNG is missing or older than src.
func iconCacheStale(cachePath, src string) bool {
	ci, err := os.Stat(cachePath)
	if err != nil {
		return true
	}
	si, err := os.Stat(src)
	if err != nil {
		return false // can't compare; keep cache
	}
	return si.ModTime().After(ci.ModTime())
}
