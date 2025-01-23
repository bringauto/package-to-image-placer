package image

import (
	"archive/zip"
	"fmt"
	"golang.org/x/sys/unix"
	"io"
	"log"
	"os"
	"os/signal"
	"package-to-image-placer/pkg/configuration"
	"package-to-image-placer/pkg/helper"
	"package-to-image-placer/pkg/interaction"
	"package-to-image-placer/pkg/service"
	"path/filepath"
	"strings"
	"syscall"
	"time"
)

const timeout = 60 * time.Second

// CopyPackageToImagePartitions copies the specified packages to the specified partitions in the configuration.
// It iterates over each partition and package, calling MountPartitionAndCopyPackage for each combination.
func CopyPackageToImagePartitions(config *configuration.Configuration) error {
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

// MountPartitionAndCopyPackage mounts the specified partition, copies the package to it, and activates any service files found in the package.
// It handles signals for unmounting the partition and ensures the directory is populated before proceeding.
func MountPartitionAndCopyPackage(partitionNumber int, archivePath string, config *configuration.Configuration) error {
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

	var targetDirectoryFullPath string
	if config.InteractiveRun {
		targetDirectoryFullPath, err = interaction.SelectTargetDirectory(mountDir, mountDir)
		if err != nil {
			return err
		}
		config.TargetDirectory = strings.TrimPrefix(targetDirectoryFullPath, mountDir) + "/"
	} else {
		targetDirectoryFullPath = filepath.Join(mountDir, config.TargetDirectory)
		err := os.MkdirAll(targetDirectoryFullPath, os.ModePerm)
		if err != nil {
			return fmt.Errorf("failed to create target directory: %v", err)
		}
	}

	if !helper.IsWithinRootDir(mountDir, targetDirectoryFullPath) {
		return fmt.Errorf("target directory is not within the mounted partition")
	}

	serviceFiles, err := handleArchive(archivePath, mountDir, targetDirectoryFullPath, config.Overwrite)
	if err != nil {
		return err
	}

	if !service.AreAllServiceFromConfigPresent(serviceFiles, config.ServiceNames) {
		baseServiceFiles := make([]string, len(serviceFiles))
		for i, serviceFile := range serviceFiles {
			baseServiceFiles[i] = filepath.Base(serviceFile)
		}
		return fmt.Errorf("not all services from the config are present in the package.\n\tFound services:  %v\n\tConfig services: %v\n", baseServiceFiles, config.ServiceNames)
	}

	for _, serviceFile := range serviceFiles {
		if config.InteractiveRun {
			if interaction.GetUserConfirmation(fmt.Sprintf("\nDo you want to activate service: %s", serviceFile)) {
				err = service.AddService(serviceFile, mountDir, targetDirectoryFullPath, config.Overwrite)
				if err != nil {
					return fmt.Errorf("error while activating service: %v", err)
				}
				config.ServiceNames = append(config.ServiceNames, filepath.Base(serviceFile))
			}
		} else if service.IsServiceFileInList(serviceFile, config.ServiceNames) {
			err = service.AddService(serviceFile, mountDir, targetDirectoryFullPath, config.Overwrite)
			if err != nil {
				return fmt.Errorf("error while activating service: %v", err)
			}
		}
	}

	err = service.CheckRequiredServicesEnabled(mountDir, config.ServiceNames)
	if err != nil {
		return fmt.Errorf("error while checking required services: %v", err)
	}

	return nil
}

// handleArchive handles the extraction of the archive file to the target directory.
// It checks for sufficient free space and returns a list of service files found in the archive.
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

// decompressZipArchive extracts the files from the zip archive to the target directory.
// It returns a list of service files found in the archive.
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

// decompressZipFile extracts a single file from the zip archive to the destination path.
// It returns an error if the file already exists and overwrite is false.
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

	destFile, err := os.OpenFile(destFilePath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, srcZipFile.Mode())
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
