package main

import (
	"log"
)

func main() {
	// Override default panic behavior
	defer func() {
		if r := recover(); r != nil {
			log.Fatal(r)
		}
	}()

	qgo := NewModQgo()

	// Trigger the logic chain in order
	qgo.ParseArgs()
	qgo.SetupTempWorkspace()
	defer qgo.Cleanup() // Clean up at the end
	qgo.SyncSource()
	qgo.PrepareDependencies()
	qgo.Compile()
	qgo.Execute()
}
