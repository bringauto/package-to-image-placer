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
const mountMaxRetries = 3
const mountRetryDelay = 2 * time.Second

// CopyPackagesToImagePartitions copies the specified packages to the specified partitions in the configuration.
// It iterates over each partition and package, calling MountPartitionAndCopyPackages for each combination.
func CopyPackagesToImagePartitions() error {
	for _, partition := range configuration.Config.PartitionNumbers {
		log.Printf("Copying to partition: %d\n", partition)
		err := MountPartitionAndCopyPackages(partition, partition == configuration.Config.PartitionNumbers[0])
		if err != nil {
			return err
		}
	}
	return nil
}

// CopyPackageActivateService copies the package to the target directory and activates any service files found in the package.
// It also handles user interaction for enabling services and setting service name suffixes.
func CopyPackageActivateService(mountDir string, packageConfig *configuration.PackageConfig, firstPartition bool) error {
	var targetDirectoryFullPath string
	var err error
	if configuration.Config.InteractiveRun && packageConfig.IsStandardPackage && firstPartition {
		targetDirectoryFullPath, err = user.SelectTargetDirectory(mountDir, mountDir, packageConfig.PackagePath)
		if err != nil {
			return err
		}
		relPath, err := filepath.Rel(mountDir, targetDirectoryFullPath)
		if err != nil {
			return fmt.Errorf("failed to determine relative path for target directory: %v", err)
		}
		packageConfig.TargetDirectory = filepath.Join(relPath, "") + string(os.PathSeparator)
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

	serviceFile, err := handleArchive(packageConfig, mountDir, targetDirectoryFullPath)
	if err != nil {
		return err
	}

	// Configuration packages are not allowed to have services
	if !packageConfig.IsStandardPackage {
		return nil
	}

	if serviceFile == "" {
		log.Printf("No service file found in the package: %s\n", packageConfig.PackagePath)

		// check if the package has disabled services
		if packageConfig.EnableServices {
			return fmt.Errorf("package %s has no service file, but services are enabled", packageConfig.PackagePath)
		}
		return nil
	}

	if configuration.Config.InteractiveRun && firstPartition {
		packageConfig.EnableServices = user.GetUserConfirmation("Do you want to enable services for package " + packageConfig.PackagePath + "?")
		if packageConfig.EnableServices {
			packageConfig.ServiceNameSuffix, err = user.ReadStringFromUser("Enter service name suffix (leave empty for none): ")
			if err != nil {
				return fmt.Errorf("error reading service name suffix: %v", err)
			}
		}
	}

	if packageConfig.EnableServices {
		// Service name suffix should not start with a hyphen
		if strings.HasPrefix(packageConfig.ServiceNameSuffix, "-") {
			return fmt.Errorf("service name suffix should not start with a hyphen")
		}
		err = service.AddService(serviceFile, mountDir, targetDirectoryFullPath, packageConfig)
		if err != nil {
			return fmt.Errorf("error while activating service: %v", err)
		}
	}
	return nil
}

// MountPartitionAndCopyPackages mounts the specified partition, copies the package to it, and activates any service files found in the package.
// It handles signals for unmounting the partition and ensures the directory is populated before proceeding.
func MountPartitionAndCopyPackages(partitionNumber int, firstPartition bool) error {
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
		errChan = mountPartition(configuration.Config.Target, partitionNumber, mountDir, errChan)
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

	for i := range configuration.Config.Packages {
		configuration.Config.Packages[i].IsStandardPackage = true
		err = CopyPackageActivateService(mountDir, &configuration.Config.Packages[i], firstPartition)
		if err != nil {
			return fmt.Errorf("error while copying package: %v", err)
		}
	}
	// tmpPackage := configuration.PackageConfig{EnableServices: false, ServiceNameSuffix: "", TargetDirectory: "", IsStandardPackage: false}
	for i := range configuration.Config.ConfigurationPackages {
		tmpPackage := configuration.PackageConfig{EnableServices: false, ServiceNameSuffix: "", TargetDirectory: "", IsStandardPackage: false}
		tmpPackage.PackagePath = configuration.Config.ConfigurationPackages[i].PackagePath
		tmpPackage.OverwriteFiles = configuration.Config.ConfigurationPackages[i].OverwriteFiles
		err = CopyPackageActivateService(mountDir, &tmpPackage, firstPartition)
		if err != nil {
			return fmt.Errorf("error while copying configuration package: %v", err)
		}
		configuration.Config.ConfigurationPackages[i].OverwriteFiles = tmpPackage.OverwriteFiles
	}
	return nil
}

// handleArchive handles the extraction of the archive file to the target directory.
// It checks for sufficient free space and returns a service file if found.
func handleArchive(packageConfig *configuration.PackageConfig, mountDir string, targetDir string) (string, error) {
	archivePath := packageConfig.PackagePath
	zipReader, err := zip.OpenReader(archivePath)
	if err != nil {
		return "", fmt.Errorf("failed to open zip file: %v", err)
	}
	defer zipReader.Close()

	err = findAllFilesInZip(&zipReader.Reader, packageConfig.OverwriteFiles)
	if err != nil {
		return "", err
	}

	packageSize := getArchiveSize(zipReader)
	err = checkFreeSize(mountDir, packageSize)
	if err != nil {
		return "", err
	}

	targetArchiveDir := helper.GetTargetArchiveDirName(targetDir, archivePath, packageConfig.IsStandardPackage)

	os.MkdirAll(targetArchiveDir, os.ModePerm)
	serviceFile, err := decompressZipArchiveAndReturnService(zipReader, targetArchiveDir, mountDir, packageConfig)
	if err != nil {
		return "", err
	}
	return serviceFile, nil
}

// mountPartition mounts the specified partition to the mount directory using guestmount.
// It sends any errors encountered to the provided error channel.
func mountPartition(targetImageName string, partitionNumber int, mountDir string, errChan chan string) chan string {
	log.Printf("Mounting partition to %s", mountDir)
	var err error
	for range mountMaxRetries {
		cmd := fmt.Sprintf("guestmount -a %s -m /dev/sda%d -o uid=%d -o gid=%d --rw %s --no-fork", targetImageName, partitionNumber, unix.Getuid(), unix.Getgid(), mountDir)
		_, err = helper.RunCommand(cmd, false)
		if err == nil {
			break
		}
		log.Printf("Error mounting partition %d: %v. Retrying in %v...", partitionNumber, err, mountRetryDelay)
		time.Sleep(mountRetryDelay)
	}

	if err != nil {
		errChan <- err.Error()
	}
	return errChan
}

// unmount unmounts the specified mount directory using guestunmount.
func unmount(mountDir string) {
	syscall.Sync()
	log.Printf("Unmounting partition")
	helper.RunCommand("guestunmount "+mountDir, true)

	waitUntilDirectoryIsUnmounted(mountDir, timeout)
	time.Sleep(time.Second) // Give it a moment to clear
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

// findAllFilesInZip checks if all specified files exist in the zip archive.
// It returns an error if any of the files are not found.
func findAllFilesInZip(zipReader *zip.Reader, targetFileNames []string) error {
	fileMap := make(map[string]*zip.File, len(zipReader.File))
	for _, file := range zipReader.File {
		if !file.FileInfo().IsDir() {
			fileMap["/"+file.Name] = file
		}
	}

	for _, targetFileName := range targetFileNames {
		if _, exists := fileMap[targetFileName]; !exists {
			return fmt.Errorf("file %s not found in the zip archive %s", targetFileName, zipReader.Comment)
		}
	}
	return nil
}

// decompressZipArchiveAndReturnService extracts the files from the zip archive to the target directory.
// It returns a list of service files found in the archive.
func decompressZipArchiveAndReturnService(zipReader *zip.ReadCloser, targetDir string, mountDir string, packageConfig *configuration.PackageConfig) (string, error) {
	serviceFile := ""

	for _, file := range zipReader.File {
		targetFilePath := filepath.Join(targetDir, file.Name)

		if !helper.IsWithinRootDir(targetDir, targetFilePath) {
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
		if err := decompressZipFile(targetFilePath, file, mountDir, packageConfig); err != nil {
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
func decompressZipFile(destFilePath string, srcZipFile *zip.File, mountDir string, packageConfig *configuration.PackageConfig) error {
	log.Printf("Decompressing file %s to %s", srcZipFile.Name, destFilePath)
	// Check if the destination file already exists
	_, err := os.Stat(destFilePath)
	if err == nil {
		destFilePathInPackage := helper.RemoveMountDirAndPackageName(destFilePath, mountDir, packageConfig.TargetDirectory, packageConfig.PackagePath)
		if configuration.Config.InteractiveRun {
			if user.GetUserConfirmation("File: " + destFilePathInPackage + " already exists. Do you want to overwrite it?") {
				packageConfig.OverwriteFiles = append(packageConfig.OverwriteFiles, destFilePathInPackage)
			} else {
				return fmt.Errorf("file %s already exists and user chose not to overwrite", destFilePathInPackage)
			}
		}
		if slices.Contains(packageConfig.OverwriteFiles, destFilePathInPackage) {
			os.Remove(destFilePath)
			log.Printf("File %s already exists and is marked for overwrite", destFilePathInPackage)
		} else {
			return fmt.Errorf("file %s already exists and is not marked for overwrite", destFilePathInPackage)
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

// waitUntilDirectoryIsUnmounted waits until the directory is empty or the timeout is reached.
// Use to make sure directory is unmounted before proceeding.
func waitUntilDirectoryIsUnmounted(mountDir string, timeout time.Duration) {
	start := time.Now()
	log.Print("Waiting for directory to be empty...")
	for {
		// Check if the mount dir exists
		if _, err := os.Stat(mountDir); os.IsNotExist(err) {
			log.Printf("Directory %s does not exist, assuming unmounted", mountDir)
			return
		}
		ls_output, err := helper.RunCommand("ls \""+mountDir+"\"", false)
		if err != nil {
			return
		}

		if strings.TrimSpace(ls_output) == "" {
			return
		}

		if time.Since(start) > timeout {
			log.Printf("directory %s is not empty within the timeout period. Continuing", mountDir)
			return
		}
		log.Print("Directory is not empty, waiting...")
		time.Sleep(1 * time.Second) // Adjust the sleep duration as needed
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
