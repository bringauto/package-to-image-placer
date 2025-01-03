package main

import (
	"flag"
	"fmt"
	"github.com/diskfs/go-diskfs/disk"
	"log"
	"os"
	"os/exec"
	packagePlacer "package-to-image-placer/pkg"
	"path/filepath"
	"strconv"
	"strings"
)

func main() {
	selectedFile, err := fuzzySelect(".")
	if err != nil {
		fmt.Printf("Error: %s\n", err)
		return
	}

	fmt.Printf("Selected file: %s\n", selectedFile)
	return
	err = packagePlacer.AllDepsInstalled()
	if err != nil {
		fmt.Printf("Error: %s\n", err)
		os.Exit(1)
	}
	sourceImage := flag.String("source", "", "Source image")
	targetImage := flag.String("target", "", "Target image path")
	flag.Parse()

	if *targetImage == "" || *sourceImage == "" {
		fmt.Printf("Missing arguments, start with -h to see arguments.\n")
		return
	}

	if packagePlacer.DoesFileExists(*targetImage) {
		askUser := fmt.Sprintf("File %s already exists. Do you want to delete it? [Y|y to confirm]\n", *targetImage)
		if !getUserConfirmation(askUser) {
			println("file already exists and user chose not to delete it")
			return
		}
		if err := os.Remove(*targetImage); err != nil {
			fmt.Printf("unable to delete existing file: %s", err)
			return
		}
	}
	imageData, err := packagePlacer.CloneImage(*sourceImage, *targetImage)
	if err != nil {
		fmt.Printf("Error: %s\n", err)
		return
	}

	targetPartitions := selectPartitions(imageData.TargetDisk)
	// TODO Will only accept zip files, will unzip them.
	for _, partition := range targetPartitions {
		err = packagePlacer.UnzipPackageToImage(*targetImage, "test-image-folder", partition, "")
		if err != nil {
			println(err.Error())
			return
		}
	}
	return
}

func getFiles(dir string) ([]string, error) {
	var files []string
	dirContent, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}
	for _, file := range dirContent {
		if file.IsDir() && file.Name() != dir {
			files = append(files, file.Name()+"/")
		} else if strings.HasSuffix(file.Name(), ".zip") {
			files = append(files, file.Name())
		}
	}

	files = append([]string{"../"}, files...)
	return files, nil
}

func fuzzySelect(dir string) (string, error) {
	absDir, err := filepath.Abs(dir)
	if err != nil {
		return "", err
	}
	log.Printf("Current directory: %s\n", absDir)
	files, err := getFiles(dir)
	if err != nil {
		return "", err
	}

	header := "Choose file to copy. Press esc to quit.\nCurrent directory: " + absDir
	cmd := exec.Command("fzf", "--header", header) // TODO use https://pkg.go.dev/github.com/junegunn/fzf/src#Run
	cmd.Stdin = strings.NewReader(strings.Join(files, "\n"))
	output, err := cmd.Output()
	if err != nil {
		return "", err
	}

	selectedFile := strings.TrimSpace(string(output))
	if strings.HasSuffix(selectedFile, "/") || selectedFile == "../" {
		newDir := filepath.Join(dir, selectedFile)
		return fuzzySelect(newDir)
	}

	return selectedFile, nil
}
func selectPartitions(disk *disk.Disk) []int {
	totalPartitions := packagePlacer.ListPartitions(disk)
	var partitions []int

	for {
		partitions = getUserIntInput("Select partitions to which the package will be copied (space separated): ")
		valid := true
		partitionSet := make(map[int]struct{})

		for _, partition := range partitions {
			if partition < 1 || partition > totalPartitions {
				log.Printf("Invalid partition number: %d\n", partition)
				valid = false
				break
			}
			if _, exists := partitionSet[partition]; exists {
				log.Printf("Duplicate partition number: %d\n", partition)
				valid = false
				break
			}
			partitionSet[partition] = struct{}{} // Create empty struct to save memory
		}

		if valid {
			break
		} else {
			fmt.Println("Please enter valid and unique partition numbers.")
		}
	}

	return partitions
}

func getUserIntInput(message string) []int {
	fmt.Print(message)
	var input string
	_, err := fmt.Scanln(&input)
	if err != nil {
		fmt.Println(err)
	}

	fields := strings.Fields(input)
	integers := make([]int, len(fields))
	for i, field := range fields {
		num, err := strconv.Atoi(field)
		if err != nil {
			fmt.Println(err)
			continue
		}
		integers[i] = num
	}
	return integers
}
func getUserConfirmation(message string) bool {
	var b []byte = make([]byte, 1)
	fmt.Print(message)
	_, err := os.Stdin.Read(b)
	if err != nil {
		return false
	}

	return !(string(b) != "Y" && string(b) != "y")
}
