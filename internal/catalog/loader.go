package catalog

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"dops/internal/domain"

	"gopkg.in/yaml.v3"
)

type CatalogWithRunbooks struct {
	Catalog  domain.Catalog
	Runbooks []domain.Runbook
}

type CatalogLoader interface {
	LoadAll(catalogs []domain.Catalog, defaultRisk domain.RiskLevel) ([]CatalogWithRunbooks, error)
	FindByID(id string) (*domain.Runbook, *domain.Catalog, error)
}

type FileSystem interface {
	ReadFile(path string) ([]byte, error)
	ReadDir(path string) ([]os.DirEntry, error)
}

type DiskCatalogLoader struct {
	fs     FileSystem
	loaded []CatalogWithRunbooks
}

func NewDiskLoader(fs FileSystem) *DiskCatalogLoader {
	return &DiskCatalogLoader{fs: fs}
}

func (l *DiskCatalogLoader) LoadAll(catalogs []domain.Catalog, defaultRisk domain.RiskLevel) ([]CatalogWithRunbooks, error) {
	var result []CatalogWithRunbooks

	for _, cat := range catalogs {
		if !cat.Active {
			continue
		}

		ceiling := cat.Policy.MaxRiskLevel
		if ceiling == "" {
			ceiling = defaultRisk
		}

		runbooks, err := l.loadCatalog(cat.Name, expandHome(cat.Path), ceiling)
		if err != nil {
			return nil, fmt.Errorf("load catalog %q: %w", cat.Name, err)
		}

		if len(runbooks) > 0 {
			result = append(result, CatalogWithRunbooks{
				Catalog:  cat,
				Runbooks: runbooks,
			})
		}
	}

	l.loaded = result
	return result, nil
}

func (l *DiskCatalogLoader) FindByID(id string) (*domain.Runbook, *domain.Catalog, error) {
	for i := range l.loaded {
		for j := range l.loaded[i].Runbooks {
			if l.loaded[i].Runbooks[j].ID == id {
				return &l.loaded[i].Runbooks[j], &l.loaded[i].Catalog, nil
			}
		}
	}
	return nil, nil, fmt.Errorf("runbook %q not found", id)
}

func (l *DiskCatalogLoader) loadCatalog(catalogName, catalogPath string, ceiling domain.RiskLevel) ([]domain.Runbook, error) {
	entries, err := l.fs.ReadDir(catalogPath)
	if err != nil {
		return nil, fmt.Errorf("read catalog dir: %w", err)
	}

	var runbooks []domain.Runbook

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		rbPath := filepath.Join(catalogPath, entry.Name(), "runbook.yaml")
		rb, err := l.loadRunbook(rbPath)
		if err != nil {
			return nil, fmt.Errorf("load runbook %q: %w", entry.Name(), err)
		}

		if rb.RiskLevel.Exceeds(ceiling) {
			continue
		}

		// Generate ID as catalog.runbook if not set in YAML.
		if rb.ID == "" {
			rb.ID = catalogName + "." + entry.Name()
		}

		runbooks = append(runbooks, *rb)
	}

	return runbooks, nil
}

func (l *DiskCatalogLoader) loadRunbook(path string) (*domain.Runbook, error) {
	data, err := l.fs.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read runbook: %w", err)
	}

	var rb domain.Runbook
	if err := yaml.Unmarshal(data, &rb); err != nil {
		return nil, fmt.Errorf("parse runbook: %w", err)
	}

	return &rb, nil
}

func expandHome(path string) string {
	if strings.HasPrefix(path, "~/") {
		home, err := os.UserHomeDir()
		if err != nil {
			return path
		}
		return filepath.Join(home, path[2:])
	}
	return path
}

var _ CatalogLoader = (*DiskCatalogLoader)(nil)
