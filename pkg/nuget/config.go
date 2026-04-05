package nuget

// NuGet.Config discovery, XML parsing, and credential injection.

import (
	"encoding/xml"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"strings"

	"github.com/cli/go-gh/v2/pkg/auth"
	"github.com/srz-zumix/go-gh-extension/pkg/ioutil"
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
	u, err := url.Parse(rawURL)
	if err != nil {
		return false
	}

	host := u.Hostname()
	if host == "nuget.pkg.github.com" {
		return true
	}
	if strings.HasPrefix(host, "nuget.") {
		return true
	}
	return strings.HasPrefix(u.EscapedPath(), "/_registry/nuget/")
}

// gitHubHostFromNuGetURL extracts the GitHub authentication host from a
// GitHub Packages NuGet registry URL.
// Examples:
//   - https://nuget.pkg.github.com/<owner>/... → "github.com"
//   - https://nuget.<ghes-host>/<owner>/...    → "<ghes-host>"
//   - https://<ghes-host>/_registry/nuget/...  → "<ghes-host>"
func gitHubHostFromNuGetURL(rawURL string) string {
	u, err := url.Parse(rawURL)
	if err != nil {
		return ""
	}
	h := u.Hostname()
	if h == "nuget.pkg.github.com" {
		return "github.com"
	}
	if after, ok := strings.CutPrefix(h, "nuget."); ok {
		return after
	}
	if strings.HasPrefix(u.EscapedPath(), "/_registry/nuget/") {
		return h
	}
	return ""
}

// isXMLNCNameStart reports whether r is a valid first character of an XML NCName
// (XML 1.0 spec, §2.3, excluding ':').
func isXMLNCNameStart(r rune) bool {
	return r == '_' ||
		(r >= 'A' && r <= 'Z') ||
		(r >= 'a' && r <= 'z') ||
		(r >= 0xC0 && r <= 0xD6) ||
		(r >= 0xD8 && r <= 0xF6) ||
		(r >= 0xF8 && r <= 0x2FF) ||
		(r >= 0x370 && r <= 0x37D) ||
		(r >= 0x37F && r <= 0x1FFF) ||
		(r >= 0x200C && r <= 0x200D) ||
		(r >= 0x2070 && r <= 0x218F) ||
		(r >= 0x2C00 && r <= 0x2FEF) ||
		(r >= 0x3001 && r <= 0xD7FF) ||
		(r >= 0xF900 && r <= 0xFDCF) ||
		(r >= 0xFDF0 && r <= 0xFFFD) ||
		(r >= 0x10000 && r <= 0xEFFFF)
}

// isXMLNCNameChar reports whether r is valid anywhere in an XML NCName
// (XML 1.0 spec, §2.3, excluding ':').
func isXMLNCNameChar(r rune) bool {
	return isXMLNCNameStart(r) ||
		r == '-' || r == '.' ||
		(r >= '0' && r <= '9') ||
		r == 0xB7 ||
		(r >= 0x0300 && r <= 0x036F) ||
		(r >= 0x203F && r <= 0x2040)
}

// nugetEncodeSourceKey encodes a NuGet source key to a valid XML NCName,
// matching .NET's XmlConvert.EncodeName. Characters invalid in XML NCNames are
// replaced with _xHHHH_ (upper-hex, zero-padded to 4 digits). An underscore
// immediately followed by 'x' is also escaped (_x005F_) to prevent ambiguity
// with the encoding scheme itself.
func nugetEncodeSourceKey(key string) string {
	if key == "" {
		return key
	}
	runes := []rune(key)
	var b strings.Builder
	for i, r := range runes {
		if (i == 0 && !isXMLNCNameStart(r)) || (i > 0 && !isXMLNCNameChar(r)) {
			fmt.Fprintf(&b, "_x%04X_", r)
			continue
		}
		// Escape '_x' to prevent collision with the _xHHHH_ encoding scheme.
		if r == '_' && i+1 < len(runes) && runes[i+1] == 'x' {
			b.WriteString("_x005F_")
			continue
		}
		b.WriteRune(r)
	}
	return b.String()
}

