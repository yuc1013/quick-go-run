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

// Global state to avoid passing data through many function calls
var state = struct {
	goFile     string
	goFileIdx  int
	tempDir    string
	targetPath string
	binaryPath string
}{}

func main() {
	// Override default panic behavior
	defer func() {
		if r := recover(); r != nil {
			log.Fatal(r)
		}
	}()

	// Trigger the logic chain in order
	parseArgs()
	setupTempWorkspace()
	defer os.RemoveAll(state.tempDir) // Clean up at the end
	syncSource()
	prepareDependencies()
	compile()
	execute()
}

// 1. Parse and record argument state
func parseArgs() {
	if len(os.Args) < 2 || (len(os.Args) >= 2 && os.Args[1] != "run") {
		panic("Usage: qgo run [build flags] <file.go> [arguments...]")
	}

	for i, arg := range os.Args[2:] {
		if strings.HasSuffix(arg, ".go") {
			state.goFile = arg
			state.goFileIdx = i + 2
			break
		}
	}
	if state.goFile == "" {
		log.Printf("qgo: no .go files specified")
		panic("")
	}
}

// 2. Initialize environment
func setupTempWorkspace() {
	var err error
	state.tempDir, err = os.MkdirTemp("", "qgo-*")
	if err != nil {
		panic(err)
	}
	// Note: because this is global state, we clean it up at the end of main
	// In real-world code, you might also call a cleanup function after execute
}

// 3. Move source code
func syncSource() {
	state.targetPath = filepath.Join(state.tempDir, filepath.Base(state.goFile))

	source, err := os.Open(state.goFile)
	if err != nil {
		panic(err)
	}
	defer source.Close()

	destination, err := os.Create(state.targetPath)
	if err != nil {
		panic(err)
	}
	defer destination.Close()

	if _, err := io.Copy(destination, source); err != nil {
		panic(err)
	}
}

// 4. Run go mod
func prepareDependencies() {
	runInTemp("go", "mod", "init", "qgo/runtime")
	log.Printf("qgo: >> resolving dependencies...")
	runInTemp("go", "mod", "tidy")
}

// 5. Compile
func compile() {
	binaryName := "qgo_bin"
	if filepath.Base(os.Args[0]) == "qgo_bin" {
		binaryName = "qgo_bin_exec"
	}
	state.binaryPath = filepath.Join(state.tempDir, binaryName)

	args := []string{"build", "-o", state.binaryPath}
	args = append(args, os.Args[2:state.goFileIdx]...)
	args = append(args, filepath.Base(state.goFile))

	runInTemp("go", args...)
}

// 6. Final execution
func execute() {
	appArgs := os.Args[state.goFileIdx+1:]
	cmd := exec.Command(state.binaryPath, appArgs...)
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

// Helper: run a command inside the temp directory
func runInTemp(name string, args ...string) {
	cmd := exec.Command(name, args...)
	cmd.Dir = state.tempDir
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		panic(fmt.Sprintf("command failed: %s %s", name, strings.Join(args, " ")))
	}
}
