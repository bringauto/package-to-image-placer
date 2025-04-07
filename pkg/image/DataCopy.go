package image

import (
	"archive/zip"
	"fmt"
	"io"
	"log"
	"os"
	"os/signal"
	"package-to-image-placer/pkg/configuration"
	"package-to-image-placer/pkg/helper"
	"package-to-image-placer/pkg/service"
	"package-to-image-placer/pkg/user"
	"path/filepath"
	"slices"
	"strings"
	"syscall"
	"time"

	"golang.org/x/sys/unix"
)

const timeout = 60 * time.Second

// CopyPackageToImagePartitions copies the specified packages to the specified partitions in the configuration.
// It iterates over each partition and package, calling MountPartitionAndCopyPackages for each combination.
func CopyPackageToImagePartitions(config *configuration.Configuration) error {
	for _, partition := range config.PartitionNumbers {
		log.Printf("Copying to partition: %d\n", partition)
		err := MountPartitionAndCopyPackages(partition, config)
		if err != nil {
			return err
		}
	}
	return nil
}

func CopyPackageActivateService(mountDir string, config *configuration.Configuration, packageConfig *configuration.PackageConfig) error {
	var targetDirectoryFullPath string
	var err error
	if config.InteractiveRun {
		targetDirectoryFullPath, err = user.SelectTargetDirectory(mountDir, mountDir, packageConfig.PackagePath)
		if err != nil {
			return err
		}
		packageConfig.TargetDirectory = strings.TrimPrefix(targetDirectoryFullPath, mountDir) + "/"
	} else {
		targetDirectoryFullPath = filepath.Join(mountDir, packageConfig.TargetDirectory)
		err := os.MkdirAll(targetDirectoryFullPath, os.ModePerm)
		if err != nil {
			return fmt.Errorf("failed to create target directory: %v", err)
		}
	}
	log.Printf("Copying package to target directory: %s\n", targetDirectoryFullPath)
	if !helper.IsWithinRootDir(mountDir, targetDirectoryFullPath) {
		return fmt.Errorf("target directory is not within the mounted partition")
	}

	serviceFile, err := handleArchive(packageConfig, mountDir, targetDirectoryFullPath, config.InteractiveRun)
	if err != nil {
		return err
	}

	if config.InteractiveRun {
		packageConfig.EnableServices = user.GetUserConfirmation("Do you want to enable services for package " + packageConfig.PackagePath + "?")
		if packageConfig.EnableServices {
			packageConfig.ServiceNameSuffix, err = user.ReadStringFromUser("Enter service name suffix (leave empty for none): ")
			if err != nil {
				return fmt.Errorf("error reading service name suffix: %v", err)
			}
			if strings.HasPrefix(packageConfig.ServiceNameSuffix, "-") {
				return fmt.Errorf("service name suffix should not start with a hyphen")
			}
		}
	}

	if packageConfig.EnableServices {
		err = service.AddService(serviceFile, mountDir, targetDirectoryFullPath, packageConfig)
		if err != nil {
			return fmt.Errorf("error while activating service: %v", err)
		}
	}
	return nil
}

// MountPartitionAndCopyPackages mounts the specified partition, copies the package to it, and activates any service files found in the package.
// It handles signals for unmounting the partition and ensures the directory is populated before proceeding.
func MountPartitionAndCopyPackages(partitionNumber int, config *configuration.Configuration) error {
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
		populatedChan <- waitUntilDirectoryIsPopulated(mountDir, timeout)
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

	for _, packageConfig := range config.Packages {
		err = CopyPackageActivateService(mountDir, config, &packageConfig)
		if err != nil {
			return fmt.Errorf("error while copying package: %v", err)
		}
	}

	return nil
}

// handleArchive handles the extraction of the archive file to the target directory.
// It checks for sufficient free space and returns a service file if found.
func handleArchive(packageConfig *configuration.PackageConfig, mountDir, targetDir string, interactiveRun bool) (string, error) {
	archivePath := packageConfig.PackagePath
	zipReader, err := zip.OpenReader(archivePath)
	if err != nil {
		return "", fmt.Errorf("failed to open zip file: %v", err)
	}
	defer zipReader.Close()

	packageSize := getArchiveSize(zipReader)
	err = checkFreeSize(mountDir, packageSize)
	if err != nil {
		return "", err
	}
	targetArchiveDir := filepath.Join(targetDir, strings.TrimSuffix(filepath.Base(archivePath), ".zip"))
	os.MkdirAll(targetArchiveDir, os.ModePerm)
	serviceFile, err := decompressZipArchiveAndReturnService(zipReader, targetArchiveDir, interactiveRun, &packageConfig.OverwriteFiles)
	if err != nil {
		return "", err
	}
	return serviceFile, nil
}

// mountPartition mounts the specified partition to the mount directory using guestmount.
// It sends any errors encountered to the provided error channel.
func mountPartition(targetImageName string, partitionNumber int, mountDir string, errChan chan string) chan string {
	log.Printf("Mounting partition to %s", mountDir)
	// TODO set userId in config??
	cmd := fmt.Sprintf("guestmount -a %s -m /dev/sda%d -o uid=%d -o gid=%d --rw %s --no-fork", targetImageName, partitionNumber, unix.Getuid(), unix.Getgid(), mountDir)
	_, err := helper.RunCommand(cmd, false)
	if err != nil {
		errChan <- err.Error()
	}
	return errChan
}