// WriteConfigWithCredentials reads the NuGet.Config at srcPath, injects
// GitHub Packages credentials by looking up the gh auth token for each
// source's host, and writes the result to dstPath.
// dstPath may equal srcPath to overwrite the file in place.
// Returns an error when no GitHub Packages sources are found in the config.
func WriteConfigWithCredentials(srcPath, dstPath string) error {
	data, err := os.ReadFile(srcPath)
	if err != nil {
		return fmt.Errorf("failed to read NuGet.Config: %w", err)
	}

	var cfg configuration
	if err := xml.Unmarshal(data, &cfg); err != nil {
		return fmt.Errorf("failed to parse NuGet.Config: %w", err)
	}

	// Collect source keys and their auth hosts for GitHub Packages sources.
	// encodedKey is the XML NCName-safe encoding of key used in credential sections.
	type ghSource struct{ key, encodedKey, host string }
	var githubSources []ghSource
	githubSourceKeys := make(map[string]bool)
	if cfg.PackageSources != nil {
		for _, item := range cfg.PackageSources.Items {
			if item.XMLName.Local == "add" && IsGitHubPackagesURL(item.Value) {
				key := strings.TrimSpace(item.Key)
				if key == "" {
					return fmt.Errorf("invalid GitHub Packages source in NuGet.Config: empty key for URL %q", item.Value)
				}

				host := gitHubHostFromNuGetURL(item.Value)
				if host == "" {
					return fmt.Errorf("invalid GitHub Packages source in NuGet.Config: failed to derive GitHub host from URL %q", item.Value)
				}

				encodedKey := nugetEncodeSourceKey(key)
				githubSources = append(githubSources, ghSource{key: key, encodedKey: encodedKey, host: host})
				githubSourceKeys[encodedKey] = true
			}
		}
	}

	if len(githubSourceKeys) == 0 {
		return fmt.Errorf("no GitHub Packages sources found in NuGet.Config")
	}

	// Build a map from encoded source key to token, looking up per host.
	// sourceTokens is keyed by the XML-encoded form because that is what NuGet
	// writes into <packageSourceCredentials> element names.
	sourceTokens := make(map[string]string)
	for _, src := range githubSources {
		if _, seen := sourceTokens[src.encodedKey]; seen {
			continue
		}
		token, _ := auth.TokenForHost(src.host)
		if token != "" {
			sourceTokens[src.encodedKey] = token
		}
	}

	if len(sourceTokens) == 0 {
		return fmt.Errorf("no gh auth token found for any GitHub Packages source in NuGet.Config")
	}

	if cfg.PackageSourceCredentials == nil {
		cfg.PackageSourceCredentials = &sourceCredentials{}
	}

	// Update existing credential entries for GitHub Packages sources.
	// Only Username and ClearTextPassword are upserted; other keys (e.g.
	// ValidAuthenticationTypes) are preserved.
	existingCreds := make(map[string]bool)
	for i, src := range cfg.PackageSourceCredentials.Sources {
		token, ok := sourceTokens[src.XMLName.Local]
		if !ok {
			continue
		}
		existingCreds[src.XMLName.Local] = true
		updated := make([]addItem, 0, len(src.Items))
		usernameSet, passwordSet := false, false
		for _, item := range src.Items {
			switch item.Key {
			case "Username":
				item.Value = "gh-pkg-kit"
				usernameSet = true
			case "ClearTextPassword":
				item.Value = token
				passwordSet = true
			}
			updated = append(updated, item)
		}
		if !usernameSet {
			updated = append(updated, addItem{XMLName: xml.Name{Local: "add"}, Key: "Username", Value: "gh-pkg-kit"})
		}
		if !passwordSet {
			updated = append(updated, addItem{XMLName: xml.Name{Local: "add"}, Key: "ClearTextPassword", Value: token})
		}
		cfg.PackageSourceCredentials.Sources[i].Items = updated
	}

	// Add credential entries for sources that had none yet.
	// Iterate in package source order to keep output stable across runs.
	addedCreds := make(map[string]bool)
	for _, src := range githubSources {
		if existingCreds[src.encodedKey] || addedCreds[src.encodedKey] {
			continue
		}
		token, ok := sourceTokens[src.encodedKey]
		if !ok {
			continue
		}
		cfg.PackageSourceCredentials.Sources = append(cfg.PackageSourceCredentials.Sources, credentialSource{
			XMLName: xml.Name{Local: src.encodedKey},
			Items: []addItem{
				{XMLName: xml.Name{Local: "add"}, Key: "Username", Value: "gh-pkg-kit"},
				{XMLName: xml.Name{Local: "add"}, Key: "ClearTextPassword", Value: token},
			},
		})
		addedCreds[src.encodedKey] = true
	}

	output, err := xml.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal NuGet.Config: %w", err)
	}

	content := append([]byte(xml.Header), output...)

	if err := ioutil.WriteFileAtomic(dstPath, content, 0600); err != nil {
		return fmt.Errorf("failed to write NuGet.Config: %w", err)
	}

	return nil
}
