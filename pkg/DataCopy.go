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

func CopyPackageToImagePartitions(config *Configuration) error {
	for _, partition := range config.PartitionNumbers {
		log.Printf("Copying to partition: %d\n", partition)
		for _, archive := range config.Packages {
			err := MountPartitionAndCopyPackage(partition, archive, config)
			if err != nil {
				return err
			}
		}
	}
	return nil
}

func MountPartitionAndCopyPackage(partitionNumber int, archivePath string, config *Configuration) error {
	mountDir, err := os.MkdirTemp("", "mount-dir-")
	if err != nil {
		return fmt.Errorf("failed to create temporary directory: %v", err)
	}
	defer os.RemoveAll(mountDir)

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
		errChan = mountPartition(config.Target, partitionNumber, mountDir, errChan)
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
		log.Printf("Successfully mounted partition")
		if err != nil {
			return err
		}
	case <-time.After(timeout):
		return fmt.Errorf("mount command timed out")
	}

	defer func() {
		unmount(mountDir)
	}()

	var targetDirectoryFullPath string
	if config.InteractiveRun {
		targetDirectoryFullPath, err = interaction.SelectTargetDirectory(mountDir, mountDir)
		if err != nil {
			return err
		}
		config.TargetDirectory = strings.TrimPrefix(targetDirectoryFullPath, mountDir) + "/"
	} else {
		targetDirectoryFullPath = filepath.Join(mountDir, config.TargetDirectory)
	}

	serviceFiles, err := handleArchive(archivePath, mountDir, targetDirectoryFullPath, config.Overwrite)
	if err != nil {
		return err
	}

	for _, serviceFile := range serviceFiles {
		if config.InteractiveRun {
			if interaction.GetUserConfirmation(fmt.Sprintf("Do you want to activate service: %s", serviceFile)) {
				err = AddService(serviceFile, mountDir, targetDirectoryFullPath, config.Overwrite)
				if err != nil {
					return fmt.Errorf("error while activating service: %v", err)
				}
				config.ServiceNames = append(config.ServiceNames, filepath.Base(serviceFile))
			}
		} else if isServiceFileInConfig(serviceFile, config.ServiceNames) {
			err = AddService(serviceFile, mountDir, targetDirectoryFullPath, config.Overwrite)
			if err != nil {
				return fmt.Errorf("error while activating service: %v", err)
			}
		}
	}

	return nil
}

func handleArchive(archivePath, mountDir, targetDir string, overwrite bool) ([]string, error) {
	zipReader, err := zip.OpenReader(archivePath)
	if err != nil {
		return nil, fmt.Errorf("failed to open zip file: %v", err)
	}
	defer zipReader.Close()

	packageSize := getArchiveSize(zipReader)
	err = checkFreeSize(mountDir, packageSize)
	if err != nil {
		return nil, err
	}

	serviceFiles, err := decompressZipArchive(zipReader, targetDir, overwrite)
	if err != nil {
		return nil, err
	}
	return serviceFiles, nil
}

func mountPartition(targetImageName string, partitionNumber int, mountDir string, errChan chan string) chan string {
	log.Printf("Mounting partition to %s", mountDir)
	// TODO set user in config??
	cmd := fmt.Sprintf("guestmount -a %s -m /dev/sda%d -o uid=%d -o gid=%d --rw %s --no-fork", targetImageName, partitionNumber, unix.Getuid(), unix.Getgid(), mountDir)
	err := RunCommand(cmd, "./", true)
	errChan <- err
	return errChan
}

func unmount(mountDir string) {
	log.Printf("Unmounting partition")
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
	log.Printf("Copying package of size %dMB to filesystem of size %dMB\n", packageSize/1024/1024, freeSpace/1024/1024)
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
			log.Println("creating directory ", targetFilePath)
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
