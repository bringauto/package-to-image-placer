package interaction

import (
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
	"strconv"
	"strings"
)

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

func SelectFile(dir string) (string, error) {
	absDir, err := filepath.Abs(dir)
	if err != nil {
		return "", err
	}
	//log.Printf("Current directory: %s\n", absDir)
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
}

func SelectPartitions(diskPath string) ([]int, error) {
	allPartitions := getPartitionInfo(diskPath)
	partitionInfo := make([]string, len(allPartitions))
	for index, partition := range allPartitions {
		partitionInfo[index] = fmt.Sprintf("Partition %d: %s\n\tFilesystem: %s", partition.partitionNumber, partition.partitionUUID, partition.filesystemType)
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
func SelectTargetDirectory(searchDir string) (string, error) {
	// TODO make sure not to get out of root dir
	dirs, err := getDirectories(searchDir)
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
		var newDir string
		_, err := fmt.Scanln(&newDir)
		if err != nil {
			return "", err
		}
		newDir = filepath.Join(searchDir, newDir)
		err = os.Mkdir(newDir, 0755)
		if err != nil {
			return "", err
		}
		return newDir, nil
	} else {
		return SelectTargetDirectory(filepath.Join(searchDir, selectedDir))
	}
}

// getDirectories returns a list of directories in the provided path. The list includes the parent directory (..).
func getDirectories(path string) ([]string, error) {
	var dirs []string
	dirContent, err := os.ReadDir(path)
	if err != nil {
		return nil, err
	}
	for _, file := range dirContent {
		if file.IsDir() && file.Name() != path {
			dirs = append(dirs, file.Name()+"/")
		}
	}

	dirs = append([]string{"../"}, dirs...)
	return dirs, nil
}

func GetUserIntInput(message string) []int {
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

// GetUserConfirmation asks the user for confirmation. The message is displayed to the user.
func GetUserConfirmation(message string) bool {
	var b = make([]byte, 1)
	fmt.Print(message + " [Y|y to confirm, any other key to cancel]\n")
	_, err := os.Stdin.Read(b)
	if err != nil {
		return false
	}

	return !(string(b) != "Y" && string(b) != "y")
}

func SetUpCommandline() {
	log.Printf("Setting up command line...")
	if !term.IsTerminal(int(os.Stdout.Fd())) {
		log.Printf("Warning: /dev/tty not available, skipping terminal setup")
		return
	}
	//do not cache characters
	exec.Command("stty", "-F", "/dev/tty", "cbreak", "min", "1").Run()
}

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
	return selectedIndex[0], nil
}

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
		}
		partitions = append(partitions, partition)
	}
	return partitions
}

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
