package package_to_image_placer

import (
	"bufio"
	"bytes"
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

func RunCommand(command, path string, verbose bool) string {
	var errbuf bytes.Buffer
	var outputString string

	splitted := strings.Split(command, " ")
	program := splitted[0]
	arguments := splitted[1:]

	cmd := exec.Command(program, arguments...)

	cmd.Dir = path

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
			fmt.Print(line)
		}
		outputString += line
		line, err = reader.ReadString('\n')
	}

	err = cmd.Wait()

	stderr_string := errbuf.String()
	if err != nil {
		if exitError, ok := err.(*exec.ExitError); ok {
			log.Printf("Return code: %v, stderr: %v", exitError.ExitCode(), stderr_string)
			panic("Error while running command: " + command)
		}
	}
	return outputString
}
