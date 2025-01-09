package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"
	packagePlacer "package-to-image-placer/pkg"
	"package-to-image-placer/pkg/interaction"
)

func main() {
	err := packagePlacer.AllDepsInstalled()
	if err != nil {
		log.Printf("Error: %s\n", err)
		os.Exit(1)
	}

	config, err := parseArguments(os.Args[1:])
	if err != nil {
		log.Fatalf("Error parsing arguments: %v", err)
	}

	if config.InteractiveRun {
		interaction.SetUpCommandline()
		config.Packages, err = interaction.SelectFilesInDir(config.PackageDir)
		if err != nil {
			log.Printf("Error: %s\n", err)
			return
		}
	}
	if len(config.Packages) == 0 {
		log.Printf("No files selected\n")
		return
	}
	log.Printf("Packages to copy: %s\n", config.Packages)

	if !config.NoClone {
		if packagePlacer.DoesFileExists(config.Target) {
			askUser := fmt.Sprintf("File %s already exists. Do you want to delete it?", config.Target)
			if config.InteractiveRun && !interaction.GetUserConfirmation(askUser) {
				log.Fatalf("file already exists and user chose not to delete it")
			}
			if err := os.Remove(config.Target); err != nil {
				log.Fatalf("unable to delete existing file: %s", err)
			}
		}
		err := packagePlacer.CloneImage(config.Source, config.Target)
		if err != nil {
			log.Printf("Error: %s\n", err)
			return
		}
	}

	if config.InteractiveRun {
		config.PartitionNumbers, err = interaction.SelectPartitions(config.Target)
		if err != nil {
			log.Printf("Error while selecting partitions: %s\n", err)
		}
	}
	err = packagePlacer.CopyPackageToImagePartitions(&config)
	if err != nil {
		log.Printf("Error: %s\n", err)
		return
	}

	if config.InteractiveRun && interaction.GetUserConfirmation("Do you want to save the configuration?") {
		err = packagePlacer.CreateConfigurationFile(config, "config.json")
		if err != nil {
			log.Printf("Error: %s\n", err)
		}
	}
}

func parseArguments(args []string) (packagePlacer.Configuration, error) {
	var config packagePlacer.Configuration
	flags := flag.NewFlagSet("package-to-image-placer", flag.ContinueOnError)

	configFile := flags.String("config", "", "Path to configuration file (non-interactive mode)")
	targetImage := flags.String("target", "", "Target image path")
	sourceImage := flags.String("source", "", "Source image")
	noClone := flags.Bool("no-clone", false, "Do not clone source image. Target image must exist. If operation is not successful, may cause damage the image")
	overwrite := flags.Bool("overwrite", false, "Overwrite files in target image")
	targetDirectory := flags.String("target-dir", "", "Target directory in image")
	packageDirectory := flags.String("package-dir", "./", "Default package directory, from which package finder starts (interactive mode)")
	showUsage := flags.Bool("h", false, "Show usage")

	err := flags.Parse(args)
	if err != nil {
		return config, err
	}

	if *showUsage {
		fmt.Println("Usage:\n\tpackage-to-image-placer -source <source image> -target <target image>\n")
		flags.PrintDefaults()
		return config, fmt.Errorf("usage shown")
	}

	interactiveRun := true
	if *configFile != "" {
		interactiveRun = false
		file, err := os.Open(*configFile)
		if err != nil {
			return config, fmt.Errorf("Error opening config file: %v", err)
		}
		defer file.Close()

		decoder := json.NewDecoder(file)
		err = decoder.Decode(&config)
		if err != nil {
			return config, fmt.Errorf("Error decoding config file: %v", err)
		}
	}

	if *sourceImage != "" {
		config.Source = *sourceImage
	}
	if *targetImage != "" {
		config.Target = *targetImage
	}
	if *overwrite {
		config.Overwrite = *overwrite
	}
	if *targetDirectory != "" {
		config.TargetDirectory = *targetDirectory
	}
	if *noClone {
		config.NoClone = *noClone
	}
	if *packageDirectory != "" {
		config.PackageDir = *packageDirectory
	}

	if config.Target == "" {
		return config, fmt.Errorf("target image path is missing, start with -h to see arguments")
	}
	if config.Target == config.Source {
		return config, fmt.Errorf("source and target image paths are the same")
	}
	if config.Source == "" && !config.NoClone {
		return config, fmt.Errorf("Either 'source' or 'no-clone' must be defined, start with -h to see arguments.\n")
	}
	if config.NoClone && !packagePlacer.DoesFileExists(config.Target) {
		return config, fmt.Errorf("Target image does not exist\n")
	}
	config.InteractiveRun = interactiveRun
	return config, nil
}
