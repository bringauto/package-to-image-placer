package helper

import (
	"fmt"
	"log"
	"os"
	"os/exec"
	"strings"
)

func DoesFileExists(file string) bool {
	_, err := os.Stat(file)
	return !os.IsNotExist(err)
}

// AllDepsInstalled checks if all required dependencies are installed.
// Returns true if all dependencies are installed, false otherwise.
func AllDepsInstalled() error {
	log.Printf("Checking if all dependencies installed...")
	notInstalled := []string{}
	allInstalled := true
	for _, dep := range dependencies {
		_, err1 := exec.LookPath(dep) // check if executable exists
		cmdCheckPackage := exec.Command("dpkg", "-s", dep)
		err2 := cmdCheckPackage.Run() // check if package is installed
		if err1 != nil && err2 != nil {
			allInstalled = false
			notInstalled = append(notInstalled, dep)
		}
	}
	if !allInstalled {
		return fmt.Errorf("these dependencies are not installed: %s", strings.Join(notInstalled, ", "))
	}
	return nil
}
