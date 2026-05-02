package runner

import (
	"fmt"
	"os"
	"os/exec"
)

// Run takes the generated C code and executes it using TCC.
// It encapsulates all environment paths and compiler flags.
func Run(cCode string) error {
	// 1. Define paths (These could be moved to a config file later)
	tccPath := "../../tinycc/tcc"
	libPath := "/Users/fedora/repo/tinycc"

	// 2. Prepare the command:
	// -run: compile and run immediately
	// -B: set the library path for TCC internal files
	// -: read source code from stdin

	//cmd := exec.Command(tccPath, "-B"+libPath, "-run", "-")
	outputName := "output"

	cmd := exec.Command(tccPath, "-B"+libPath, "-o", outputName, "-")

	// 3. Set environment variables for TCC
	cmd.Env = append(os.Environ(), "TCC_LIB_PATH="+libPath)

	// 4. Set up pipes for input and output
	stdin, err := cmd.StdinPipe()
	if err != nil {
		return fmt.Errorf("failed to create stdin pipe: %v", err)
	}

	// Redirect TCC output to Fox compiler's output
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	// 5. Start the TCC process
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("failed to start TCC process: %v", err)
	}

	// 6. Pipe the generated C code into TCC
	_, err = stdin.Write([]byte(cCode))
	if err != nil {
		return fmt.Errorf("failed to write to stdin: %v", err)
	}
	stdin.Close() // Signal EOF to TCC so it starts compiling

	// 7. Wait for execution to complete
	if err := cmd.Wait(); err != nil {
		return fmt.Errorf("execution finished with error: %v", err)
	}

	return nil
}