// unmount unmounts the specified mount directory using guestunmount.
func unmount(mountDir string) {
	log.Printf("Unmounting partition")
	helper.RunCommand("guestunmount "+mountDir, true)
	waitUntilDirectoryIsEmpty(mountDir, timeout)
}

// getArchiveSize calculates the total uncompressed size of the files in the zip archive.
func getArchiveSize(zipReader *zip.ReadCloser) uint64 {
	packageSize := uint64(0)
	for _, file := range zipReader.File {
		packageSize += file.UncompressedSize64
	}
	return packageSize
}

// checkFreeSize checks if there is enough free space in the mount directory to copy the package.
// It returns an error if there is not enough space.
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

// decompressZipArchiveAndReturnService extracts the files from the zip archive to the target directory.
// It returns a list of service files found in the archive.
func decompressZipArchiveAndReturnService(zipReader *zip.ReadCloser, targetDir string, interactiveRun bool, overwriteFiles *[]string) (string, error) {
	serviceFile := ""

	for _, file := range zipReader.File {
		targetFilePath := filepath.Join(targetDir, file.Name)
		//fmt.Println("unzipping file ", targetFilePath)

		if !helper.IsWithinRootDir(targetDir, targetFilePath) { // Check if the file is within the target directory
			return "", fmt.Errorf("invalid file path")
		}
		if file.FileInfo().IsDir() {
			log.Printf("creating directory %s\n", targetFilePath)
			if err := os.MkdirAll(targetFilePath, os.ModePerm); err != nil {
				return "", err
			}
			continue
		}

		if err := os.MkdirAll(filepath.Dir(targetFilePath), os.ModePerm); err != nil {
			return "", err
		}
		if err := decompressZipFile(targetFilePath, file, interactiveRun, overwriteFiles); err != nil {
			return "", err
		}
		if strings.HasSuffix(file.Name, ".service") {
			if serviceFile != "" {
				return "", fmt.Errorf("multiple service files found in the package archive")
			}
			serviceFile = targetFilePath
		}
	}
	return serviceFile, nil
}

// decompressZipFile extracts a single file from the zip archive to the destination path.
// It returns an error if the file already exists and overwrite is false.
func decompressZipFile(destFilePath string, srcZipFile *zip.File, interactiveRun bool, overwriteFiles *[]string) error {
	log.Printf("Decompressing file %s to %s", srcZipFile.Name, destFilePath)
	// Check if the destination file already exists
	_, err := os.Stat(destFilePath)
	if err == nil {
		if interactiveRun {
			if user.GetUserConfirmation("File: " + destFilePath + " already exists. Do you want to overwrite it?") {
				*overwriteFiles = append(*overwriteFiles, destFilePath)
			} else {
				return fmt.Errorf("file %s already exists and user chose not to overwrite", destFilePath)
			}
		} else {
			if slices.Contains(*overwriteFiles, destFilePath) {
				return fmt.Errorf("file %s already exists and is not marked for overwrite", destFilePath)
			}
		}
	}
	srcFile, err := srcZipFile.Open()
	if err != nil {
		return fmt.Errorf("unable to open file %s: %v", srcZipFile.Name, err)
	}
	defer srcFile.Close()

	if srcZipFile.FileInfo().Mode()&os.ModeSymlink != 0 {
		linkTarget, err := io.ReadAll(srcFile)
		if err != nil {
			return fmt.Errorf("unable to read symlink target for %s: %v", srcZipFile.Name, err)
		}
		err = os.Symlink(string(linkTarget), destFilePath)
		if err != nil {
			return fmt.Errorf("unable to create symlink %s: %v", destFilePath, err)
		}
	} else {
		destFile, err := os.OpenFile(destFilePath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, srcZipFile.Mode())
		if err != nil {
			return fmt.Errorf("unable to create file %s: %v", destFilePath, err)
		}
		defer destFile.Close()

		_, err = io.Copy(destFile, srcFile)
		if err != nil {
			return fmt.Errorf("unable to copy file %s: %v", srcZipFile.Name, err)
		}
	}
	// TODO do we want to change permissions if the file already exists?
	// Set the file permissions
	//err = os.Chmod(destFilePath, fileMode)
	//if err != nil {
	// return fmt.Errorf("unable to set file permissions for %s: %v", destFilePath, err)
	//}
	return nil
}

// waitUntilDirectoryIsPopulated waits until the directory is populated or the timeout is reached.
// It returns an error if the directory is not populated within the timeout period.
func waitUntilDirectoryIsPopulated(dirPath string, timeout time.Duration) error {
	start := time.Now()
	for {
		populated, err := isDirectoryPopulated(dirPath)
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

// waitUntilDirectoryIsEmpty waits until the directory is empty or the timeout is reached.
// Use to make sure directory is unmounted before proceeding.
func waitUntilDirectoryIsEmpty(dirPath string, timeout time.Duration) {
	start := time.Now()
	for {
		populated, err := isDirectoryPopulated(dirPath)
		if err != nil {
			log.Printf("Error checking directory: %v", err)
		}
		if !populated {
			return
		}
		if time.Since(start) > timeout {
			log.Printf("directory %s is not empty within the timeout period. Continuing", dirPath)
		}
		time.Sleep(100 * time.Millisecond) // Adjust the sleep duration as needed
	}
}

// isDirectoryPopulated checks if the given directory is populated.
// It returns true if the directory contains at least one file or subdirectory.
func isDirectoryPopulated(dirPath string) (bool, error) {
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
