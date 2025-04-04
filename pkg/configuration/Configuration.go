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
}

type ConfigurationPackage struct {
	PackagePath    string   `json:"path"`
	OverwriteFiles []string `json:"overwrite-files"`
}

type Configuration struct {
	Source                string                 `json:"source"`
	Target                string                 `json:"target"`
	NoClone               bool                   `json:"no-clone"`
	Packages              []PackageConfig        `json:"packages"`
	ConfigurationPackages []ConfigurationPackage `json:"configuration-packages"`
	PartitionNumbers      []int                  `json:"partition-numbers"`
	Overwrite             bool                   `json:"overwrite"`
	LogPath               string                 `json:"log-path"`
	InteractiveRun        bool                   `json:"-"` // Ignored by JSON
	PackageDir            string                 `json:"-"` // Ignored by JSON
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

func ValidateConfiguration(config Configuration) error {
	// TODO rework checks
	if config.Target == "" {
		return fmt.Errorf("target image path is missing, start with -h to see arguments")
	}
	if config.Target == config.Source {
		return fmt.Errorf("source and target image paths are the same")
	}
	if config.Source == "" && !config.NoClone {
		return fmt.Errorf("either 'source' or 'no-clone' must be defined, start with -h to see arguments.\n")
	} else if !config.NoClone && !helper.DoesFileExists(config.Source) {
		return fmt.Errorf("source image path does not exist\n")
	}
	if config.NoClone && !helper.DoesFileExists(config.Target) {
		return fmt.Errorf("target image does not exist\n")
	}

	if !config.InteractiveRun {
		if len(config.Packages) == 0 {
			return fmt.Errorf("no packages defined in configuration") // TODO check if packages exist
		}
		for _, pkg := range config.Packages {
			if !helper.DoesFileExists(pkg.PackagePath) {
				return fmt.Errorf("package %s does not exist", pkg.PackagePath)
			}
		}
		if len(config.PartitionNumbers) == 0 {
			return fmt.Errorf("no partition numbers defined in configuration")
		}
	}

	// config.LogPath is there to avoid the error message when the log path is not defined
	if !helper.DoesFileExists(config.LogPath) && config.LogPath != "" {
		return fmt.Errorf("log path does not exist")

	}
	return nil
}
