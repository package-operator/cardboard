package dev

import (
	"bytes"
	"fmt"
	"io/fs"
	"os"
	"path"
	"sort"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"sigs.k8s.io/yaml"
)

// Loads kubernets objects from all .yaml files in the given folder.
// Does not recurse into subfolders.
// Preserves lexical file order.
func LoadKubernetesObjectsFromFolder(folderPath string) ([]unstructured.Unstructured, error) {
	folder, err := os.Open(folderPath)
	if err != nil {
		return nil, fmt.Errorf("open %q: %w", folderPath, err)
	}
	defer folder.Close()

	files, err := folder.Readdir(-1)
	if err != nil {
		return nil, fmt.Errorf("read directory: %w", err)
	}
	sort.Sort(fileInfosByName(files))

	var objects []unstructured.Unstructured
	for _, file := range files {
		if file.IsDir() {
			continue
		}

		if path.Ext(file.Name()) != ".yaml" {
			continue
		}

		objs, err := LoadKubernetesObjectsFromFile(path.Join(folderPath, file.Name()))
		if err != nil {
			return nil, fmt.Errorf("loading kubernetes objects from file %q: %w", file, err)
		}
		objects = append(objects, objs...)
	}
	return objects, nil
}

// Loads kubernetes objects from the given file.
func LoadKubernetesObjectsFromFile(filePath string) ([]unstructured.Unstructured, error) {
	fileYaml, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("reading %s: %w", filePath, err)
	}

	return LoadKubernetesObjectsFromBytes(fileYaml)
}

// Loads kubernetes objects from given bytes.
// A single file may contain multiple objects separated by "---\n".
func LoadKubernetesObjectsFromBytes(fileYaml []byte) ([]unstructured.Unstructured, error) {
	// Trim empty starting and ending objects
	fileYaml = bytes.Trim(fileYaml, "-\n")

	var objects []unstructured.Unstructured
	// Split for every included yaml document.
	for i, yamlDocument := range bytes.Split(fileYaml, []byte("---\n")) {
		obj := unstructured.Unstructured{}
		if err := yaml.Unmarshal(yamlDocument, &obj); err != nil {
			return nil, fmt.Errorf(
				"unmarshalling yaml document at index %d: %w", i, err)
		}

		objects = append(objects, obj)
	}

	return objects, nil
}

// Sorts fs.FileInfo objects by basename.
type fileInfosByName []fs.FileInfo

func (x fileInfosByName) Len() int { return len(x) }

func (x fileInfosByName) Less(i, j int) bool {
	iName := path.Base(x[i].Name())
	jName := path.Base(x[j].Name())
	return iName < jName
}

func (x fileInfosByName) Swap(i, j int) { x[i], x[j] = x[j], x[i] }
