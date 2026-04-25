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

func main() {
	// Override default panic behavior
	defer func() {
		if r := recover(); r != nil {
			log.Fatal(r)
		}
	}()

	i := 0
	n := 6

	stepLog := func(info string) {
		log.Printf("[%d/%d] qgo: %s", i+1, n, info)
	}

	stepLog("Parsing args")
	parseArgs()
	i++

	stepLog("Setting up temp workspace")
	setupTempWorkspace()
	i++

	defer os.RemoveAll(tempDir)

	stepLog("Syncing source")
	syncSource()
	i++

	stepLog("Preparing dependencies")
	prepareDependencies()
	i++

	stepLog("Compiling")
	compile()
	i++

	stepLog("Executing")
	execute()
	i++
}

var (
	goFiles         []string
	goFilesStartIdx int
	goFilesEndIdx   int
	tempDir         string
	targetPaths     []string
	binaryPath      string
)

// step: parse command
func parseArgs() {
	if len(os.Args) < 2 || os.Args[1] != "run" {
		panic("Usage: qgo run [build flags] <file.go>(s) [arguments...]")
	}

	// get the first gofile index
	for i, arg := range os.Args[2:] {
		if strings.HasSuffix(arg, ".go") {
			goFilesStartIdx = i + 2
			break
		}
	}

	// get the last gofile index
	for i, arg := range os.Args[goFilesStartIdx:] {
		if strings.HasSuffix(arg, ".go") {
			goFilesEndIdx = i + goFilesStartIdx
		} else {
			break
		}
	}

	goFiles = append(goFiles, os.Args[goFilesStartIdx:goFilesEndIdx+1]...)

	if len(goFiles) == 0 {
		panic("qgo: no .go files specified")
	}
}

// step: create tempdir
func setupTempWorkspace() {
	var err error
	tempDir, err = os.MkdirTemp("", "qgo-*")
	if err != nil {
		panic(err)
	}
	// Note: because this is global pl. we clean it up at the end of main
	// In real-world code, you might also call a cleanup function after execute
}

// step: copy code
func syncSource() {
	for _, goFile := range goFiles {
		targetPath := filepath.Join(tempDir, filepath.Base(goFile))
		targetPaths = append(targetPaths, targetPath)

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
	}
}

// step: pull dependencies
func prepareDependencies() {
	runInTemp("go", "mod", "init", "qgo/runtime")
	log.Printf(">> go mod resolving dependencies...")
	runInTemp("go", "mod", "tidy")
}

// 5. compile
func compile() {
	binaryName := "qgo_bin"
	if filepath.Base(os.Args[0]) == "qgo_bin" {
		binaryName = "qgo_bin_exec"
	}
	binaryPath = filepath.Join(tempDir, binaryName)

	args := []string{"build", "-o", binaryPath}
	args = append(args, os.Args[2:goFilesStartIdx]...)
	for _, goFile := range goFiles {
		args = append(args, filepath.Base(goFile))
	}

	runInTemp("go", args...)
}

// 6. Final execution
func execute() {
	appArgs := os.Args[goFilesEndIdx+1:]
	cmd := exec.Command(binaryPath, appArgs...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Stdin = os.Stdin

	if err := cmd.Run(); err != nil {
		// if exitErr, ok := err.(*exec.ExitError); ok {
		// 	os.Exit(exitErr.ExitCode())
		// }
		panic(err)
	}
}

// helper: run a command inside the temp directory
func runInTemp(name string, args ...string) {
	cmd := exec.Command(name, args...)
	cmd.Dir = tempDir
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		panic(fmt.Sprintf("command failed: %s %s", name, strings.Join(args, " ")))
	}
}
