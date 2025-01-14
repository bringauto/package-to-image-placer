package package_to_image_placer

import (
	"fmt"
	"github.com/coreos/go-systemd/v22/unit"
	"io"
	"log"
	"os"
	"path/filepath"
	"strings"
)

// AddService adds a service file to the image, update paths in it and activates it
// It returns an error if the service file is missing required fields: ExecStart, Type, User, RestartSec, WorkingDirectory
// or if the service file has a Type other than 'simple' or WantedBy other than 'multi-user.target'
func AddService(serviceFile string, mountDir string, packageDir string, overwrite bool) error {
	log.Printf("Activating service %s", filepath.Base(serviceFile))
	opts, err := parseServiceFile(serviceFile)
	if err != nil {
		return err
	}

	err = checkServiceFileContent(opts)
	if err != nil {
		return err
	}

	err = updatePathsInServiceFile(opts, mountDir, packageDir, serviceFile)
	if err != nil {
		return fmt.Errorf("failed to update paths in service file: %v", err)
	}

	err = writeOptsToFile(serviceFile, opts)
	if err != nil {
		return err
	}

	destPath, err := activateService(mountDir, serviceFile, overwrite)
	if err != nil {
		return err
	}
	fmt.Println("Activated service file:", destPath)
	return nil
}

// activateService copies the service file to the image and creates a symlink to it in the multi-user.target.wants directory
func activateService(mountDir string, serviceFile string, overwrite bool) (string, error) {
	destPath := filepath.Join(mountDir, "etc/systemd/system", filepath.Base(serviceFile))
	symlinkPath := filepath.Join(mountDir, "/etc/systemd/system/multi-user.target.wants", filepath.Base(serviceFile))
	if !overwrite {
		if _, err := os.Lstat(destPath); err == nil {
			return "", fmt.Errorf("service file already exists: '%s'... Use -overwrite to overwrite the service", destPath)
		}
		if _, err := os.Lstat(symlinkPath); err == nil {
			return "", fmt.Errorf("service symlink already exists: '%s'... Use -overwrite to overwrite the symlink", symlinkPath)
		}
	}
	err := copyFile(destPath, serviceFile, 0644)
	if err != nil {
		return "", fmt.Errorf("failed to copy service file: %v", err)
	}
	err = os.Symlink(filepath.Join("..", filepath.Base(serviceFile)), symlinkPath)
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
	return nil
}

var requiredFields = []string{"ExecStart", "Type", "User", "RestartSec", "WorkingDirectory"}

// checkAndParseServiceFileContent checks presence and value of required fields and parses the content.
func checkServiceFileContent(optsMap map[string]unit.UnitOption) error {
	var allFieldsPresent = true
	for _, field := range requiredFields {
		if _, fieldPresent := optsMap[field]; !fieldPresent {
			allFieldsPresent = false
			log.Printf("Required field %s is missing in file %s\n", field, optsMap)
		}
	}

	if optsMap["Type"].Value != "simple" {
		return fmt.Errorf("only services with 'Type=simple' are supported")
	}
	if optsMap["WantedBy"].Value != "multi-user.target" {
		return fmt.Errorf("only services with 'WantedBy=multi-user.target' are supported")
	}

	if !allFieldsPresent {
		return fmt.Errorf("required fields are missing in service file %s", optsMap)
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

	originalExecutable := strings.Trim(splitStringPreserveSubstrings(execStart)[0], "'\"")
	executableWithoutWorkDir := strings.TrimPrefix(originalExecutable, workingDir)

	newWorkDirWithMountDir, err := findExecutableInPath(filepath.Dir(serviceFile), executableWithoutWorkDir, packageDir)
	if err != nil {
		return fmt.Errorf("executable %s not found in package directory %s", executableWithoutWorkDir, err)
	}

	newWorkDir := strings.TrimPrefix(newWorkDirWithMountDir, mountDir) + "/"
	if !strings.HasPrefix(newWorkDir, "/") {
		newWorkDir = "/" + newWorkDir // Make sure to have an absolute path
	}

	newExecutablePath := filepath.Join(newWorkDir, executableWithoutWorkDir)
	newExecStartCommand := strings.ReplaceAll(execStart, originalExecutable, newExecutablePath)
	newExecStartCommand = strings.ReplaceAll(newExecStartCommand, workingDir, newWorkDir)

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
	currentPath := startPath
	for strings.HasPrefix(currentPath, packageDir) {
		potentialPath := filepath.Join(currentPath, executable)
		if _, err := os.Stat(potentialPath); err == nil {
			return currentPath, nil
		}
		currentPath = filepath.Dir(currentPath)
	}
	return "", fmt.Errorf("executable %s not found within package directory %s", executable, packageDir)
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

func parseRequiredOption(serviceFile string) ([]string, error) {
	opts, err := parseServiceFile(serviceFile)
	if err != nil {
		return nil, fmt.Errorf("failed to parse service file %s", serviceFile)
	}
	requiresOpt, exists := opts["Requires"]
	if !exists {
		return nil, nil // no Requires option
	}
	return splitStringPreserveSubstrings(requiresOpt.Value), nil
}
