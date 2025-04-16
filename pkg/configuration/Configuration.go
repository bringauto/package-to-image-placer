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
	InteractiveRun:        false,
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

func ValidateConfiguration() error {
	// TODO rework checks
	if Config.Target == "" {
		return fmt.Errorf("target image path is missing, start with -h to see arguments")
	}
	if Config.Target == Config.Source {
		return fmt.Errorf("source and target image paths are the same")
	}
	if Config.Source == "" && !Config.NoClone {
		return fmt.Errorf("either 'source' or 'no-clone' must be defined, start with -h to see arguments.\n")
	} else if !Config.NoClone && !helper.DoesFileExists(Config.Source) {
		return fmt.Errorf("source image path does not exist\n")
	}
	if Config.NoClone && !helper.DoesFileExists(Config.Target) {
		return fmt.Errorf("target image does not exist\n")
	}

	if !Config.InteractiveRun {
		if len(Config.Packages) == 0 {
			return fmt.Errorf("no packages defined in configuration") // TODO check if packages exist
		}
		for _, pkg := range Config.Packages {
			if !helper.DoesFileExists(pkg.PackagePath) {
				return fmt.Errorf("package %s does not exist", pkg.PackagePath)
			}
		}
		if len(Config.PartitionNumbers) == 0 {
			return fmt.Errorf("no partition numbers defined in configuration")
		}
	}

	// Config.LogPath is there to avoid the error message when the log path is not defined
	if !helper.DoesFileExists(Config.LogPath) && Config.LogPath != "" {
		return fmt.Errorf("log path does not exist")

	}
	return nil
}
