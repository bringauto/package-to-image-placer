package service

import (
	"fmt"
	"io"
	"log"
	"os"
	"package-to-image-placer/pkg/configuration"
	"package-to-image-placer/pkg/helper"
	"package-to-image-placer/pkg/user"
	"path/filepath"
	"slices"
	"strings"

	"github.com/coreos/go-systemd/v22/unit"
)

// AddService adds the serviceFile to /etc/systemd/system in image
// update paths based on packageDir (removes mountDir prefix) in it and activates the service.
// It returns an error if the service file is missing required fields: ExecStart, Type, User, RestartSec, WorkingDirectory
// or if the service file has a Type other than 'simple' or WantedBy other than 'multi-user.target'
// serviceFile: full path to the service file in the target image
// mountDir: path to the target image mount point
// packageDir: path to the package directory in the target image
func AddService(serviceFile string, mountDir string, packageDir string, packageConfig *configuration.PackageConfig) error {
	log.Printf("Activating service %s", filepath.Base(serviceFile))
	opts, err := parseServiceFile(serviceFile)
	if err != nil {
		return err
	}

	err = checkServiceFileContent(opts)
	if err != nil {
		return fmt.Errorf("invalid service file: %s\n%v", serviceFile, err)
	}

	err = updatePathsInServiceFile(opts, mountDir, packageDir, serviceFile)
	if err != nil {
		return fmt.Errorf("failed to update paths in service file: %v", err)
	}

	err = writeOptsToFile(serviceFile, opts)
	if err != nil {
		return err
	}

	destPath, err := activateService(mountDir, serviceFile, packageConfig)
	if err != nil {
		return err
	}
	fmt.Println("Activated service file:", destPath)
	return nil
}

// checkAndHandleServiceFileOverwrite checks if the file or symlink exists and handles overwriting based on user input or configuration.
func checkAndHandleServiceFileOverwrite(destPath string, symlinkPath string, serviceFile string, mountDir string, packageConfig *configuration.PackageConfig) error {
	destFilePathInPackage := helper.RemoveMountDirAndPackageName(serviceFile, mountDir, packageConfig.TargetDirectory, packageConfig.PackagePath)

	if (helper.DoesFileExists(destPath) || helper.DoesFileExists(symlinkPath)) && !slices.Contains(packageConfig.OverwriteFiles, destFilePathInPackage) {
		if configuration.Config.InteractiveRun {
			if user.GetUserConfirmation("Service file " + destFilePathInPackage + " already exists. Do you want to overwrite it?") {
				packageConfig.OverwriteFiles = append(packageConfig.OverwriteFiles, destFilePathInPackage)

			} else {
				return fmt.Errorf("file %s already exists and user chose not to overwrite it", destFilePathInPackage)
			}
		} else if !slices.Contains(packageConfig.OverwriteFiles, destFilePathInPackage) {
			return fmt.Errorf("file %s already exists and is not in the overwrite list", destFilePathInPackage)
		}
	}
	return nil
}

// activateService copies the service file to the image and creates a symlink to it in the multi-user.target.wants directory
func activateService(mountDir string, serviceFile string, packageConfig *configuration.PackageConfig) (string, error) {
	serviceDestFile := serviceFile
	if packageConfig.ServiceNameSuffix != "" {
		serviceDestFile = strings.TrimSuffix(serviceFile, ".service") + "-" + packageConfig.ServiceNameSuffix + ".service"
	}

	destPath := filepath.Join(mountDir, "/etc/systemd/system", filepath.Base(serviceDestFile))
	symlinkPath := filepath.Join(mountDir, "/etc/systemd/system/multi-user.target.wants", filepath.Base(serviceDestFile))

	err := checkAndHandleServiceFileOverwrite(destPath, symlinkPath, serviceFile, mountDir, packageConfig)
	if err != nil {
		return "", err
	}

	err = helper.CopyFile(destPath, serviceFile, 0644)
	if err != nil {
		return "", fmt.Errorf("failed to copy service file: %v", err)
	}
	err = os.Symlink(filepath.Join("..", filepath.Base(serviceDestFile)), symlinkPath)
	if err != nil && !os.IsExist(err) {
		return "", fmt.Errorf("failed to create symlink: %v", err)
	}
	return destPath, nil
}

// writeOptsToFile writes the updated options to the service file
func writeOptsToFile(serviceFile string, opts map[string]unit.UnitOption) error {
	file, err := os.Create(serviceFile)
	if err != nil {
		return fmt.Errorf("failed to open service file for writing: %v", err)
	}
	defer file.Close()

	reader := unit.Serialize(createUnitOptionsSlice(opts))
	_, err = io.Copy(file, reader)
	if err != nil {
		return fmt.Errorf("failed to write updated options to service file: %v", err)
	}
	if err := file.Sync(); err != nil {
		return fmt.Errorf("failed to flush data to service file: %v", err)
	}
	return nil
}

