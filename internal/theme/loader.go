package theme

import (
	_ "embed"
	"encoding/json"
	"fmt"
	"math/rand"
	"path/filepath"

	"dops/internal/domain"
)

// bundledNames lists all real theme names for random selection.
var bundledNames = []string{
	"doop", "tokyomidnight", "catppuccin-mocha", "catppuccin-latte",
	"nord", "rosepine-dawn", "espresso", "unicorn", "dracula",
	"solarized", "gruvbox", "monokai", "kanagawa", "everforest",
	"synthwave", "one-dark", "nightowl", "github", "ayu", "zenburn",
}

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

// bundledThemes maps theme names to their embedded JSON data.
var bundledThemes = map[string][]byte{
	"doop":             bundledDoop,
	"tokyomidnight":    bundledTokyomidnight,
	"catppuccin-mocha": bundledCatppuccinMocha,
	"catppuccin-latte": bundledCatppuccinLatte,
	"nord":             bundledNord,
	"rosepine-dawn":    bundledRosepineDawn,
	"espresso":         bundledEspresso,
	"unicorn":          bundledUnicorn,
	"dracula":          bundledDracula,
	"solarized":        bundledSolarized,
	"gruvbox":          bundledGruvbox,
	"monokai":          bundledMonokai,
	"kanagawa":         bundledKanagawa,
	"everforest":       bundledEverforest,
	"synthwave":        bundledSynthwave,
	"one-dark":         bundledOneDark,
	"nightowl":         bundledNightowl,
	"github":           bundledGithub,
	"ayu":              bundledAyu,
	"zenburn":          bundledZenburn,
}

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
	// Random theme: pick a different bundled theme each launch.
	if name == "rainbow" {
		name = bundledNames[rand.Intn(len(bundledNames))] // #nosec G404 -- cosmetic theme selection, not security-sensitive
	}

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

	// 3. Fall back to github (default theme)
	if name != "github" {
		return l.loadBundled("github")
	}

	return nil, fmt.Errorf("theme %q not found and fallback failed", name)
}

func (l *FileThemeLoader) loadBundled(name string) (*domain.ThemeFile, error) {
	data, ok := bundledThemes[name]
	if !ok {
		return nil, fmt.Errorf("no bundled theme %q", name)
	}
	return parseTheme(data)
}

func parseTheme(data []byte) (*domain.ThemeFile, error) {
	var themeFile domain.ThemeFile
	if err := json.Unmarshal(data, &themeFile); err != nil {
		return nil, fmt.Errorf("parse theme: %w", err)
	}
	return &themeFile, nil
}

// BundledNames returns the list of all built-in theme names.
func BundledNames() []string {
	out := make([]string, len(bundledNames))
	copy(out, bundledNames)
	return out
}

var _ ThemeLoader = (*FileThemeLoader)(nil)
