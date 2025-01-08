package package_to_image_placer

import (
	"archive/zip"
	"fmt"
	"golang.org/x/sys/unix"
	"io"
	"log"
	"os"
	"os/signal"
	"package-to-image-placer/pkg/interaction"
	"path/filepath"
	"strings"
	"syscall"
	"time"
)

const timeout = 60 * time.Second

func UnzipPackageToImage(targetImageName string, archivePath string, partitionNumber int, targetFolderPath string, overwrite bool) error {
	mountDir, err := os.MkdirTemp("", "mount-dir-")
	if err != nil {
		return fmt.Errorf("failed to create temporary directory: %v", err)
	}
	//defer os.RemoveAll(mountDir)
	println("mounting partition to " + mountDir)

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		sig := <-sigChan
		fmt.Printf("Received signal: %s\n", sig)
		unmount(mountDir)
		os.Exit(1)
	}()

	errChan := make(chan string)
	go func() {
		cmd := fmt.Sprintf("guestmount -a %s -m /dev/sda%d -o uid=%d -o gid=%d --rw %s --no-fork", targetImageName, partitionNumber, unix.Getuid(), unix.Getgid(), mountDir)
		err := RunCommand(cmd, "./", true)
		errChan <- err
	}()

	populatedChan := make(chan error)
	go func() {
		populatedChan <- WaitUntilDirectoryIsPopulated(mountDir, timeout)
	}()

	select {
	case err := <-errChan:
		if err != "" {
			return fmt.Errorf("failed to mount partition: %v", err)
		}
	case err := <-populatedChan: // Wait until the directory is populated
		println("Successfully mounted partition")
		if err != nil {
			return err
		}
	case <-time.After(timeout):
		return fmt.Errorf("mount command timed out")
	}

	defer func() {
		unmount(mountDir)
	}()

	targetDir, err := interaction.SelectTargetDirectory(mountDir, mountDir)
	if err != nil {
		return err
	}

	zipReader, err := zip.OpenReader(archivePath)
	if err != nil {
		return fmt.Errorf("failed to open zip file: %v", err)
	}
	defer zipReader.Close()

	packageSize := getArchiveSize(zipReader)
	err = checkFreeSize(mountDir, packageSize)
	if err != nil {
		return err
	}

	serviceFiles, err := decompressZipArchive(zipReader, targetDir, overwrite)
	if err != nil {
		return err
	}

	err = AddService(serviceFiles, mountDir, targetDir, overwrite)
	if err != nil {
		return fmt.Errorf("error while activating service: %v", err)
	}
	return nil
}

func unmount(mountDir string) {
	println("Unmounting partition")
	RunCommand("guestunmount "+mountDir, "./", false)
}

func getArchiveSize(zipReader *zip.ReadCloser) uint64 {
	packageSize := uint64(0)
	for _, file := range zipReader.File {
		packageSize += file.UncompressedSize64
	}
	return packageSize
}

func checkFreeSize(mountDir string, packageSize uint64) error {
	var stat unix.Statfs_t
	err := unix.Statfs(mountDir, &stat)
	if err != nil {
		return fmt.Errorf("error getting free space on filesystem: %s", err.Error())
	}

	freeSpace := stat.Bfree * uint64(stat.Bsize)
	if packageSize > freeSpace {
		return fmt.Errorf("not enough space to copy package. Free space on partition: %dMB, package size: %dMB", freeSpace/1024/1024, packageSize/1024/1024)
	}
	fmt.Printf("Copying package of size %dMB to filesystem of size %dMB\n", packageSize/1024/1024, freeSpace/1024/1024)
	return nil
}

func decompressZipArchive(zipReader *zip.ReadCloser, targetDir string, overwrite bool) ([]string, error) {
	var serviceFiles []string

	for _, file := range zipReader.File {
		targetFilePath := filepath.Join(targetDir, file.Name)
		//fmt.Println("unzipping file ", targetFilePath)

		if !strings.HasPrefix(targetFilePath, filepath.Clean(targetDir)+string(os.PathSeparator)) { // Check if the file is within the target directory
			return nil, fmt.Errorf("invalid file path")
		}
		if file.FileInfo().IsDir() {
			fmt.Println("creating directory ", targetFilePath)
			if err := os.MkdirAll(targetFilePath, os.ModePerm); err != nil {
				return nil, err
			}
			continue
		}

		if err := os.MkdirAll(filepath.Dir(targetFilePath), os.ModePerm); err != nil {
			return nil, err
		}
		if err := decompressZipFile(targetFilePath, file, overwrite); err != nil {
			return nil, err
		}
		if strings.HasSuffix(file.Name, ".service") {
			serviceFiles = append(serviceFiles, targetFilePath)
		}
	}
	return serviceFiles, nil
}

