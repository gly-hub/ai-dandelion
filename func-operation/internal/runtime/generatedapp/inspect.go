package generatedapp

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"time"
)

type AppInspection struct {
	Exists          bool
	ManifestVersion string
	Name            string
	Description     string
	Actions         []string
	UpdatedAtMicro  int64
	FrontendEntry   string
	BackendModule   string
}

func (s *Service) InspectApp(appID string) (AppInspection, error) {
	appID = strings.TrimSpace(appID)
	if appID == "" {
		return AppInspection{}, errors.New("app id is required")
	}
	dir := filepath.Join(s.rootDir, appID)
	manifestPath := filepath.Join(dir, "manifest.json")
	raw, err := os.ReadFile(manifestPath)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return AppInspection{Exists: false}, nil
		}
		return AppInspection{}, err
	}
	item, _, _, err := decodeManifestWithDir(dir, raw)
	if err != nil {
		return AppInspection{}, err
	}
	updatedAt := fileLatestModTimeMicro(
		manifestPath,
		filepath.Join(dir, "frontend.js"),
		filepath.Join(dir, firstNonEmpty(strings.TrimSpace(item.BackendModule), "backend.wasm")),
	)
	return AppInspection{
		Exists:          true,
		ManifestVersion: strings.TrimSpace(item.Version),
		Name:            firstNonEmpty(strings.TrimSpace(item.Name), appID),
		Description:     strings.TrimSpace(item.Description),
		Actions:         append([]string(nil), item.Actions...),
		UpdatedAtMicro:  updatedAt,
		FrontendEntry:   fmtFrontendEntry(appID),
		BackendModule:   firstNonEmpty(strings.TrimSpace(item.BackendModule), "backend.wasm"),
	}, nil
}

func fmtFrontendEntry(appID string) string {
	return "/func-operation/generated-apps/" + appID + "/frontend.js"
}

func fileLatestModTimeMicro(paths ...string) int64 {
	var latest int64
	for _, path := range paths {
		info, err := os.Stat(path)
		if err != nil {
			continue
		}
		mod := info.ModTime().UnixMicro()
		if mod > latest {
			latest = mod
		}
	}
	if latest == 0 {
		return time.Now().UnixMicro()
	}
	return latest
}
