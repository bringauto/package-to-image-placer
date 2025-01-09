package package_to_image_placer

import (
	"encoding/json"
	"os"
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

// Creates a file and places the configuration on form of a JSON string
func CreateConfigurationFile(config Configuration, path string) error {
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
