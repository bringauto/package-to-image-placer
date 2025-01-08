package package_to_image_placer

import (
	"fmt"
	"github.com/coreos/go-systemd/unit"
	"io"
	"log"
	"os"
	"package-to-image-placer/pkg/interaction"
	"path/filepath"
	"strings"
)

func AddService(serviceFiles []string, mountDir string, packageDir string, overwrite bool) error {
	fmt.Println("Found service files:", strings.Join(serviceFiles, ", "))
	for _, serviceFile := range serviceFiles {
		if !interaction.GetUserConfirmation(fmt.Sprintf("Do you want to add %s service?", filepath.Base(serviceFile))) {
			continue
		}
		opts, err := checkAndParseServiceFileContent(serviceFile)
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
	}
	return nil
}

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

func checkAndParseServiceFileContent(serviceFile string) (map[string]unit.UnitOption, error) {
	file, err := os.Open(serviceFile)
	if err != nil {
		return nil, fmt.Errorf("unable to open service file: %v", err)
	}
	defer file.Close()

	opts, err := unit.Deserialize(file)
	if err != nil {
		return nil, fmt.Errorf("error parsing service file: %v", err)
	}

	optsMap := make(map[string]unit.UnitOption)
	for _, opt := range opts {
		optsMap[opt.Name] = *opt
	}

	var allFieldsPresent = true
	for _, field := range requiredFields {
		if _, fieldPresent := optsMap[field]; !fieldPresent {
			allFieldsPresent = false
			log.Printf("Required field %s is missing in file %s\n", field, serviceFile)
		}
	}

	if optsMap["Type"].Value != "simple" {
		return nil, fmt.Errorf("only services with 'Type=simple' are supported")
	}
	if optsMap["WantedBy"].Value != "multi-user.target" {
		return nil, fmt.Errorf("only services with 'WantedBy=multi-user.target' are supported")
	}

	if !allFieldsPresent {
		return nil, fmt.Errorf("required fields are missing in service file %s", serviceFile)
	}

	return optsMap, nil
}

func updatePathsInServiceFile(optsMap map[string]unit.UnitOption, mountDir, packageDir, serviceFile string) error {
	workingDirOpt := optsMap["WorkingDirectory"]
	workingDir := workingDirOpt.Value
	execOpt := optsMap["ExecStart"]
	execStart := execOpt.Value

	// Replaces all occurrences of workingDir with packageDir
	//updatedExecStart := strings.ReplaceAll(execStart, workingDir, sysPackagePath)
	originalExecutable := strings.Trim(splitStringPreserveSubstrings(execStart)[0], "'\"")
	executableWithoutWorkDir := strings.TrimPrefix(originalExecutable, workingDir)

	newWorkDir, err := findExecutableInPath(filepath.Dir(serviceFile), executableWithoutWorkDir, packageDir)
	if err != nil {
		return fmt.Errorf("executable %s not found in package directory %s", executableWithoutWorkDir, err)
	}

	sysPackagePath := strings.TrimPrefix(newWorkDir, mountDir) + "/"
	if !strings.HasPrefix(sysPackagePath, "/") {
		sysPackagePath = "/" + sysPackagePath // Make sure to have an absolute path
	}

	newExecutablePath := filepath.Join(sysPackagePath, executableWithoutWorkDir)
	newExecStartCommand := strings.ReplaceAll(execStart, originalExecutable, newExecutablePath)

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

func findExecutableInPath(startPath, executable, packageDir string) (string, error) {
	currentPath := startPath
	for strings.HasPrefix(currentPath, packageDir) {
		potentialPath := filepath.Join(currentPath, executable)
		if _, err := os.Stat(potentialPath); err == nil {
			return potentialPath, nil
		}
		currentPath = filepath.Dir(currentPath)
	}
	return "", fmt.Errorf("executable %s not found within package directory %s", executable, packageDir)
}

func createUnitOptionsSlice(optsMap map[string]unit.UnitOption) []*unit.UnitOption {
	var unitOptions []*unit.UnitOption
	for _, opt := range optsMap {
		optCopy := opt // create a copy to get a unique address
		unitOptions = append(unitOptions, &optCopy)
	}
	return unitOptions
}

func FindServiceFiles(folderPath string) ([]string, error) {
	var serviceFiles []string

	err := filepath.Walk(folderPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() && strings.HasSuffix(info.Name(), ".service") {
			serviceFiles = append(serviceFiles, path)
		}
		return nil
	})

	if err != nil {
		return nil, fmt.Errorf("error walking the path %s: %v", folderPath, err)
	}
	return serviceFiles, nil
}
