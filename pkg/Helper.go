package package_to_image_placer

import (
	"bufio"
	"bytes"
	"fmt"
	"log"
	"os"
	"os/exec"
	"regexp"
	"strings"
)

// DoesFileExists checks if file exists.
func DoesFileExists(file string) bool {
	_, err := os.Stat(file)
	return !os.IsNotExist(err)
}

// AllDepsInstalled checks if all required dependencies are installed.
var dependencies = []string{"guestmount", "guestunmount", "stty"}

// AllDepsInstalled checks if all required dependencies are installed.
// Returns true if all dependencies are installed, false otherwise.
func AllDepsInstalled() error {
	log.Printf("Checking if all dependencies installed...")
	var notInstalled []string
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

// splitStringPreserveSubstrings splits a string into substrings while preserving substrings in quotes.
// e.g. "'a b' c" -> ["'a b'", "c"]
func splitStringPreserveSubstrings(input string) []string {
	re := regexp.MustCompile(`"[^"]*"|\S+`)
	return re.FindAllString(input, -1)
}

// RunCommand runs a command. Returns stdout. If error occurs, returns also stderr.
func RunCommand(command string, verbose bool) (string, error) {
	var errbuf bytes.Buffer
	var outputString string

	split := strings.Split(command, " ")
	program := split[0]
	arguments := split[1:]

	cmd := exec.Command(program, arguments...)

	stdout, _ := cmd.StdoutPipe()
	cmd.Stderr = &errbuf
	err := cmd.Start()
	if err != nil {
		log.Printf("Error in RunCommand: %s", err.Error())
	}

	reader := bufio.NewReader(stdout)
	line, err := reader.ReadString('\n')
	for err == nil {
		if verbose {
			log.Println(line)
		}
		outputString += line
	}
	outputString += line

	err = cmd.Wait()

	stderrString := errbuf.String()
	if err != nil {
		if exitError, ok := err.(*exec.ExitError); ok {
			if verbose {
				log.Printf("Return code: %v, stderr: %v", exitError.ExitCode(), stderrString)
			}
			return outputString, fmt.Errorf("return code: %v, stderr: %v", exitError.ExitCode(), stderrString)
		}
	}
	return outputString, nil
}
