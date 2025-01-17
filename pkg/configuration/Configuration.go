package configuration

import (
	"encoding/json"
	"fmt"
	"os"
	"package-to-image-placer/pkg/helper"
	"package-to-image-placer/pkg/interaction"
	"path/filepath"
)

type Configuration struct {
	Source           string   `json:"source"`
	Target           string   `json:"target"`
	NoClone          bool     `json:"no-clone"`
	Packages         []string `json:"packages"`
	PartitionNumbers []int    `json:"partition-numbers"`
	TargetDirectory  string   `json:"target-directory"`
	ServiceNames     []string `json:"service-names"`
	Overwrite        bool     `json:"overwrite"`
	InteractiveRun   bool     `json:"-"`
	PackageDir       string   `json:"-"`
}

// CreateConfigurationFile Creates a file and places the configuration in form of a JSON string
func CreateConfigurationFile(config Configuration) error {
	var file *os.File

	for {
		fmt.Printf("Enter path to save configuration file: ")
		path, err := interaction.ReadStringFromUser("Enter path to save configuration file: ")
		if err != nil {
			return err
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
	err := encoder.Encode(config)
	if err != nil {
		return err
	}
	return nil
}

func ValidateConfiguration(config Configuration) error {
	if config.Target == "" {
		return fmt.Errorf("target image path is missing, start with -h to see arguments")
	}
	if config.Target == config.Source {
		return fmt.Errorf("source and target image paths are the same")
	}
	if config.Source == "" && !config.NoClone {
		return fmt.Errorf("Either 'source' or 'no-clone' must be defined, start with -h to see arguments.\n")
	} else if !config.NoClone && !helper.DoesFileExists(config.Source) {
		return fmt.Errorf("Source image path does not exist\n")
	}
	if config.NoClone && !helper.DoesFileExists(config.Target) {
		return fmt.Errorf("Target image does not exist\n")
	}

	if !config.InteractiveRun {
		if len(config.Packages) == 0 {
			return fmt.Errorf("no packages defined in configuration") // TODO check if packages exist
		}
		for _, packagePath := range config.Packages {
			if !helper.DoesFileExists(packagePath) {
				return fmt.Errorf("package %s does not exist", packagePath)
			}
		}
		if len(config.PartitionNumbers) == 0 {
			return fmt.Errorf("no partition numbers defined in configuration")
		}
	}
	return nil
}
