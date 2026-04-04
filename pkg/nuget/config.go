package nuget

// NuGet.Config discovery, XML parsing, and credential injection.

import (
	"encoding/xml"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"strings"
)

// configuration represents the XML structure of a NuGet.Config file.
type configuration struct {
	XMLName                  xml.Name           `xml:"configuration"`
	PackageSources           *packageSources    `xml:"packageSources"`
	PackageSourceCredentials *sourceCredentials `xml:"packageSourceCredentials"`
	Other                    []rawSection       `xml:",any"`
}

type packageSources struct {
	Items []addItem `xml:",any"`
}

type addItem struct {
	XMLName xml.Name
	Key     string     `xml:"key,attr,omitempty"`
	Value   string     `xml:"value,attr,omitempty"`
	Attrs   []xml.Attr `xml:",any,attr"`
}

type sourceCredentials struct {
	Sources []credentialSource `xml:",any"`
}

type credentialSource struct {
	XMLName xml.Name
	Items   []addItem `xml:"add"`
}

type rawSection struct {
	XMLName xml.Name
	Attrs   []xml.Attr `xml:",any,attr"`
	Content []byte     `xml:",innerxml"`
}

// ResolveConfigPath returns the path to the NuGet.Config file.
// If configFile is non-empty the given path is used (returns "" when the file
// does not exist). Otherwise the current directory and each of its parents are
// searched for NuGet.Config / nuget.config.
func ResolveConfigPath(configFile string) string {
	if configFile != "" {
		if _, err := os.Stat(configFile); err == nil {
			return configFile
		}
		return ""
	}

	dir, err := os.Getwd()
	if err != nil {
		return ""
	}
	for {
		for _, name := range []string{"NuGet.Config", "nuget.config", "NuGet.config", "nuget.Config"} {
			p := filepath.Join(dir, name)
			if _, err := os.Stat(p); err == nil {
				return p
			}
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
		dir = parent
	}
	return ""
}

// IsGitHubPackagesURL returns true when rawURL points to a GitHub Packages
// NuGet registry. Recognised patterns:
//   - github.com:   nuget.pkg.github.com
//   - GHES legacy: /_registry/nuget/ path prefix
//   - GHES modern: nuget.<host> subdomain
func IsGitHubPackagesURL(rawURL string) bool {
	if strings.Contains(rawURL, "nuget.pkg.github.com") || strings.Contains(rawURL, "/_registry/nuget/") {
		return true
	}
	u, err := url.Parse(rawURL)
	if err != nil {
		return false
	}
	return strings.HasPrefix(u.Hostname(), "nuget.")
}

// WriteConfigWithCredentials reads the NuGet.Config at srcPath, injects
// GitHub Packages credentials using token, and writes the result to dstPath.
// dstPath may equal srcPath to overwrite the file in place.
// Returns an error when no GitHub Packages sources are found in the config.
func WriteConfigWithCredentials(srcPath, dstPath, token string) error {
	data, err := os.ReadFile(srcPath)
	if err != nil {
		return fmt.Errorf("failed to read NuGet.Config: %w", err)
	}

	var cfg configuration
	if err := xml.Unmarshal(data, &cfg); err != nil {
		return fmt.Errorf("failed to parse NuGet.Config: %w", err)
	}

	// Collect source keys that point to GitHub Packages.
	githubSourceKeys := make(map[string]bool)
	if cfg.PackageSources != nil {
		for _, item := range cfg.PackageSources.Items {
			if item.XMLName.Local == "add" && IsGitHubPackagesURL(item.Value) {
				githubSourceKeys[item.Key] = true
			}
		}
	}

	if len(githubSourceKeys) == 0 {
		return fmt.Errorf("no GitHub Packages sources found in NuGet.Config")
	}

	if cfg.PackageSourceCredentials == nil {
		cfg.PackageSourceCredentials = &sourceCredentials{}
	}

	// Update existing credential entries for GitHub Packages sources.
	existingCreds := make(map[string]bool)
	for i, src := range cfg.PackageSourceCredentials.Sources {
		if githubSourceKeys[src.XMLName.Local] {
			existingCreds[src.XMLName.Local] = true
			cfg.PackageSourceCredentials.Sources[i].Items = []addItem{
				{XMLName: xml.Name{Local: "add"}, Key: "Username", Value: "gh-pkg-kit"},
				{XMLName: xml.Name{Local: "add"}, Key: "ClearTextPassword", Value: token},
			}
		}
	}

	// Add credential entries for sources that had none yet.
	for key := range githubSourceKeys {
		if !existingCreds[key] {
			cfg.PackageSourceCredentials.Sources = append(cfg.PackageSourceCredentials.Sources, credentialSource{
				XMLName: xml.Name{Local: key},
				Items: []addItem{
					{XMLName: xml.Name{Local: "add"}, Key: "Username", Value: "gh-pkg-kit"},
					{XMLName: xml.Name{Local: "add"}, Key: "ClearTextPassword", Value: token},
				},
			})
		}
	}

	output, err := xml.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal NuGet.Config: %w", err)
	}

	content := append([]byte(xml.Header), output...)
	if err := os.WriteFile(dstPath, content, 0600); err != nil {
		return fmt.Errorf("failed to write NuGet.Config: %w", err)
	}

	return nil
}
