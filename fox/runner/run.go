package runner

import (
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
)

// Run executes the external compiler infrastructure pipeline.
// It synchronizes physical resource locations and manages stream lifetimes.
func Run(cCode string) error {
	cwd, err := os.Getwd()
	if err != nil {
		log.Fatalf("Failed to retrieve current working directory: %v", err)
	}

	genFileName := filepath.Join(cwd, "main_gen.c")
	fgcSourcePath := filepath.Join(cwd, "foxgc", "fgc.c")
	outputExecutablePath := filepath.Join(cwd, "output")

	// Emit the program payload onto the persistent storage layer
	err = os.WriteFile(genFileName, []byte(cCode), 0644)
	if err != nil {
		log.Fatalf("Failed to physically write generated C code: %v", err)
	}

	tccPath := "../../tinycc/tcc"
	libPath := "/Users/fedora/repo/tinycc"

	// Bind the compilation phase to specific structural paths, avoiding stdin duplication
	cmd := exec.Command(tccPath, "-B"+libPath, "-o", outputExecutablePath, genFileName, fgcSourcePath)

	cmd.Env = append(os.Environ(), "TCC_LIB_PATH="+libPath)

	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	// Invoke the backend compiler binary descriptor directly
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("failed to start TCC process: %v", err)
	}

	if err := cmd.Wait(); err != nil {
		return fmt.Errorf("execution finished with error: %v", err)
	}

	return nil
}
