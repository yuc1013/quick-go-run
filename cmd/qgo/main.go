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

// EnvQgo process args to run the file
type EnvQgo struct {
	goFile     string
	goFileIdx  int
	tempDir    string
	targetPath string
	binaryPath string

	totalN    int
	finishedN int
}

func NewEnvQgo() *EnvQgo {
	return &EnvQgo{
		finishedN: 0,
		totalN:    6,
	}
}

// 1. Parse and record argument state
func ParseArgs(env *EnvQgo) {
	log.Printf("[%d/%d] qgo: Parsing args", env.finishedN+1, env.totalN)

	if len(os.Args) < 2 || (len(os.Args) >= 2 && os.Args[1] != "run") {
		panic("Usage: qgo run [build flags] <file.go> [arguments...]")
	}

	for i, arg := range os.Args[2:] {
		if strings.HasSuffix(arg, ".go") {
			env.goFile = arg
			env.goFileIdx = i + 2
			break
		}
	}
	if env.goFile == "" {
		panic("qgo: no .go files specified")
	}

	env.finishedN += 1
}

// 2. Initialize environment
func SetupTempWorkspace(env *EnvQgo) {
	log.Printf("[%d/%d] qgo: Setting up temp workspace", env.finishedN+1, env.totalN)

	var err error
	env.tempDir, err = os.MkdirTemp("", "qgo-*")
	if err != nil {
		panic(err)
	}
	// Note: because this is global pl. we clean it up at the end of main
	// In real-world code, you might also call a cleanup function after execute

	env.finishedN += 1
}

// 3. Move source code
func SyncSource(env *EnvQgo) {
	log.Printf("[%d/%d] qgo: Syncing source", env.finishedN+1, env.totalN)

	env.targetPath = filepath.Join(env.tempDir, filepath.Base(env.goFile))

	source, err := os.Open(env.goFile)
	if err != nil {
		panic(err)
	}
	defer source.Close()

	destination, err := os.Create(env.targetPath)
	if err != nil {
		panic(err)
	}
	defer destination.Close()

	if _, err := io.Copy(destination, source); err != nil {
		panic(err)
	}

	env.finishedN += 1
}

// 4. Run go mod
func PrepareDependencies(env *EnvQgo) {
	log.Printf("[%d/%d] qgo: Preparing dependencies", env.finishedN+1, env.totalN)

	env.runInTemp("go", "mod", "init", "qgo/runtime")
	log.Printf(">> go mod resolving dependencies...")
	env.runInTemp("go", "mod", "tidy")

	env.finishedN += 1
}

// 5. Compile
func Compile(env *EnvQgo) {
	log.Printf("[%d/%d] qgo: Compiling", env.finishedN+1, env.totalN)

	binaryName := "qgo_bin"
	if filepath.Base(os.Args[0]) == "qgo_bin" {
		binaryName = "qgo_bin_exec"
	}
	env.binaryPath = filepath.Join(env.tempDir, binaryName)

	args := []string{"build", "-o", env.binaryPath}
	args = append(args, os.Args[2:env.goFileIdx]...)
	args = append(args, filepath.Base(env.goFile))

	env.runInTemp("go", args...)

	env.finishedN += 1
}

// 6. Final execution
func Execute(env *EnvQgo) {
	log.Printf("[%d/%d] qgo: Executing", env.finishedN+1, env.totalN)

	appArgs := os.Args[env.goFileIdx+1:]
	cmd := exec.Command(env.binaryPath, appArgs...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Stdin = os.Stdin

	if err := cmd.Run(); err != nil {
		// if exitErr, ok := err.(*exec.ExitError); ok {
		// 	os.Exit(exitErr.ExitCode())
		// }
		panic(err)
	}

	env.finishedN += 1
}

// Helper: run a command inside the temp directory
func (env *EnvQgo) runInTemp(name string, args ...string) {
	cmd := exec.Command(name, args...)
	cmd.Dir = env.tempDir
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		panic(fmt.Sprintf("command failed: %s %s", name, strings.Join(args, " ")))
	}
}

func main() {
	// Override default panic behavior
	defer func() {
		if r := recover(); r != nil {
			log.Fatal(r)
		}
	}()

	env := NewEnvQgo()

	// Trigger the logic chain in order
	ParseArgs(env)
	SetupTempWorkspace(env)
	defer os.RemoveAll(env.tempDir) // Clean up at the end
	SyncSource(env)
	PrepareDependencies(env)
	Compile(env)
	Execute(env)
}
