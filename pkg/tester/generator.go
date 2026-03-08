package tester

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"text/template"
)

type PackageJSON struct {
	Name         string            `json:"name"`
	Version      string            `json:"version"`
	Description  string            `json:"description"`
	Private      bool              `json:"private"`
	Type         string            `json:"type,omitempty"`
	Dependencies map[string]string `json:"dependencies"`
}

type TestPackage struct {
	Name           string
	Version        string
	PackageName    string
	PackageVersion string
	ModuleType     string
	OutputDir      string
}

type Generator struct {
	templatesDir string
	detector     *Detector
}

func NewGenerator(templatesDir string, registry RegistryConfig) *Generator {
	return &Generator{templatesDir: templatesDir, detector: NewDetector(registry)}
}

func (g *Generator) GenerateAll(name, version, outputDir string) ([]string, error) {
	info, err := g.detector.DetectPackage(name, version)
	if err != nil {
		return nil, fmt.Errorf("failed to detect package: %w", err)
	}

	normalizedName := NormalizePackageName(name)
	pkgDir := filepath.Join(outputDir, fmt.Sprintf("%s@%s", normalizedName, version))

	var generatedDirs []string

	installDir := filepath.Join(pkgDir, "install")
	if err := g.generateInstallTest(info, installDir); err != nil {
		return nil, fmt.Errorf("failed to generate install test: %w", err)
	}
	generatedDirs = append(generatedDirs, installDir)

	importDir := filepath.Join(pkgDir, "import")
	if err := g.generateImportTest(info, importDir); err != nil {
		return nil, fmt.Errorf("failed to generate import test: %w", err)
	}
	generatedDirs = append(generatedDirs, importDir)

	protoDir := filepath.Join(pkgDir, "prototype")
	if err := g.generatePrototypeTest(info, protoDir); err != nil {
		return nil, fmt.Errorf("failed to generate prototype test: %w", err)
	}
	generatedDirs = append(generatedDirs, protoDir)

	if info.HasBin {
		cliDir := filepath.Join(pkgDir, "cli")
		if err := g.generateCLITest(info, cliDir); err != nil {
			return nil, fmt.Errorf("failed to generate CLI test: %w", err)
		}
		generatedDirs = append(generatedDirs, cliDir)
	}

	return generatedDirs, nil
}

func (g *Generator) generateInstallTest(info *PackageInfo, outputDir string) error {
	data := TestPackage{
		Name:           fmt.Sprintf("test-install-%s", NormalizePackageName(info.Name)),
		Version:        "1.0.0",
		PackageName:    info.Name,
		PackageVersion: info.Version,
		ModuleType:     g.detector.GetPackageJSONType(info),
		OutputDir:      outputDir,
	}
	pkgJSON := PackageJSON{
		Name:         data.Name,
		Version:      data.Version,
		Description:  fmt.Sprintf("Install-time behavior test for %s@%s", info.Name, info.Version),
		Private:      true,
		Dependencies: map[string]string{info.Name: info.Version},
	}
	return g.generateTestPackage("install-test", data, outputDir, pkgJSON)
}

func (g *Generator) generateImportTest(info *PackageInfo, outputDir string) error {
	data := TestPackage{
		Name:           fmt.Sprintf("test-import-%s", NormalizePackageName(info.Name)),
		Version:        "1.0.0",
		PackageName:    info.Name,
		PackageVersion: info.Version,
		ModuleType:     g.detector.GetPackageJSONType(info),
		OutputDir:      outputDir,
	}
	pkgJSON := PackageJSON{
		Name:         data.Name,
		Version:      data.Version,
		Description:  fmt.Sprintf("Import-time behavior test for %s@%s", info.Name, info.Version),
		Private:      true,
		Type:         data.ModuleType,
		Dependencies: map[string]string{info.Name: info.Version},
	}
	return g.generateTestPackage("import-test", data, outputDir, pkgJSON)
}