func copyFilesRecursively(destPath, srcPath string) error {
	// Get the source info
	srcInfo, err := os.Stat(srcPath)
	if err != nil {
		return fmt.Errorf("unable to get source info: %v", err)
	}

	// If srcPath is a file, copy it directly
	if !srcInfo.IsDir() {
		return copyFile(filepath.Join(destPath, filepath.Base(srcPath)), srcPath, srcInfo.Mode())
	}
	//baseDir := filepath.Base(srcPath)
	//destPath = filepath.Join(destPath, baseDir)
	// Create the destination directory
	err = os.MkdirAll(destPath, srcInfo.Mode())
	if err != nil {
		return fmt.Errorf("unable to create destination directory: %v", err)
	}

	// Read the source directory
	files, err := os.ReadDir(srcPath)
	if err != nil {
		return fmt.Errorf("unable to read directory %s: %v", srcPath, err)
	}

	for _, file := range files {
		srcFilePath := filepath.Join(srcPath, file.Name())
		destFilePath := filepath.Join(destPath, file.Name())

		fileInfo, _ := file.Info()
		if file.IsDir() {
			err := copyFilesRecursively(destFilePath, srcFilePath)
			if err != nil {
				fmt.Printf("Unable to copy directory %s: %v\n", srcFilePath, err)
				continue
			}
		} else {
			err := copyFile(destFilePath, srcFilePath, fileInfo.Mode())
			if err != nil {
				fmt.Printf("Unable to copy file %s: %v\n", srcFilePath, err)
			}
		}
	}
	return nil
}

func decompressZipFile(destFilePath string, srcZipFile *zip.File, overwrite bool) error {
	log.Printf("Decompressing file %s to %s", srcZipFile.Name, destFilePath)
	if !overwrite {
		_, err := os.Stat(destFilePath)
		if err == nil {
			return fmt.Errorf("file %s already exists. Use -overwrite flag to overwrite existing files.", destFilePath)
		}
	}
	srcFile, err := srcZipFile.Open()
	if err != nil {
		return fmt.Errorf("unable to open file %s: %v", srcZipFile.Name, err)
	}
	defer srcFile.Close()

	destFile, err := os.OpenFile(destFilePath, os.O_CREATE|os.O_WRONLY, srcZipFile.Mode())
	if err != nil {
		return fmt.Errorf("unable to create file %s: %v", destFilePath, err)
	}
	defer destFile.Close()

	_, err = io.Copy(destFile, srcFile)
	if err != nil {
		return fmt.Errorf("unable to copy file %s: %v", srcZipFile.Name, err)
	}
	// TODO do we want to change permissions if the file already exists?
	// Set the file permissions
	//err = os.Chmod(destFilePath, fileMode)
	//if err != nil {
	//	return fmt.Errorf("unable to set file permissions for %s: %v", destFilePath, err)
	//}
	return nil
}

func copyFile(destFilePath, srcFilePath string, fileMode os.FileMode) error {
	srcFile, err := os.Open(srcFilePath)
	if err != nil {
		return fmt.Errorf("unable to open file %s: %v", srcFilePath, err)
	}
	defer srcFile.Close()

	destFile, err := os.OpenFile(destFilePath, os.O_CREATE|os.O_WRONLY, fileMode)
	if err != nil {
		return fmt.Errorf("unable to create file %s: %v", destFilePath, err)
	}
	defer destFile.Close()

	_, err = io.Copy(destFile, srcFile)
	if err != nil {
		return fmt.Errorf("unable to copy file %s: %v", srcFilePath, err)
	}
	// TODO do we want to change permissions if the file already exists?
	// Set the file permissions
	//err = os.Chmod(destFilePath, fileMode)
	//if err != nil {
	//	return fmt.Errorf("unable to set file permissions for %s: %v", destFilePath, err)
	//}
	return nil
}
func FolderSize(path string) (uint64, error) {
	var size uint64
	err := filepath.Walk(path, func(_ string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() {
			size += uint64(info.Size())
		}
		return err
	})
	return size, err
}

// WaitUntilDirectoryIsPopulated waits until the directory is populated or the timeout is reached
func WaitUntilDirectoryIsPopulated(dirPath string, timeout time.Duration) error {
	start := time.Now()
	for {
		populated, err := IsDirectoryPopulated(dirPath)
		if err != nil {
			return err
		}
		if populated {
			return nil
		}
		if time.Since(start) > timeout {
			return fmt.Errorf("directory %s is not populated within the timeout period", dirPath)
		}
		time.Sleep(500 * time.Millisecond) // Adjust the sleep duration as needed
	}
}

// IsDirectoryPopulated checks if the given directory is populated
func IsDirectoryPopulated(dirPath string) (bool, error) {
	dir, err := os.Open(dirPath)
	if err != nil {
		return false, fmt.Errorf("unable to open directory: %v", err)
	}
	defer dir.Close()

	// Read directory contents
	_, err = dir.Readdirnames(1) // Or use dir.Readdir(1) for more details
	if err == nil {
		return true, nil // Directory is populated
	}
	if err == io.EOF {
		return false, nil // Directory is empty
	}
	return false, fmt.Errorf("error reading directory: %v", err)
}
