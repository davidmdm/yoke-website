package main

import (
	"bytes"
	"flag"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"github.com/davidmdm/yoke-website/internal"
	"github.com/yosssi/gohtml"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintf(os.Stderr, "%v\n", err)
		os.Exit(1)
	}
}

func run() error {
	contentDir := flag.String("content", "./content", "path to content directory")
	outputDir := flag.String("output", "./cmd/server/content", "path to output directory")

	flag.Parse()

	*contentDir = filepath.Clean(*contentDir)
	*outputDir = filepath.Clean(*outputDir)

	if err := os.RemoveAll(*outputDir); err != nil {
		return err
	}

	var baseTrie internal.PrefixTree[[]byte]

	var copyFunc fs.WalkDirFunc = func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		path = filepath.Clean(path)

		outputPath := func() string {
			if path == *contentDir {
				return *outputDir
			}
			return filepath.Join(*outputDir, strings.TrimPrefix(path, *contentDir))
		}()

		if d.IsDir() {
			return os.MkdirAll(outputPath, 0o755)
		}

		if d.Name() == "_base.html" {
			baseHTML, err := os.ReadFile(path)
			if err != nil {
				return err
			}

			dirPath := filepath.Dir(path)
			rootHTML := baseTrie.GetLongestSubmatch(dirPath)

			merged := func() []byte {
				if len(rootHTML) == 0 {
					return baseHTML
				}
				baseHTML = bytes.Replace(rootHTML, []byte("<!-- REPLACE ME -->"), baseHTML, 1)
				return gohtml.FormatBytes(baseHTML)
			}()

			baseTrie.Insert(dirPath, merged)

			return nil
		}

		if strings.HasPrefix(d.Name(), "_") {
			return nil
		}

		data, err := os.ReadFile(path)
		if err != nil {
			return fmt.Errorf("failed to read file: %v", err)
		}

		if filepath.Ext(d.Name()) == ".html" {
			baseHTML := baseTrie.GetLongestSubmatch(path)
			if len(baseHTML) > 0 {
				data = bytes.Replace(baseHTML, []byte("<!-- REPLACE ME -->"), data, 1)
				data = gohtml.FormatBytes(data)
			}
		}

		if err := os.WriteFile(outputPath, data, 0o644); err != nil {
			return fmt.Errorf("failed to write file: %v", err)
		}

		return nil
	}

	if err := filepath.WalkDir(*contentDir, copyFunc); err != nil {
		return fmt.Errorf("failed to copy %s -> %s : %v", *contentDir, *outputDir, err)
	}

	return nil
}
