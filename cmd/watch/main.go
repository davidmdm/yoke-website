package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"sync"

	"github.com/davidmdm/ansi"
	"github.com/fsnotify/fsnotify"
)

var (
	cyan   = ansi.MakeStyle(ansi.FgCyan).Sprint
	yellow = ansi.MakeStyle(ansi.FgYellow)
)

var debug = func(format string, args ...any) {
}

func init() {
	if ok, _ := strconv.ParseBool(os.Getenv("DEBUG")); ok {
		debug = yellow.Printf
	}
}

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run() error {
	flagDir := flag.String("d", ".", "directories to watch")
	flagExt := flag.String("e", "", "extensions to watch")
	command := flag.String("x", "", "command to run")

	flag.Parse()

	dirs := strings.Split(*flagDir, ",")

	exts := strings.Split(*flagExt, ",")

	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		return fmt.Errorf("failed to create watcher: %w", err)
	}
	defer watcher.Close()

	for _, dir := range dirs {
		for _, item := range GetRecursiveDirs(dir) {
			debug("adding dir:  %s\n", item)
			if err := watcher.Add(item); err != nil {
				return fmt.Errorf("failed to add watcher %s: %w", dir, err)
			}
		}
	}

	// Send empty event to trigger initial build on startup.

	wg := new(sync.WaitGroup)

	launch := func() context.CancelFunc {
		wg.Add(1)

		ctx, cancel := context.WithCancel(context.Background())

		go func() {
			defer wg.Done()

			cmd := WithStdPipes(exec.CommandContext(ctx, "bash", "-c", *command))
			fmt.Printf("\nrunning: %s\n\n", cyan(*command))
			if err := cmd.Run(); err != nil {
				fmt.Println(err)
			}
			debug("exiting current process\n")
		}()

		return cancel
	}

	cancel := launch()

	for {
		select {
		case err := <-watcher.Errors:
			fmt.Println(err)
			continue
		case evt := <-watcher.Events:
			if evt.Op == fsnotify.Chmod {
				continue
			}

			debug("recieved event: %s %s\n", evt.Name, evt.Op)

			ext := strings.TrimLeft(filepath.Ext(evt.Name), ".")
			if len(exts) > 0 && !slices.Contains(exts, ext) {
				continue
			}
			if evt.Op == fsnotify.Remove {
				if err := watcher.Remove(evt.Name); err != nil {
					debug("failed to remove %s: %v\n", evt.Name, err)
				}
				continue
			}
			if evt.Op == fsnotify.Create {
				if err := watcher.Add(evt.Name); err != nil {
					debug("failed to add %s: %v\n", evt.Name, err)
				}
				continue
			}

			cancel()
			wg.Wait()

			cancel = launch()
		}
	}
}

func GetRecursiveDirs(path string) []string {
	entries, err := os.ReadDir(path)
	if err != nil {
		panic(err)
	}

	var result []string
	result = append(result, path)

	for _, entry := range entries {
		if entry.IsDir() {
			base := entry.Name()
			if base == "node_modules" || base == ".vscode" || base == ".git" {
				continue
			}
			result = append(result, GetRecursiveDirs(filepath.Join(path, base))...)
		}
	}

	return result
}

func WithStdPipes(cmd *exec.Cmd) *exec.Cmd {
	cmd.Stdin = os.Stdin
	cmd.Stderr = os.Stderr
	cmd.Stdout = os.Stdout
	return cmd
}
