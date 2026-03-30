package theme

import (
	"encoding/json"
	"fmt"
	"strings"

	"dops/internal/domain"
)

// ThemeMode indicates whether to resolve dark or light theme tokens.
type ThemeMode int

const (
	ThemeDark  ThemeMode = iota
	ThemeLight
)

type ResolvedTheme struct {
	Name   string
	Colors map[string]string // token name → hex color (e.g. "background" → "#1a1b26")
}

func Resolve(tf *domain.ThemeFile, isDark bool) (*ResolvedTheme, error) {
	mode := ThemeDark
	if !isDark {
		mode = ThemeLight
	}

	resolved := &ResolvedTheme{
		Name:   tf.Name,
		Colors: make(map[string]string),
	}

	for name, raw := range tf.Theme {
		if err := resolveToken(resolved, tf.Defs, mode, name, raw); err != nil {
			return nil, fmt.Errorf("resolve token %q: %w", name, err)
		}
	}

	return resolved, nil
}

func resolveToken(resolved *ResolvedTheme, defs map[string]string, mode ThemeMode, prefix string, raw json.RawMessage) error {
	// Try as a simple ThemeToken first
	var token domain.ThemeToken
	if err := json.Unmarshal(raw, &token); err == nil && (token.Dark != "" || token.Light != "") {
		ref := token.Dark
		if mode == ThemeLight {
			ref = token.Light
		}
		hex, err := resolveRef(defs, ref)
		if err != nil {
			return err
		}
		resolved.Colors[prefix] = hex
		return nil
	}

	// Try as a nested map of tokens (e.g., risk: {low: {dark, light}, ...})
	var nested map[string]domain.ThemeToken
	if err := json.Unmarshal(raw, &nested); err == nil && len(nested) > 0 {
		for subName, subToken := range nested {
			ref := subToken.Dark
			if mode == ThemeLight {
				ref = subToken.Light
			}
			hex, err := resolveRef(defs, ref)
			if err != nil {
				return fmt.Errorf("nested %q: %w", subName, err)
			}
			resolved.Colors[prefix+"."+subName] = hex
		}
		return nil
	}

	return fmt.Errorf("cannot parse token value")
}

func resolveRef(defs map[string]string, ref string) (string, error) {
	if ref == "none" {
		return "none", nil
	}
	if strings.HasPrefix(ref, "#") {
		return ref, nil
	}
	hex, ok := defs[ref]
	if !ok {
		return "", fmt.Errorf("unknown def reference: %q", ref)
	}
	return hex, nil
}
