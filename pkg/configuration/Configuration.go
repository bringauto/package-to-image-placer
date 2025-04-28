package configuration

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"package-to-image-placer/pkg/helper"
	"package-to-image-placer/pkg/user"
	"path/filepath"
)

type PackageConfig struct {
	PackagePath       string   `json:"package-path"`
	EnableServices    bool     `json:"enable-services"`
	ServiceNameSuffix string   `json:"service-name-suffix"`
	TargetDirectory   string   `json:"target-directory"`
	OverwriteFiles    []string `json:"overwrite-files"`
	IsStandardPackage bool     `json:"-"`
}

type ConfigurationPackage struct {
	PackagePath    string   `json:"package-path"`
	OverwriteFiles []string `json:"overwrite-files"`
}

type Configuration struct {
	Source                string                 `json:"source"`
	Target                string                 `json:"target"`
	NoClone               bool                   `json:"no-clone"`
	Packages              []PackageConfig        `json:"packages"`
	ConfigurationPackages []ConfigurationPackage `json:"configuration-packages"`
	PartitionNumbers      []int                  `json:"partition-numbers"`
	LogPath               string                 `json:"log-path"`
	InteractiveRun        bool                   `json:"-"` // Ignored by JSON
	PackageDir            string                 `json:"-"` // Ignored by JSON
	ConfigFile            string                 `json:"-"` // Ignored by JSON
}

// Global variable to hold the configuration
// This is a workaround to avoid passing the configuration around
var Config = Configuration{
	Source:                "",
	Target:                "",
	NoClone:               false,
	Packages:              []PackageConfig{},
	ConfigurationPackages: []ConfigurationPackage{},
	PartitionNumbers:      []int{},
	LogPath:               "",
	InteractiveRun:        true,
	PackageDir:            "./",
	ConfigFile:            "",
}

// CreateConfigurationFile Creates a file and places the configuration in form of a JSON string
func CreateConfigurationFile(config Configuration) (string, error) {
	var file *os.File
	var path string
	var err error

	for {
		path, err = user.ReadStringFromUser("Enter path to save configuration file: ")
		if err != nil {
			return "", err
		}

		file, err = os.Create(path)
		if err != nil {
			if os.IsNotExist(err) {
				dir := filepath.Dir(path)
				fmt.Printf("Error: Directory %s does not exist. Please create the directory and try again.\n", dir)
				continue
			}
			fmt.Printf("Error creating file: %v. Please try again.\n", err)
			continue
		}
		defer file.Close()
		break
	}

	encoder := json.NewEncoder(file)
	encoder.SetIndent("", "\t")
	err = encoder.Encode(config)
	if err != nil {
		return "", err
	}
	return path, nil
}

// UpdateConfigurationFile Updates the configuration file with the given path
func UpdateConfigurationFile(config Configuration, path string) error {
	log.Printf("Updating configuration file %s\n", path)
	file, err := os.Create(path)
	if err != nil {
		return err
	}
	defer file.Close()

	encoder := json.NewEncoder(file)
	encoder.SetIndent("", "\t")
	err = encoder.Encode(config)
	if err != nil {
		return err
	}
	return nil
}

// LoadConfigurationFile Loads the configuration file from the given path
func ValidateConfiguration() error {
	if err := validatePaths(); err != nil {
		return err
	}
	if err := validateSourceAndTarget(); err != nil {
		return err
	}
	if err := validatePackagesAndPartitions(); err != nil {
		return err
	}
	if err := validateLogPath(); err != nil {
		return err
	}
	return nil
}

// LoadConfigurationFile Loads the configuration file from the given path
func validatePaths() error {
	if Config.Target == "" {
		return fmt.Errorf("target image path is missing, start with -h to see arguments")
	}
	if Config.Target == Config.Source {
		return fmt.Errorf("source and target image paths are the same")
	}
	return nil
}

// validateSourceAndTarget validates the source and target paths
func validateSourceAndTarget() error {
	if Config.Source != "" && Config.NoClone {
		return fmt.Errorf("source image and no-clone are mutually exclusive")
	}
	if Config.Source == "" && !Config.NoClone {
		return fmt.Errorf("either 'source' or 'no-clone' must be defined, start with -h to see arguments")
	}
	if Config.Source != "" && !helper.DoesFileExists(Config.Source) {
		return fmt.Errorf("source image path: %s does not exist", Config.Source)
	}
	if Config.NoClone && !helper.DoesFileExists(Config.Target) {
		return fmt.Errorf("target image path: %s does not exist", Config.Target)
	}
	return nil
}

// validatePackagesAndPartitions validates the packages and partitions
func validatePackagesAndPartitions() error {
	if !Config.InteractiveRun {
		if len(Config.Packages) == 0 && len(Config.ConfigurationPackages) == 0 {
			return fmt.Errorf("no packages defined in configuration")
		}
		for _, pkg := range Config.Packages {
			if !helper.DoesFileExists(pkg.PackagePath) {
				return fmt.Errorf("package %s does not exist", pkg.PackagePath)
			}
		}
		for _, pkg := range Config.ConfigurationPackages {
			if !helper.DoesFileExists(pkg.PackagePath) {
				return fmt.Errorf("configuration package %s does not exist", pkg.PackagePath)
			}
		}

		if len(Config.PartitionNumbers) == 0 {
			return fmt.Errorf("no partition numbers defined in configuration")
		}
	}
	return nil
}

// validateLogPath validates the log path
func validateLogPath() error {
	if Config.LogPath != "" && !helper.DoesFileExists(Config.LogPath) {
		return fmt.Errorf("log path does not exist")
	}
	return nil
}

// convertOneRelativePathToWorkingDir converts a relative path to an absolute path
func convertOneRelativePathToWorkingDir(path string) string {
	// Add location of configuration file to the path
	if path == "" || filepath.IsAbs(path) {
		return path
	}
	configFileBasePath := filepath.Dir(Config.ConfigFile)

	return filepath.Join(configFileBasePath, path)

}

// ConvertRelativePathsToWorkingDir converts all relative paths in the configuration to absolute paths
func ConvertRelativePathsToWorkingDir() {
	// Convert relative paths from configuration to relative paths to the working directory
	Config.Source = convertOneRelativePathToWorkingDir(Config.Source)
	Config.Target = convertOneRelativePathToWorkingDir(Config.Target)
	Config.LogPath = convertOneRelativePathToWorkingDir(Config.LogPath)
	for i, pkg := range Config.Packages {
		Config.Packages[i].PackagePath = convertOneRelativePathToWorkingDir(pkg.PackagePath)
	}
	for i, pkg := range Config.ConfigurationPackages {
		Config.ConfigurationPackages[i].PackagePath = convertOneRelativePathToWorkingDir(pkg.PackagePath)
	}
}
