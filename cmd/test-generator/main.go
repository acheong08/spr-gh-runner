package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"

	"git.duti.dev/secure-package-registry/spr-gh-runner/pkg/tester"
)

func main() {
	packageName := flag.String("package", "", "Package name")
	version := flag.String("version", "", "Package version")
	outputDir := flag.String("output", "./test-pkg", "Output directory")
	registryURL := flag.String("registry-url", "", "Registry URL")
	registryType := flag.String("registry-type", "", "Registry type (npm or gitea)")
	registryOwner := flag.String("registry-owner", "", "Registry owner for Gitea registries")
	templatesDir := flag.String("templates-dir", filepath.Join(".", "templates"), "Templates directory")
	flag.Parse()

	if *packageName == "" || *version == "" {
		fmt.Fprintln(os.Stderr, "--package and --version are required")
		os.Exit(1)
	}
	if *registryURL == "" {
		fmt.Fprintln(os.Stderr, "--registry-url is required")
		os.Exit(1)
	}
	if *registryType == "" {
		fmt.Fprintln(os.Stderr, "--registry-type is required")
		os.Exit(1)
	}

	var registry tester.RegistryConfig
	switch *registryType {
	case "npm":
		registry = tester.NewNPMRegistryConfig(*registryURL)
	case "gitea":
		if *registryOwner == "" {
			fmt.Fprintln(os.Stderr, "--registry-owner is required for gitea registries")
			os.Exit(1)
		}
		registry = tester.NewGiteaRegistryConfig(*registryURL, *registryOwner)
	default:
		fmt.Fprintf(os.Stderr, "unsupported --registry-type %q\n", *registryType)
		os.Exit(1)
	}

	generator := tester.NewGenerator(*templatesDir, registry)
	generated, err := generator.GenerateAll(*packageName, *version, *outputDir)
	if err != nil {
		fmt.Fprintf(os.Stderr, "generate tests: %v\n", err)
		os.Exit(1)
	}

	for _, dir := range generated {
		fmt.Println(dir)
	}
}
