package package_to_image_placer

type Configuration struct {
	Source           string   `json:"source"`
	Target           string   `json:"target"`
	Overwrite        bool     `json:"overwrite"`
	NoClone          bool     `json:"no-clone"`
	Packages         []string `json:"packages"`
	PartitionNumbers []int    `json:"partition-numbers"`
	TargetDirectory  string   `json:"target-directory"`
	ServiceNames     []string `json:"service-names"`
}
