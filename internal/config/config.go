package config

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
)

const fileMode = 0o600

type Settings struct {
	ThemeName   string `json:"theme_name"`
	ShowSidebar bool   `json:"show_sidebar"`
	DefaultFeed string `json:"default_feed"`
	SortMode    string `json:"sort_mode"`
	HideRead    bool   `json:"hide_read"`
}

func Defaults() Settings {
	return Settings{ThemeName: "Phosphor", ShowSidebar: true, DefaultFeed: "top"}
}

type Store struct {
	path string
}

func NewStore(path string) Store { return Store{path: path} }

func DefaultPath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".hackernews", "config.json"), nil
}

func (s Store) Load() (Settings, error) {
	settings := Defaults()
	if s.path == "" {
		return settings, nil
	}
	data, err := os.ReadFile(s.path)
	if errors.Is(err, os.ErrNotExist) {
		return settings, nil
	}
	if err != nil {
		return settings, err
	}
	if len(data) == 0 {
		return settings, nil
	}
	if err := json.Unmarshal(data, &settings); err != nil {
		return Defaults(), err
	}
	if settings.ThemeName == "" {
		settings.ThemeName = Defaults().ThemeName
	}
	if settings.ThemeName == "Miami" {
		settings.ThemeName = "Synthwave"
	}
	if settings.DefaultFeed == "" {
		settings.DefaultFeed = Defaults().DefaultFeed
	}
	return settings, nil
}

func (s Store) Save(settings Settings) error {
	if err := os.MkdirAll(filepath.Dir(s.path), 0o700); err != nil {
		return err
	}
	data, err := json.MarshalIndent(settings, "", "  ")
	if err != nil {
		return err
	}
	data = append(data, '\n')
	tmp, err := os.CreateTemp(filepath.Dir(s.path), ".config-*.tmp")
	if err != nil {
		return err
	}
	tmpPath := tmp.Name()
	defer os.Remove(tmpPath)
	if _, err := tmp.Write(data); err != nil {
		tmp.Close()
		return err
	}
	if err := tmp.Chmod(fileMode); err != nil {
		tmp.Close()
		return err
	}
	if err := tmp.Sync(); err != nil {
		tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	return os.Rename(tmpPath, s.path)
}
