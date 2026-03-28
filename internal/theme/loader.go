package theme

import (
	_ "embed"
	"encoding/json"
	"fmt"
	"path/filepath"

	"dops/internal/domain"
)

//go:embed doop.json
var bundledDoop []byte

//go:embed tokyomidnight.json
var bundledTokyomidnight []byte

//go:embed catppuccin-mocha.json
var bundledCatppuccinMocha []byte

//go:embed catppuccin-latte.json
var bundledCatppuccinLatte []byte

//go:embed nord.json
var bundledNord []byte

//go:embed rosepine-dawn.json
var bundledRosepineDawn []byte

//go:embed espresso.json
var bundledEspresso []byte

//go:embed unicorn.json
var bundledUnicorn []byte

//go:embed dracula.json
var bundledDracula []byte

//go:embed solarized.json
var bundledSolarized []byte

//go:embed gruvbox.json
var bundledGruvbox []byte

//go:embed monokai.json
var bundledMonokai []byte

//go:embed kanagawa.json
var bundledKanagawa []byte

//go:embed everforest.json
var bundledEverforest []byte

//go:embed synthwave.json
var bundledSynthwave []byte

//go:embed one-dark.json
var bundledOneDark []byte

//go:embed nightowl.json
var bundledNightowl []byte

//go:embed github.json
var bundledGithub []byte

//go:embed ayu.json
var bundledAyu []byte

//go:embed zenburn.json
var bundledZenburn []byte

type ThemeLoader interface {
	Load(name string) (*domain.ThemeFile, error)
}

type FileSystem interface {
	ReadFile(path string) ([]byte, error)
}

type FileThemeLoader struct {
	fs        FileSystem
	themesDir string
}

func NewFileLoader(fs FileSystem, themesDir string) *FileThemeLoader {
	return &FileThemeLoader{fs: fs, themesDir: themesDir}
}

func (l *FileThemeLoader) Load(name string) (*domain.ThemeFile, error) {
	// 1. Try user theme
	userPath := filepath.Join(l.themesDir, name+".json")
	data, err := l.fs.ReadFile(userPath)
	if err == nil {
		return parseTheme(data)
	}

	// 2. Try bundled theme
	tf, err := l.loadBundled(name)
	if err == nil {
		return tf, nil
	}

	// 3. Fall back to tokyomidnight (default theme)
	if name != "tokyomidnight" {
		return l.loadBundled("tokyomidnight")
	}

	return nil, fmt.Errorf("theme %q not found and fallback failed", name)
}

func (l *FileThemeLoader) loadBundled(name string) (*domain.ThemeFile, error) {
	switch name {
	case "doop":
		return parseTheme(bundledDoop)
	case "tokyomidnight":
		return parseTheme(bundledTokyomidnight)
	case "catppuccin-mocha":
		return parseTheme(bundledCatppuccinMocha)
	case "catppuccin-latte":
		return parseTheme(bundledCatppuccinLatte)
	case "nord":
		return parseTheme(bundledNord)
	case "rosepine-dawn":
		return parseTheme(bundledRosepineDawn)
	case "espresso":
		return parseTheme(bundledEspresso)
	case "unicorn":
		return parseTheme(bundledUnicorn)
	case "dracula":
		return parseTheme(bundledDracula)
	case "solarized":
		return parseTheme(bundledSolarized)
	case "gruvbox":
		return parseTheme(bundledGruvbox)
	case "monokai":
		return parseTheme(bundledMonokai)
	case "kanagawa":
		return parseTheme(bundledKanagawa)
	case "everforest":
		return parseTheme(bundledEverforest)
	case "synthwave":
		return parseTheme(bundledSynthwave)
	case "one-dark":
		return parseTheme(bundledOneDark)
	case "nightowl":
		return parseTheme(bundledNightowl)
	case "github":
		return parseTheme(bundledGithub)
	case "ayu":
		return parseTheme(bundledAyu)
	case "zenburn":
		return parseTheme(bundledZenburn)
	default:
		return nil, fmt.Errorf("no bundled theme %q", name)
	}
}

func parseTheme(data []byte) (*domain.ThemeFile, error) {
	var tf domain.ThemeFile
	if err := json.Unmarshal(data, &tf); err != nil {
		return nil, fmt.Errorf("parse theme: %w", err)
	}
	return &tf, nil
}

var _ ThemeLoader = (*FileThemeLoader)(nil)
