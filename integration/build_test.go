package integration

import (
	"fmt"
	"github.com/magefile/mage/mg"
	"github.com/magefile/mage/sh"
	"github.com/mt-sre/devkube/dev"
	"github.com/stretchr/testify/assert"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"testing"
)

var (
	projectRoot string
	cacheDir    string
	testDataDir string
	runtime     dev.ContainerRuntime
)

func init() {
	dir, err := filepath.Abs("..")
	if err != nil {
		panic(err)
	}
	projectRoot = dir
	cacheDir = filepath.Join(projectRoot, ".cache")
	testDataDir = filepath.Join(projectRoot, "integration/test-data")

	runtime, err = dev.DetectContainerRuntime()
	if err != nil {
		panic(err)
	}
}

func buildBinary() error {
	args := []string{"build", filepath.Join(testDataDir, "test-stub/main.go")}
	cmd := exec.Command("go", args...)
	cmd.Dir = testDataDir
	return cmd.Run()
}

func cleanCache(cache string) error {
	if err := os.RemoveAll(cache); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("deleting cache: %w", err)
	}
	if err := os.Remove(cache + ".tar"); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("deleting image cache: %w", err)
	}
	if err := os.MkdirAll(cache, os.ModePerm); err != nil {
		return fmt.Errorf("creating cache dir: %w", err)
	}
	return nil
}

func populateCache(cache string, files ...string) error {
	for _, f := range files {
		if err := sh.Copy(filepath.Join(cache, filepath.Base(f)), f); err != nil {
			return fmt.Errorf("copying %s: %w", f, err)
		}
	}
	return nil
}

func detectImage(tagPattern string) (bool, error) {
	out, err := exec.Command(string(runtime), "images").Output() //nolint:gosec
	if err != nil {
		return false, err
	}
	return regexp.Match(tagPattern, out)
}

func TestBuildImage(t *testing.T) {
	cache := filepath.Join(cacheDir, "test-stub")

	deps := []interface{}{
		mg.F(buildBinary),
		mg.F(cleanCache, cache),
		mg.F(populateCache, cache,
			filepath.Join(testDataDir, "main"),
			filepath.Join(testDataDir, "passwd"),
			filepath.Join(testDataDir, "test-stub.Containerfile")),
	}

	buildInfo := dev.ImageBuildInfo{
		ImageTag:      "test-stub",
		CacheDir:      cache,
		ContainerFile: "test-stub.Containerfile",
		ContextDir:    ".",
		Runtime:       string(runtime),
	}

	assert.NoError(t, dev.BuildImage(&buildInfo, deps))

	match, err := detectImage("/test-stub")
	assert.NoError(t, err)
	assert.True(t, match)
}

func TestBuildPackage(t *testing.T) {
	cache := filepath.Join(cacheDir, "test-stub-package")
	testPackageDir := filepath.Join(testDataDir, "test-stub-package")

	deps := []interface{}{
		mg.F(cleanCache, cache),
		mg.F(populateCache, cache,
			filepath.Join(testPackageDir, "manifest.yaml"),
			filepath.Join(testPackageDir, "deployment.yaml.gotmpl"),
			filepath.Join(testPackageDir, "namespace.template.yaml.gotmpl")),
	}

	buildInfo := dev.PackageBuildInfo{
		ImageTag:   "test-stub-package",
		CacheDir:   cache,
		SourcePath: cache,
		OutputPath: cache + ".tar",
		Runtime:    string(runtime),
	}

	assert.NoError(t, dev.BuildPackage(&buildInfo, deps))

	match, err := detectImage("/test-stub-package")
	assert.NoError(t, err)
	assert.True(t, match)
}