var requiredFields = []string{"ExecStart", "Type", "User", "RestartSec", "WorkingDirectory", "WantedBy"}

// checkAndParseServiceFileContent checks presence and value of required fields and parses the content.
func checkServiceFileContent(optsMap map[string]unit.UnitOption) error {
	var missingFields []string
	for _, field := range requiredFields {
		if _, fieldPresent := optsMap[field]; !fieldPresent {
			missingFields = append(missingFields, field)
		}
	}
	if len(missingFields) > 0 {
		return fmt.Errorf("missing required fields: %v", strings.Join(missingFields, ", "))
	}

	if optsMap["Type"].Value != "simple" {
		return fmt.Errorf("only services with 'Type=simple' are supported")
	}
	if optsMap["WantedBy"].Value != "multi-user.target" {
		return fmt.Errorf("only services with 'WantedBy=multi-user.target' are supported")
	}

	return nil
}

func parseServiceFile(serviceFile string) (map[string]unit.UnitOption, error) {
	file, err := os.Open(serviceFile)
	if err != nil {
		return nil, fmt.Errorf("unable to open service file: %v", err)
	}
	defer file.Close()

	opts, err := unit.DeserializeOptions(file)
	if err != nil {
		if err.Error() == "unexpected newline encountered while parsing option name" {
			log.Printf("WARNING: Service file %s has an unexpected newline. This may cause issues.\n", serviceFile)
		} else {
			return nil, fmt.Errorf("error parsing service file: %v", err)
		}
	}

	optsMap := make(map[string]unit.UnitOption)
	for _, opt := range opts {
		optsMap[opt.Name] = *opt
	}
	return optsMap, nil
}

// updatePathsInServiceFile updates the paths in the service file to point to the package directory
// It updates working directory and ExecStart path in the service file to point to the package directory based on the original paths.
// It returns an error if the executable is not found in the package directory
func updatePathsInServiceFile(optsMap map[string]unit.UnitOption, mountDir, packageDir, serviceFile string) error {
	log.Printf("Updating paths in service file %s", serviceFile)
	workingDirOpt := optsMap["WorkingDirectory"]
	workingDir := workingDirOpt.Value
	execOpt := optsMap["ExecStart"]
	execStart := execOpt.Value

	execStartStrings := helper.SplitStringPreserveSubstrings(execStart)
	originalExecutable := strings.Trim(execStartStrings[0], "'\"")
	executableWithoutWorkDir := strings.TrimPrefix(originalExecutable, workingDir)

	newWorkDirWithMountDir, err := findExecutableInPath(filepath.Dir(serviceFile), executableWithoutWorkDir, packageDir)
	if err != nil {
		return fmt.Errorf("unable to find executable %s: %s", executableWithoutWorkDir, err)
	}

	newWorkDir := strings.TrimPrefix(newWorkDirWithMountDir, mountDir) + "/"
	if !strings.HasPrefix(newWorkDir, "/") {
		newWorkDir = "/" + newWorkDir // Make sure to have an absolute path
	}

	newExecutablePath := filepath.Join(newWorkDir, executableWithoutWorkDir)
	newExecStartCommand := newExecutablePath
	for i := 1; i < len(execStartStrings); i++ {
		replaced := strings.Replace(execStartStrings[i], workingDir, newWorkDir, 1)
		newExecStartCommand = strings.Join([]string{newExecStartCommand, replaced}, " ")
	}

	log.Printf("Updated ExecStart path from: %s to: %s", execStart, newExecutablePath)

	optsMap["ExecStart"] = unit.UnitOption{
		Section: execOpt.Section,
		Name:    execOpt.Name,
		Value:   newExecStartCommand,
	}
	optsMap["WorkingDirectory"] = unit.UnitOption{
		Section: workingDirOpt.Section,
		Name:    workingDirOpt.Name,
		Value:   newWorkDir,
	}
	return nil
}

// findExecutableInPath searches for the executable in the given path and package directory.
// It returns the path where the executable is found or an error if not found.
// Starting from the given path, it goes up the directory tree until it reaches the package directory.
func findExecutableInPath(startPath, executable, packageDir string) (string, error) {
	var searchInPath func(string, bool) (string, error)
	searchInPath = func(currentPath string, recursive bool) (string, error) {
		if !strings.HasPrefix(currentPath+"/", packageDir) {
			return "", fmt.Errorf("executable %s not found within package directory %s", executable, packageDir)
		}

		// Check if the executable exists in the current directory
		potentialPath := filepath.Join(currentPath, executable)
		if _, err := os.Stat(potentialPath); err == nil {
			return currentPath, nil
		}

		// Recursively search in subdirectories
		if recursive {
			files, err := os.ReadDir(currentPath)
			if err != nil {
				return "", fmt.Errorf("error reading directory %s: %v", currentPath, err)
			}
			for _, file := range files {
				if file.IsDir() {
					foundPath, err := searchInPath(filepath.Join(currentPath, file.Name()), true)
					if err == nil {
						return foundPath, nil
					}
				}
			}
		}

		// Move up one directory and continue searching
		return searchInPath(filepath.Dir(currentPath), false)
	}

	return searchInPath(startPath, true)
}

