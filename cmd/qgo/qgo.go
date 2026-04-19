package main

import (
	"fmt"
	"io"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

type ModQgo struct {
	// 1. Parse and record argument state
	ParseArgs func()
	// 2. Initialize environment
	SetupTempWorkspace func()
	// 3. Move source code
	SyncSource func()
	// 4. Run go mod
	PrepareDependencies func()
	// 5. Compile
	Compile func()
	// 6. Final execution
	Execute func()
	// Helper: cleanup all temp folder
	Cleanup func()
}

func NewModQgo() *ModQgo {
	var goFile string
	var goFileIdx int
	var tempDir string
	var targetPath string
	var binaryPath string
	finished := 0
	total := 6

	// Helper: run a command inside the temp directory
	runInTemp := func(name string, args ...string) {
		cmd := exec.Command(name, args...)
		cmd.Dir = tempDir
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		if err := cmd.Run(); err != nil {
			panic(fmt.Sprintf("command failed: %s %s", name, strings.Join(args, " ")))
		}
	}

	cleanup := func() {
		err := os.RemoveAll(tempDir)
		if err != nil {
			panic(err)
		}
	}

	parseArgs := func() {
		log.Printf("[%d/%d] qgo: Parsing args", finished+1, total)
		if len(os.Args) < 2 || (len(os.Args) >= 2 && os.Args[1] != "run") {
			panic("Usage: qgo run [build flags] <file.go> [arguments...]")
		}
		for i, arg := range os.Args[2:] {
			if strings.HasSuffix(arg, ".go") {
				goFile = arg
				goFileIdx = i + 2
				break
			}
		}
		if goFile == "" {
			panic("qgo: no .go files specified")
		}
		finished += 1
	}

	setupTempWorkspace := func() {
		log.Printf("[%d/%d] qgo: Setting up temp workspace", finished+1, total)
		var err error
		tempDir, err = os.MkdirTemp("", "qgo-*")
		if err != nil {
			panic(err)
		}
		finished += 1
	}

	syncSource := func() {
		log.Printf("[%d/%d] qgo: Syncing source", finished+1, total)
		targetPath = filepath.Join(tempDir, filepath.Base(goFile))
		source, err := os.Open(goFile)
		if err != nil {
			panic(err)
		}
		defer source.Close()
		destination, err := os.Create(targetPath)
		if err != nil {
			panic(err)
		}
		defer destination.Close()
		if _, err := io.Copy(destination, source); err != nil {
			panic(err)
		}
		finished += 1
	}

	prepareDependencies := func() {
		log.Printf("[%d/%d] qgo: Preparing dependencies", finished+1, total)
		runInTemp("go", "mod", "init", "qgo/runtime")
		log.Printf(">> go mod resolving dependencies...")
		runInTemp("go", "mod", "tidy")
		finished += 1
	}

	compile := func() {
		log.Printf("[%d/%d] qgo: Compiling", finished+1, total)
		binaryName := "qgo_bin"
		if filepath.Base(os.Args[0]) == "qgo_bin" {
			binaryName = "qgo_bin_exec"
		}
		binaryPath = filepath.Join(tempDir, binaryName)
		args := []string{"build", "-o", binaryPath}
		args = append(args, os.Args[2:goFileIdx]...)
		args = append(args, filepath.Base(goFile))
		runInTemp("go", args...)
		finished += 1
	}

	execute := func() {
		log.Printf("[%d/%d] qgo: Executing", finished+1, total)
		appArgs := os.Args[goFileIdx+1:]
		cmd := exec.Command(binaryPath, appArgs...)
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		cmd.Stdin = os.Stdin
		if err := cmd.Run(); err != nil {
			panic(err)
		}
		finished += 1
	}

	return &ModQgo{
		ParseArgs:           parseArgs,
		SetupTempWorkspace:  setupTempWorkspace,
		SyncSource:          syncSource,
		PrepareDependencies: prepareDependencies,
		Compile:             compile,
		Execute:             execute,
		Cleanup:             cleanup,
	}
}
