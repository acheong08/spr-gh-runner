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
	registryURL := flag.String("registry-url", "https://registry.npmjs.org", "Registry URL")
	registryOwner := flag.String("registry-owner", "", "Registry owner for private registries")
	registryToken := flag.String("registry-token", "", "Registry bearer token")
	templatesDir := flag.String("templates-dir", filepath.Join(".", "templates"), "Templates directory")
	flag.Parse()

	if *packageName == "" || *version == "" {
		fmt.Fprintln(os.Stderr, "--package and --version are required")
		os.Exit(1)
	}

	generator := tester.NewGeneratorWithRegistry(*templatesDir, *registryURL, *registryOwner, *registryToken)
	generated, err := generator.GenerateAll(*packageName, *version, *outputDir)
	if err != nil {
		fmt.Fprintf(os.Stderr, "generate tests: %v\n", err)
		os.Exit(1)
	}

	for _, dir := range generated {
		fmt.Println(dir)
	}
}