func (g *Generator) generatePrototypeTest(info *PackageInfo, outputDir string) error {
	data := TestPackage{
		Name:           fmt.Sprintf("test-prototype-%s", NormalizePackageName(info.Name)),
		Version:        "1.0.0",
		PackageName:    info.Name,
		PackageVersion: info.Version,
		ModuleType:     g.detector.GetPackageJSONType(info),
		OutputDir:      outputDir,
	}
	pkgJSON := PackageJSON{
		Name:         data.Name,
		Version:      data.Version,
		Description:  fmt.Sprintf("Prototype pollution test for %s@%s", info.Name, info.Version),
		Private:      true,
		Type:         data.ModuleType,
		Dependencies: map[string]string{info.Name: info.Version},
	}
	return g.generateTestPackage("prototype-test", data, outputDir, pkgJSON)
}

func (g *Generator) generateCLITest(info *PackageInfo, outputDir string) error {
	if err := os.MkdirAll(outputDir, 0o755); err != nil {
		return fmt.Errorf("failed to create CLI marker directory: %w", err)
	}
	var firstBinName string
	for name := range info.Bin {
		firstBinName = name
		break
	}
	markerContent := fmt.Sprintf("# CLI Test Marker\nPackage: %s@%s\nBinary: %s\n\nCLI test runs via: npx %s\n", info.Name, info.Version, firstBinName, info.Name)
	markerPath := filepath.Join(outputDir, "HAS_CLI")
	if err := os.WriteFile(markerPath, []byte(markerContent), 0o644); err != nil {
		return fmt.Errorf("failed to write CLI marker: %w", err)
	}
	return nil
}

func (g *Generator) generateTestPackage(templateName string, data TestPackage, outputDir string, pkgJSON PackageJSON) error {
	if err := os.MkdirAll(outputDir, 0o755); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}
	pkgJSONPath := filepath.Join(outputDir, "package.json")
	pkgJSONData, err := json.MarshalIndent(pkgJSON, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal package.json: %w", err)
	}
	if err := os.WriteFile(pkgJSONPath, pkgJSONData, 0o644); err != nil {
		return fmt.Errorf("failed to write package.json: %w", err)
	}

	templateDir := filepath.Join(g.templatesDir, templateName)
	entries, err := os.ReadDir(templateDir)
	if err != nil {
		return fmt.Errorf("failed to read template directory: %w", err)
	}

	for _, entry := range entries {
		if entry.Name() == "package.json" {
			continue
		}
		srcPath := filepath.Join(templateDir, entry.Name())
		dstPath := filepath.Join(outputDir, entry.Name())
		if entry.IsDir() {
			if err := g.copyDir(srcPath, dstPath, data); err != nil {
				return fmt.Errorf("failed to copy directory %s: %w", entry.Name(), err)
			}
		} else {
			if err := g.processTemplateFile(srcPath, dstPath, data); err != nil {
				return fmt.Errorf("failed to process template %s: %w", entry.Name(), err)
			}
		}
	}

	return nil
}

func (g *Generator) processTemplateFile(srcPath, dstPath string, data TestPackage) error {
	content, err := os.ReadFile(srcPath)
	if err != nil {
		return fmt.Errorf("template %s: failed to read: %w", srcPath, err)
	}
	tmpl, err := template.New(filepath.Base(srcPath)).Parse(string(content))
	if err != nil {
		return fmt.Errorf("template %s: failed to parse: %w", srcPath, err)
	}
	var buf strings.Builder
	if err := tmpl.Execute(&buf, data); err != nil {
		return fmt.Errorf("template %s: failed to execute: %w", srcPath, err)
	}
	if err := os.WriteFile(dstPath, []byte(buf.String()), 0o644); err != nil {
		return fmt.Errorf("template %s: failed to write: %w", dstPath, err)
	}
	return nil
}

func (g *Generator) copyDir(srcPath, dstPath string, data TestPackage) error {
	if err := os.MkdirAll(dstPath, 0o755); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}
	entries, err := os.ReadDir(srcPath)
	if err != nil {
		return fmt.Errorf("failed to read directory: %w", err)
	}
	for _, entry := range entries {
		srcChild := filepath.Join(srcPath, entry.Name())
		dstChild := filepath.Join(dstPath, entry.Name())
		if entry.IsDir() {
			if err := g.copyDir(srcChild, dstChild, data); err != nil {
				return err
			}
		} else {
			if err := g.processTemplateFile(srcChild, dstChild, data); err != nil {
				return err
			}
		}
	}
	return nil
}
