package tester

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"
)

type PackageType string

const (
	TypeCommonJS PackageType = "commonjs"
	TypeESM      PackageType = "module"
	TypeDual     PackageType = "dual"
	TypeUnknown  PackageType = "unknown"
)

type PackageVersionInfo struct {
	Name    string            `json:"name"`
	Version string            `json:"version"`
	Type    string            `json:"type,omitempty"`
	Main    string            `json:"main,omitempty"`
	Module  string            `json:"module,omitempty"`
	Exports any               `json:"exports,omitempty"`
	Bin     json.RawMessage   `json:"bin,omitempty"`
	Scripts map[string]string `json:"scripts,omitempty"`
}

type PackageInfo struct {
	Name       string            `json:"name"`
	Version    string            `json:"version"`
	Type       PackageType       `json:"type"`
	Main       string            `json:"main"`
	Module     string            `json:"module"`
	Exports    any               `json:"exports"`
	Bin        map[string]string `json:"bin"`
	HasBin     bool              `json:"has_bin"`
	HasPrepare bool              `json:"has_prepare"`
	HasInstall bool              `json:"has_install"`
	Scripts    map[string]string `json:"scripts"`
}

type RegistryPackage struct {
	Name     string                        `json:"name"`
	Versions map[string]PackageVersionInfo `json:"versions"`
}

type Detector struct {
	HTTPClient    *http.Client
	RegistryURL   string
	RegistryOwner string
	RegistryToken string
}

func NewDetector() *Detector {
	return &Detector{
		HTTPClient: &http.Client{Timeout: 30 * time.Second},
		RegistryURL: "https://registry.npmjs.org",
	}
}

func NewDetectorWithRegistry(registryURL, registryOwner, registryToken string) *Detector {
	return &Detector{
		HTTPClient: &http.Client{Timeout: 30 * time.Second},
		RegistryURL: strings.TrimSuffix(registryURL, "/"),
		RegistryOwner: registryOwner,
		RegistryToken: registryToken,
	}
}

func (d *Detector) DetectPackage(name, version string) (*PackageInfo, error) {
	var url string
	if d.RegistryOwner != "" {
		url = fmt.Sprintf("%s/api/packages/%s/npm/%s", d.RegistryURL, d.RegistryOwner, name)
	} else {
		url = fmt.Sprintf("%s/%s", strings.TrimSuffix(d.RegistryURL, "/"), name)
	}

	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	if d.RegistryToken != "" {
		req.Header.Set("Authorization", "Bearer "+d.RegistryToken)
	}

	resp, err := d.HTTPClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch package info: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("registry returned status %d", resp.StatusCode)
	}

	var registryPkg RegistryPackage
	if err := json.NewDecoder(resp.Body).Decode(&registryPkg); err != nil {
		return nil, fmt.Errorf("failed to decode registry response: %w", err)
	}

	versionInfo, exists := registryPkg.Versions[version]
	if !exists {
		return nil, fmt.Errorf("version %s not found for package %s", version, name)
	}

	info := &PackageInfo{
		Name: versionInfo.Name,
		Version: versionInfo.Version,
		Main: versionInfo.Main,
		Module: versionInfo.Module,
		Exports: versionInfo.Exports,
		Scripts: versionInfo.Scripts,
	}
	info.Type = d.detectModuleType(&versionInfo)
	info.Bin, info.HasBin = d.parseBin(versionInfo.Bin)
	info.HasPrepare = d.hasScript(versionInfo.Scripts, "prepare")
	info.HasInstall = d.hasScript(versionInfo.Scripts, "preinstall") || d.hasScript(versionInfo.Scripts, "postinstall") || d.hasScript(versionInfo.Scripts, "install")

	return info, nil
}

func (d *Detector) detectModuleType(v *PackageVersionInfo) PackageType {
	if v.Type == "module" {
		return TypeESM
	}
	if v.Module != "" {
		return TypeESM
	}
	if v.Exports != nil {
		return TypeDual
	}
	return TypeCommonJS
}

func (d *Detector) parseBin(bin json.RawMessage) (map[string]string, bool) {
	if len(bin) == 0 {
		return nil, false
	}
	var binStr string
	if err := json.Unmarshal(bin, &binStr); err == nil {
		return map[string]string{"default": binStr}, true
	}
	var binMap map[string]string
	if err := json.Unmarshal(bin, &binMap); err == nil {
		return binMap, len(binMap) > 0
	}
	return nil, false
}

func (d *Detector) hasScript(scripts map[string]string, name string) bool {
	if scripts == nil {
		return false
	}
	_, exists := scripts[name]
	return exists
}

func (d *Detector) GetPackageJSONType(info *PackageInfo) string {
	if info.Type == TypeESM {
		return "module"
	}
	return "commonjs"
}

func NormalizePackageName(name string) string {
	if strings.HasPrefix(name, "@") {
		parts := strings.SplitN(name, "/", 2)
		if len(parts) == 2 {
			return parts[0][1:] + "__" + parts[1]
		}
	}
	return name
}
