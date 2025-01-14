package interaction

import (
	"bufio"
	"fmt"
	"github.com/diskfs/go-diskfs"
	"github.com/diskfs/go-diskfs/filesystem"
	"github.com/koki-develop/go-fzf"
	"golang.org/x/term"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"strings"
)

// SelectFilesInDir allows the user to select multiple files in a directory.
// It repeatedly prompts the user to select files until they choose to stop.
// Returns a slice of selected file paths.
func SelectFilesInDir(dir string) ([]string, error) {
	var selectedFiles []string
	chooseAnotherFile := true
	for chooseAnotherFile {
		selectedFile, err := SelectFile(dir)
		if err != nil {
			if err.Error() == "abort" {
				fmt.Println("User aborted")
				break
			}
			return nil, err
		}
		selectedFiles = append(selectedFiles, selectedFile)
		chooseAnotherFile = GetUserConfirmation("Do you want to select another file?")
	}
	selectedFiles = removeDuplicates(selectedFiles).([]string)
	return selectedFiles, nil
}

// getDirsAndZips returns a list of directories and zip files in the specified directory.
// The list includes an option to navigate to the parent directory.
func getDirsAndZips(dir string) ([]string, error) {
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

// SelectFile allows the user to select a file from the specified directory.
// It uses a fuzzy finder to present the files and directories to the user.
// Returns the selected file path.
func SelectFile(dir string) (string, error) {
	absDir, err := filepath.Abs(dir)
	if err != nil {
		return "", err
	}
	files, err := getDirsAndZips(dir)
	if err != nil {
		return "", err
	}

	header := "Choose file to copy. Press esc to quit.\nCurrent directory: " + absDir + "\n"

	selectedFileIndex, err := fuzzySelectOne(header, files)
	if err != nil {
		return "", err
	}

	selectedFile := strings.TrimSpace(files[selectedFileIndex])
	if strings.HasSuffix(selectedFile, "/") || selectedFile == "../" {
		newDir := filepath.Join(dir, selectedFile)
		return SelectFile(newDir)
	}
	selectedFile = filepath.Join(absDir, selectedFile)
	return selectedFile, nil
}

// removeDuplicates removes duplicate elements from a slice.
// It returns a new slice with unique elements.
func removeDuplicates(slice interface{}) interface{} {
	sliceValue := reflect.ValueOf(slice)
	if sliceValue.Kind() != reflect.Slice {
		fmt.Println("Warning: Provided value is not a slice")
		return slice
	}

	uniqueMap := make(map[interface{}]bool)
	uniqueSlice := reflect.MakeSlice(sliceValue.Type(), 0, sliceValue.Len())

	for i := 0; i < sliceValue.Len(); i++ {
		item := sliceValue.Index(i).Interface()
		if _, exists := uniqueMap[item]; !exists {
			uniqueMap[item] = true
			uniqueSlice = reflect.Append(uniqueSlice, sliceValue.Index(i))
		} else {
			fmt.Printf("Warning: Duplicate item found: %v\n", item)
		}
	}

	return uniqueSlice.Interface()
}

type partitionInfo struct {
	partitionNumber int
	partitionUUID   string
	filesystemType  string
	filesystemLabel string
}

// SelectPartitions allows the user to select multiple partitions from a disk image.
// It repeatedly prompts the user to select partitions until they choose to stop.
// Returns a slice of selected partition numbers.
func SelectPartitions(diskPath string) ([]int, error) {
	allPartitions := getPartitionInfo(diskPath)
	partitionInfo := make([]string, len(allPartitions))
	for index, partition := range allPartitions {
		partitionInfo[index] = fmt.Sprintf("Partition %d: %s\n\tFilesystem: '%s' Type: %s", partition.partitionNumber, partition.partitionUUID, partition.filesystemLabel, partition.filesystemType)
	}

	var partitionsNumbers []int

	chooseAnotherFile := true
	for chooseAnotherFile {
		selectedPartitionIndex, err := fuzzySelectOne("Select partition to which the package will be copied: ", partitionInfo)
		if err != nil {
			if err.Error() == "abort" {
				fmt.Println("User aborted")
				break
			}
			return nil, err
		}
		partitionsNumbers = append(partitionsNumbers, allPartitions[selectedPartitionIndex].partitionNumber)

		chooseAnotherFile = GetUserConfirmation("Do you want to select another partition?")
	}
	partitionsNumbers = removeDuplicates(partitionsNumbers).([]int)
	return partitionsNumbers, nil
}

// SelectTargetDirectory allows the user to select a directory to copy the package to.
// The user can also create a new directory.
// Returns the selected directory.
func SelectTargetDirectory(rootDir, searchDir string) (string, error) {
	// Validate that searchDir is within the rootDir
	if !isWithinRoot(rootDir, searchDir) {
		return "", fmt.Errorf("attempt to navigate outside the allowed root directory")
	}

	// Get directories within searchDir
	dirs, err := getDirectories(searchDir, rootDir)
	if err != nil {
		return "", err
	}

	// Add options for current directory and creating a new directory
	dirs = append([]string{"Select current directory", "Create new directory"}, dirs...)

	header := "Select directory to copy package to. Press esc to quit.\nCurrent directory: " + searchDir + "\n"
	idx, err := fuzzySelectOne(header, dirs)
	if err != nil {
		return "", err
	}

	// Handle the selected option
	selectedDir := dirs[idx]
	if selectedDir == "Select current directory" {
		return searchDir, nil
	} else if selectedDir == "Create new directory" {
		fmt.Print("Enter new directory name: ")
		newDir, err := ReadStringFromUser()
		if err != nil {
			return "", err
		}
		newDirPath := filepath.Join(searchDir, newDir)
		if !isWithinRoot(rootDir, newDirPath) {
			return "", fmt.Errorf("attempt to create directory outside of root directory")
		}
		err = os.Mkdir(newDirPath, 0755)
		if err != nil {
			return "", err
		}
		return newDirPath, nil
	} else {
		// Recurse into the selected directory
		nextDir := filepath.Join(searchDir, selectedDir)
		return SelectTargetDirectory(rootDir, nextDir)
	}
}

// ReadStringFromUser reads a string input from the user.
// Returns the input string.
func ReadStringFromUser() (string, error) {
	reader := bufio.NewReader(os.Stdin)

	fmt.Printf("Enter path to save configuration file: ")
	path, err := reader.ReadString('\n')
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(path), nil
}

// getDirectories returns a list of directories in the provided path.
func getDirectories(path string, rootDir string) ([]string, error) {
	var dirs []string
	dirContent, err := os.ReadDir(path)
	if err != nil {
		return nil, err
	}
	for _, file := range dirContent {
		if file.IsDir() {
			dirs = append(dirs, file.Name()+"/")
		}
	}

	if filepath.Clean(path) != filepath.Clean(rootDir) {
		dirs = append([]string{"../"}, dirs...)
	}

	return dirs, nil
}

// isWithinRoot checks if the targetPath is within rootDir.
func isWithinRoot(rootDir, targetPath string) bool {
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

// GetUserConfirmation asks the user for confirmation. The message is displayed to the user.
// Returns true if the user confirms, false otherwise.
func GetUserConfirmation(message string) bool {
	var b = make([]byte, 1)
	fmt.Print(message + " [Y|y to confirm, any other key to cancel]\n")
	_, err := os.Stdin.Read(b)
	if err != nil {
		return false
	}

	return !(string(b) != "Y" && string(b) != "y")
}

// SetUpCommandline sets up the command line for user interaction.
// It configures the terminal to not cache characters.
func SetUpCommandline() {
	log.Printf("Setting up command line...")
	if !term.IsTerminal(int(os.Stdout.Fd())) {
		log.Printf("Warning: /dev/tty not available, skipping terminal setup")
		return
	}
	//do not cache characters
	exec.Command("stty", "-F", "/dev/tty", "cbreak", "min", "1").Run()
	exec.Command("stty", "-F", "/dev/tty", "-echo").Run()
}

// CleanUpCommandLine reverts the terminal settings to their default state.
func CleanUpCommandLine() {
	log.Printf("Cleaning up command line...")
	if !term.IsTerminal(int(os.Stdout.Fd())) {
		log.Printf("Warning: /dev/tty not available, skipping terminal cleanup")
		return
	}
	// Reset terminal settings
	exec.Command("stty", "-F", "/dev/tty", "sane").Run()
	exec.Command("stty", "-F", "/dev/tty", "echo").Run()
}

// fuzzySelectOne presents a fuzzy finder to the user to select one item from a list.
// Returns the index of the selected item.
func fuzzySelectOne(prompt string, items []string) (int, error) {
	f, err := fzf.New(fzf.WithPrompt(prompt))
	if err != nil {
		return -1, err
	}
	defer f.Quit()
	selectedIndex, err := f.Find(items, func(i int) string { return items[i] })
	if err != nil {
		return -1, err
	}
	if len(selectedIndex) == 0 {
		return -1, fmt.Errorf("abort")
	}
	return selectedIndex[0], nil
}

// getPartitionInfo retrieves partition information from a disk image.
// Returns a slice of partitionInfo structs.
func getPartitionInfo(imagePath string) []partitionInfo {
	disk, _ := diskfs.Open(imagePath)
	defer disk.Close()
	table, err := disk.GetPartitionTable()
	if err != nil {
		log.Fatal(err)
	}
	var partitions []partitionInfo
	for index, p := range table.GetPartitions() {
		partitionNumber := index + 1
		fs, err := disk.GetFilesystem(partitionNumber)
		if err != nil {
			log.Printf("Error getting filesystem on partition %d: %s\n", partitionNumber, err)
		}
		partition := partitionInfo{
			partitionNumber: partitionNumber,
			partitionUUID:   p.UUID(),
			filesystemType:  typeToString(fs.Type()),
			filesystemLabel: fs.Label(),
		}
		partitions = append(partitions, partition)
	}
	return partitions
}

// typeToString converts a filesystem.Type to a string representation.
// Returns the string representation of the filesystem type.
func typeToString(t filesystem.Type) string {
	switch t {
	case filesystem.TypeFat32:
		return "FAT32"
	case filesystem.TypeISO9660:
		return "ISO9660"
	case filesystem.TypeSquashfs:
		return "Squashfs"
	case filesystem.TypeExt4:
		return "Ext4"
	default:
		return "Unknown"
	}
}
