package helper

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"log"
	"os"
	"os/exec"
	"path/filepath"
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

// ValidSourceImage checks if the source image exists and is a non empty file.
// Returns an error if the source image does not exist, is a directory or is empty.
func ValidSourceImage(imagePath string) error {
	if !DoesFileExists(imagePath) {
		return fmt.Errorf("source image file does not exist: %s", imagePath)
	}
	fileInfo, err := os.Stat(imagePath)
	if err != nil {
		return err
	}
	if fileInfo.IsDir() {
		return fmt.Errorf("source image is a directory: %s", imagePath)
	}
	if fileInfo.Size() == 0 {
		return fmt.Errorf("source image file is empty: %s", imagePath)
	}
	return nil
}

// RemoveInvalidOutputImage removes the output image if it exists.
// Returns an error if the output image exists and cannot be removed.
func RemoveInvalidOutputImage(outputImage string, no_clone bool) error {
	if DoesFileExists(outputImage) && !no_clone {
		err := os.Remove(outputImage)
		if err != nil {
			return fmt.Errorf("failed to remove invalid output image: %s", outputImage)
		}
	}
	return nil
}

// GetMountDir returns the mount directory for the given image path.
func GetTargetArchiveDirName(targetDir string, archivePath string, standardPackage bool) string {
	if standardPackage {
		return filepath.Join(targetDir, strings.TrimSuffix(filepath.Base(archivePath), ".zip"))
	}
	return targetDir
}

// SplitStringPreserveSubstrings splits a string into substrings while preserving substrings in quotes.
// e.g. "'a b' c" -> ["'a b'", "c"]
func SplitStringPreserveSubstrings(input string) []string {
	re := regexp.MustCompile(`"[^"]*"|\S+`)
	return re.FindAllString(input, -1)
}

func RemoveMountDirAndPackageName(path string, mountDir string, packageDir string, packagePath string) string {
	// strings.TrimPrefix(serviceFile, mountDir+strings.TrimSuffix(filepath.Base(packageConfig.PackagePath), ".zip"))
	path = strings.TrimPrefix(path, mountDir)

	path = strings.TrimPrefix(path, "/")
	packageDir = strings.TrimPrefix(packageDir, "/")
	path = strings.TrimPrefix(path, packageDir)

	path = strings.TrimPrefix(path, "/")
	packageName := strings.TrimSuffix(filepath.Base(packagePath), ".zip")
	path = strings.TrimPrefix(path, packageName)

	if path[0] != '/' {
		path = "/" + path
	}

	return path
}

// CopyFile copies a file from the source path to the destination path with the specified file mode.
// It returns an error if the file cannot be opened or created.
func CopyFile(destFilePath, srcFilePath string, fileMode os.FileMode) error {
	srcFile, err := os.Open(srcFilePath)
	if err != nil {
		return fmt.Errorf("unable to open file %s: %v", srcFilePath, err)
	}
	defer srcFile.Close()

	destFile, err := os.OpenFile(destFilePath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, fileMode)
	if err != nil {
		return fmt.Errorf("unable to create file %s: %v", destFilePath, err)
	}
	defer destFile.Close()

	_, err = io.Copy(destFile, srcFile)
	if err != nil {
		return fmt.Errorf("unable to copy file %s: %v", srcFilePath, err)
	}
	if err := destFile.Sync(); err != nil {
		return fmt.Errorf("failed to flush data to service file: %v", err)
	}
	// TODO do we want to change permissions if the file already exists?
	// Set the file permissions
	//err = os.Chmod(destFilePath, fileMode)
	//if err != nil {
	// return fmt.Errorf("unable to set file permissions for %s: %v", destFilePath, err)
	//}
	return nil
}

// IsWithinRootDir checks if the targetPath is within rootDir.
func IsWithinRootDir(rootDir, targetPath string) bool {
	// Clean paths to normalize
	rootDir = filepath.Clean(rootDir)
	targetPath = filepath.Clean(targetPath)

	// Get the relative path
	rel, err := filepath.Rel(rootDir, targetPath)
	if err != nil {
		return false
	}

	// Check if the relative path does not escape the root
	return !strings.HasPrefix(rel, "..") && !filepath.IsAbs(rel)
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
