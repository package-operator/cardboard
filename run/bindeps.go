package run

import (
	"context"
	"fmt"
	"os"
	"path"
	"path/filepath"

	"pkg.package-operator.run/cardboard/sh"
)

// Manages binary dependencies in a project-local folder.
type dependencyManager struct {
	path   string
	runner *sh.Runner
	dr     *dependencyRun
	depFns []Dependency
	deps   map[string]string
}

var _ Dependency = (*dependencyManager)(nil)

const (
	defaultCacheDirectory = ".cache"
)

func newDependencyManager(dr *dependencyRun) *dependencyManager {
	absCacheDir, err := filepath.Abs(defaultCacheDirectory)
	if err != nil {
		panic(err)
	}

	path := filepath.Join(absCacheDir, "deps")
	dm := &dependencyManager{
		path: path,
		dr:   dr,
		deps: map[string]string{},
	}
	dm.runner = sh.New(sh.WithEnvironment{"GOBIN": dm.Bin()})
	return dm
}

func (d *dependencyManager) ID() string {
	return fmt.Sprintf("pkg.package-operator.run/cardboard/run.dependencyManager{path:%s}.Run()", defaultCacheDirectory)
}

func (d *dependencyManager) Run(ctx context.Context) error {
	return d.dr.Parallel(ctx, DependencyID(d.ID()), d.depFns...)
}

func (d *dependencyManager) IsEmpty() bool {
	return len(d.depFns) == 0
}

// Returns the /bin directory containing the dependency binaries.
func (d *dependencyManager) Bin() string {
	return path.Join(d.path, "bin")
}

// Register a new dependency to be installed.
func (d *dependencyManager) Register(tool, packageURL, version string) error {
	newURL := depURL(packageURL, version)
	if url, ok := d.deps[tool]; ok && newURL != url {
		return fmt.Errorf("conflicting dependency for %s, already have: %s", tool, url)
	}

	installFn := func() error {
		return d.goInstall(tool, packageURL, version)
	}

	d.deps[tool] = newURL
	d.depFns = append(d.depFns, FnWithName(fmt.Sprintf("go install %s", tool), installFn))
	return nil
}

// go install a dependency into the dependency directory.
func (d *dependencyManager) goInstall(tool, packageURL, version string) error {
	if err := os.MkdirAll(d.path, os.ModePerm); err != nil {
		return fmt.Errorf("create dependency dir: %w", err)
	}

	needsRebuild, err := d.NeedsRebuild(tool, version)
	if err != nil {
		return err
	}
	if !needsRebuild {
		return nil
	}

	url := packageURL + "@v" + version
	if err := d.runner.Run("go", "install", url); err != nil {
		return fmt.Errorf("install %s: %w", url, err)
	}
	return nil
}

// Checks if a tool in the dependency directory needs to be rebuild.
func (d *dependencyManager) NeedsRebuild(tool, version string) (needsRebuild bool, err error) {
	versionFile := path.Join(d.path, "versions", tool, "v"+version)
	if err := ensureFile(versionFile); err != nil {
		return false, fmt.Errorf("ensure file: %w", err)
	}

	// Checks "tool" binary file modification date against version file.
	// If the version file is newer, tool is of the wrong version.
	rebuild, err := Path(path.Join(d.Bin(), tool), versionFile)
	if err != nil {
		return false, fmt.Errorf("rebuild check: %w", err)
	}

	return rebuild, nil
}

func depURL(packageURL, version string) string {
	return packageURL + "@v" + version
}

// ensure a file and it's file path exist.
func ensureFile(file string) error {
	dir := filepath.Dir(file)
	if err := os.MkdirAll(dir, os.ModePerm); err != nil {
		return fmt.Errorf("creating directory %s: %w", dir, err)
	}

	_, err := os.Stat(file)
	if os.IsNotExist(err) {
		f, err := os.Create(file)
		if err != nil {
			return fmt.Errorf("creating file %s: %w", file, err)
		}
		defer f.Close()
		return nil
	}
	if err != nil {
		return fmt.Errorf("checking file %s: %w", file, err)
	}
	return nil
}
