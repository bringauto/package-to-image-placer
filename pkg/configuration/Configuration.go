package configuration

import (
	"encoding/json"
	"fmt"
	"os"
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
		path, err := interaction.ReadStringFromUser()
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