// createUnitOptionsSlice converts a map of unit options to a slice of unit options.
func createUnitOptionsSlice(optsMap map[string]unit.UnitOption) []*unit.UnitOption {
	var unitOptions []*unit.UnitOption
	for _, opt := range optsMap {
		//optCopy := opt // create a copy to get a unique address
		unitOptions = append(unitOptions, &opt)
	}
	return unitOptions
}

// IsServiceFileInList checks if the service file is listed in the slice.
func IsServiceFileInList(serviceFile string, configServiceFiles []string) bool {
	for _, file := range configServiceFiles {
		if file == filepath.Base(serviceFile) {
			return true
		}
	}
	return false
}

// AreAllServiceFromConfigPresent checks if all service names in the config are present in the service files.
func AreAllServiceFromConfigPresent(serviceFiles []string, configServiceNames []string) bool {
	serviceFilesMap := make(map[string]bool)
	for _, serviceFile := range serviceFiles {
		serviceFilesMap[filepath.Base(serviceFile)] = true
	}

	for _, serviceName := range configServiceNames {
		if !serviceFilesMap[serviceName] {
			return false
		}
	}
	return true
}

// CheckRequiredServicesEnabled checks if the required services of the newly added services are enabled.
func CheckRequiredServicesEnabled(mountDir string, serviceNames []string) error {
	log.Printf("Checking if the required services of the newly added services are enabled.")
	for _, serviceName := range serviceNames {
		servicePath := filepath.Join(mountDir, "etc/systemd/system", serviceName)
		requiredServices, err := parseRequiredOption(servicePath)
		if err != nil {
			return fmt.Errorf("failed to parse service file %s: %s", serviceName, err)
		}
		for _, requiredService := range requiredServices {
			if strings.HasSuffix(requiredService, ".target") {
				if !isTargetEnabled(mountDir, requiredService) {
					return fmt.Errorf("target %s is not enabled", requiredService)
				}
				continue
			}
			serviceEnabled, err := isServiceEnabled(mountDir, filepath.Base(requiredService))
			if err != nil {
				return fmt.Errorf("error checking if service %s is enabled: %v", requiredService, err)
			}
			if !serviceEnabled {
				return fmt.Errorf("service %s is not enabled", requiredService)
			}
		}
	}
	return nil
}

// isServiceEnabled checks if the service is enabled by checking if the symlink exists in the wants or requires directory.
func isServiceEnabled(mountDir, serviceName string) (bool, error) {
	// Define the patterns to search for
	servicePath := filepath.Join(mountDir, "etc/systemd/system")
	wantsPattern := filepath.Join(servicePath, "*.wants", serviceName)
	requiresPattern := filepath.Join(servicePath, "*.requires", serviceName)

	// Search for the service file in "*.wants" directories
	wantsMatches, err := filepath.Glob(wantsPattern)
	if err != nil {
		return false, fmt.Errorf("error searching for service file in wants directories: %v", err)
	}
	if len(wantsMatches) > 0 {
		return true, nil
	}

	// Search for the service file in "*.requires" directories
	requiresMatches, err := filepath.Glob(requiresPattern)
	if err != nil {
		return false, fmt.Errorf("error searching for service file in requires directories: %v", err)
	}
	if len(requiresMatches) > 0 {
		return true, nil
	}

	// Service file not found in any of the directories
	return false, nil
}

// isTargetEnabled checks if the target is enabled by checking if the symlink exists in the target's wants or requires directory.
func isTargetEnabled(mountDir, targetName string) bool {
	servicePath := filepath.Join(mountDir, "etc/systemd/system")
	wantsPath := filepath.Join(servicePath, targetName+".wants")
	if _, err := os.Stat(wantsPath); err == nil {
		return true
	}
	requiresPath := filepath.Join(servicePath, targetName+".requires")
	if _, err := os.Stat(requiresPath); err == nil {
		return true
	}
	return false
}

// parseRequiredOption parses the Requires option from the service file and returns a slice of required services.
func parseRequiredOption(serviceFile string) ([]string, error) {
	opts, err := parseServiceFile(serviceFile)
	if err != nil {
		return nil, fmt.Errorf("failed to parse service file %s", serviceFile)
	}
	requiresOpt, exists := opts["Requires"]
	if !exists {
		return nil, nil // no Requires option
	}
	return helper.SplitStringPreserveSubstrings(requiresOpt.Value), nil
}
