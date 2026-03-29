package packages

import (
	"archive/tar"
	"compress/gzip"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"
)

// Manager handles plugin installation, removal, and updates.
type Manager struct {
	registry  *RegistryClient
	pluginDir string
	store     PluginStore
}

// PluginStore is the subset of KitStore that the package manager uses.
// Uses simple types to avoid import cycles with the kit package.
type PluginStore interface {
	SaveInstalled(name, owner, version, binaryPath, manifest string, installedAt time.Time) error
	LoadInstalled() ([]InstalledRecord, error)
	DeleteInstalled(name string) error
}

// InstalledRecord describes an installed plugin binary.
type InstalledRecord struct {
	Name        string
	Owner       string
	Version     string
	BinaryPath  string
	Manifest    string
	InstalledAt time.Time
}

// NewManager creates a package manager.
func NewManager(registry *RegistryClient, pluginDir string, store PluginStore) *Manager {
	return &Manager{
		registry:  registry,
		pluginDir: pluginDir,
		store:     store,
	}
}

// Search queries all registries for plugins matching the query or capabilities.
func (m *Manager) Search(query string, capabilities []string) ([]PluginSummary, error) {
	return m.registry.Search(query, capabilities)
}

// Install downloads a plugin binary, verifies its checksum, and stores it.
func (m *Manager) Install(owner, name, version string) (*InstalledRecord, error) {
	manifest, err := m.registry.FetchManifest(owner, name, version)
	if err != nil {
		return nil, fmt.Errorf("packages.install: %w", err)
	}

	platform := runtime.GOOS + "/" + runtime.GOARCH
	binary, ok := manifest.Platforms[platform]
	if !ok {
		return nil, fmt.Errorf("packages.install: %s/%s has no binary for %s", owner, name, platform)
	}

	// Create plugin directory
	installDir := filepath.Join(m.pluginDir, owner, name, manifest.Version)
	if err := os.MkdirAll(installDir, 0755); err != nil {
		return nil, fmt.Errorf("packages.install: create dir: %w", err)
	}

	// Download and extract
	var binaryPath string
	if strings.HasSuffix(binary.URL, ".tar.gz") || strings.HasSuffix(binary.URL, ".tgz") {
		extracted, err := downloadAndExtract(installDir, binary.URL)
		if err != nil {
			return nil, fmt.Errorf("packages.install: download+extract: %w", err)
		}
		binaryPath = extracted
	} else {
		// Raw binary download (for local testing or simple distributions)
		binaryPath = filepath.Join(installDir, name)
		if err := downloadFile(binaryPath, binary.URL); err != nil {
			return nil, fmt.Errorf("packages.install: download: %w", err)
		}
		os.Chmod(binaryPath, 0755)
	}

	// Verify checksum
	if binary.SHA256 != "" {
		computed, err := fileChecksum(binaryPath)
		if err != nil {
			os.RemoveAll(installDir)
			return nil, fmt.Errorf("packages.install: checksum compute: %w", err)
		}
		if computed != binary.SHA256 {
			os.RemoveAll(installDir)
			return nil, fmt.Errorf("packages.install: checksum mismatch for %s/%s (expected %s, got %s)", owner, name, binary.SHA256, computed)
		}
	}

	// Save manifest alongside binary
	manifestJSON, _ := json.Marshal(manifest)
	os.WriteFile(filepath.Join(installDir, "manifest.json"), manifestJSON, 0644)

	record := &InstalledRecord{
		Name:        name,
		Owner:       owner,
		Version:     manifest.Version,
		BinaryPath:  binaryPath,
		Manifest:    string(manifestJSON),
		InstalledAt: time.Now(),
	}

	if m.store != nil {
		m.store.SaveInstalled(record.Name, record.Owner, record.Version, record.BinaryPath, record.Manifest, record.InstalledAt)
	}

	return record, nil
}

// Remove deletes an installed plugin.
func (m *Manager) Remove(name string) error {
	if m.store != nil {
		plugins, _ := m.store.LoadInstalled()
		for _, p := range plugins {
			if p.Name == name {
				dir := filepath.Dir(p.BinaryPath)
				os.RemoveAll(dir)
				m.store.DeleteInstalled(name)
				return nil
			}
		}
	}
	return fmt.Errorf("plugin %q not installed", name)
}

// Update downloads the latest version of an installed plugin, replacing the old one.
func (m *Manager) Update(owner, name string) (oldVersion, newVersion string, err error) {
	current, err := m.GetInstalled(name)
	if err != nil {
		return "", "", err
	}
	manifest, err := m.registry.FetchManifest(owner, name, "")
	if err != nil {
		return "", "", err
	}
	if manifest.Version == current.Version {
		return current.Version, current.Version, nil // already latest
	}
	m.Remove(name)
	_, err = m.Install(owner, name, manifest.Version)
	return current.Version, manifest.Version, err
}

// GetInstalled returns info about an installed plugin.
func (m *Manager) GetInstalled(name string) (*InstalledRecord, error) {
	if m.store == nil {
		return nil, fmt.Errorf("no store configured")
	}
	plugins, err := m.store.LoadInstalled()
	if err != nil {
		return nil, err
	}
	for _, p := range plugins {
		if p.Name == name {
			return &p, nil
		}
	}
	return nil, fmt.Errorf("plugin %q not installed", name)
}

// ListInstalled returns all installed plugins.
func (m *Manager) ListInstalled() ([]InstalledRecord, error) {
	if m.store == nil {
		return nil, nil
	}
	return m.store.LoadInstalled()
}

// --- Download helpers ---

func downloadFile(path, url string) error {
	resp, err := http.Get(url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		return fmt.Errorf("HTTP %d from %s", resp.StatusCode, url)
	}
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()
	_, err = io.Copy(f, resp.Body)
	return err
}

func downloadAndExtract(destDir, url string) (string, error) {
	resp, err := http.Get(url)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		return "", fmt.Errorf("HTTP %d from %s", resp.StatusCode, url)
	}

	gz, err := gzip.NewReader(resp.Body)
	if err != nil {
		return "", fmt.Errorf("gzip: %w", err)
	}
	defer gz.Close()

	tr := tar.NewReader(gz)
	var binaryPath string
	for {
		hdr, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return "", err
		}

		// Prevent path traversal
		target := filepath.Join(destDir, filepath.Clean(hdr.Name))
		if !strings.HasPrefix(target, filepath.Clean(destDir)+string(os.PathSeparator)) && target != filepath.Clean(destDir) {
			continue // skip entries that escape destDir
		}

		switch hdr.Typeflag {
		case tar.TypeDir:
			os.MkdirAll(target, 0755)
		case tar.TypeReg:
			f, err := os.Create(target)
			if err != nil {
				return "", err
			}
			io.Copy(f, tr)
			f.Close()
			if hdr.Mode&0111 != 0 {
				os.Chmod(target, 0755)
				if binaryPath == "" {
					binaryPath = target
				}
			}
		}
	}

	if binaryPath == "" {
		return "", fmt.Errorf("no executable found in archive")
	}
	return binaryPath, nil
}

func fileChecksum(path string) (string, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer f.Close()
	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		return "", err
	}
	return hex.EncodeToString(h.Sum(nil)), nil
}
