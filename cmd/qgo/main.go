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

	pl := NewPipeline()

	// Trigger the logic chain in order
	pl.ParseArgs()
	pl.SetupTempWorkspace()
	defer os.RemoveAll(pl.tempDir) // Clean up at the end
	pl.SyncSource()
	pl.PrepareDependencies()
	pl.Compile()
	pl.Execute()
}

// Pipeline process args to run the file
type Pipeline struct {
	goFile     string
	goFileIdx  int
	tempDir    string
	targetPath string
	binaryPath string

	totalN    int
	finishedN int
}

func NewPipeline() *Pipeline {
	return &Pipeline{
		totalN: 6,
	}
}

// 1. Parse and record argument state
func (pl *Pipeline) ParseArgs() {
	// TODO: 打日志
	log.Printf("[%d/%d] qgo: Parsing args", pl.finishedN+1, pl.totalN)

	if len(os.Args) < 2 || (len(os.Args) >= 2 && os.Args[1] != "run") {
		panic("Usage: qgo run [build flags] <file.go> [arguments...]")
	}

	for i, arg := range os.Args[2:] {
		if strings.HasSuffix(arg, ".go") {
			pl.goFile = arg
			pl.goFileIdx = i + 2
			break
		}
	}
	if pl.goFile == "" {
		log.Printf("qgo: no .go file specified")
		panic("")
	}

	pl.finishedN += 1
}

// 2. Initialize environment
func (pl *Pipeline) SetupTempWorkspace() {
	log.Printf("[%d/%d] qgo: Setting up temp workspace", pl.finishedN+1, pl.totalN)

	var err error
	pl.tempDir, err = os.MkdirTemp("", "qgo-*")
	if err != nil {
		panic(err)
	}
	// Note: because this is global pl. we clean it up at the end of main
	// In real-world code, you might also call a cleanup function after execute

	pl.finishedN += 1
}

// 3. Move source code
func (pl *Pipeline) SyncSource() {
	log.Printf("[%d/%d] qgo: Syncing source", pl.finishedN+1, pl.totalN)

	pl.targetPath = filepath.Join(pl.tempDir, filepath.Base(pl.goFile))

	source, err := os.Open(pl.goFile)
	if err != nil {
		panic(err)
	}
	defer source.Close()

	destination, err := os.Create(pl.targetPath)
	if err != nil {
		panic(err)
	}
	defer destination.Close()

	if _, err := io.Copy(destination, source); err != nil {
		panic(err)
	}

	pl.finishedN += 1
}

// 4. Run go mod
func (pl *Pipeline) PrepareDependencies() {
	log.Printf("[%d/%d] qgo: Preparing dependencies", pl.finishedN+1, pl.totalN)

	pl.runInTemp("go", "mod", "init", "qgo/runtime")
	log.Printf(">> go mod resolving dependencies...")
	pl.runInTemp("go", "mod", "tidy")

	pl.finishedN += 1
}

// 5. Compile
func (pl *Pipeline) Compile() {
	log.Printf("[%d/%d] qgo: Compiling", pl.finishedN+1, pl.totalN)

	binaryName := "qgo_bin"
	if filepath.Base(os.Args[0]) == "qgo_bin" {
		binaryName = "qgo_bin_exec"
	}
	pl.binaryPath = filepath.Join(pl.tempDir, binaryName)

	args := []string{"build", "-o", pl.binaryPath}
	args = append(args, os.Args[2:pl.goFileIdx]...)
	args = append(args, filepath.Base(pl.goFile))

	pl.runInTemp("go", args...)

	pl.finishedN += 1
}

// 6. Final execution
func (pl *Pipeline) Execute() {
	log.Printf("[%d/%d] qgo: Executing", pl.finishedN+1, pl.totalN)

	appArgs := os.Args[pl.goFileIdx+1:]
	cmd := exec.Command(pl.binaryPath, appArgs...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Stdin = os.Stdin

	if err := cmd.Run(); err != nil {
		// if exitErr, ok := err.(*exec.ExitError); ok {
		// 	os.Exit(exitErr.ExitCode())
		// }
		panic(err)
	}

	pl.finishedN += 1
}

// Helper: run a command inside the temp directory
func (pl *Pipeline) runInTemp(name string, args ...string) {
	cmd := exec.Command(name, args...)
	cmd.Dir = pl.tempDir
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		panic(fmt.Sprintf("command failed: %s %s", name, strings.Join(args, " ")))
	}
}
