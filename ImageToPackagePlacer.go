package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"
	"package-to-image-placer/pkg/configuration"
	"package-to-image-placer/pkg/helper"
	"package-to-image-placer/pkg/image"
	"package-to-image-placer/pkg/interaction"
)

func main() {
	err := helper.AllDepsInstalled()
	if err != nil {
		log.Printf("Error: %s\n", err)
		os.Exit(1)
	}

	config, err := parseArguments(os.Args[1:])
	if err != nil {
		log.Fatalf("Error parsing arguments: %v", err)
	}
	err = configuration.ValidateConfiguration(config)
	if err != nil {
		log.Fatalf("Configuration validation error: %v", err)
	}

	newConfigFilePath := ""
	if config.InteractiveRun {
		interaction.SetUpCommandline()
		defer interaction.CleanUpCommandLine()
		config.Packages, err = interaction.SelectFilesInDir(config.PackageDir)
		if err != nil {
			log.Printf("Error: %s\n", err)
			return
		}

		imagePath := ""
		if config.NoClone {
			imagePath = config.Target
		} else {
			imagePath = config.Source
		}
		config.PartitionNumbers, err = interaction.SelectPartitions(imagePath)
		if err != nil {
			log.Printf("Error while selecting partitions: %s\n", err)
		}

		if interaction.GetUserConfirmation("\nDo you want to save the configuration?") {
			newConfigFilePath, err = configuration.CreateConfigurationFile(config)
			if err != nil {
				log.Printf("Error: %s\n", err)
			}
		}

	}

	log.Printf("Packages: %v\nwill be copied to partitions: %v\n", config.Packages, config.PartitionNumbers)

	if !config.NoClone {
		if helper.DoesFileExists(config.Target) {
			askUser := fmt.Sprintf("File %s already exists. Do you want to delete it?", config.Target)
			if config.InteractiveRun && !interaction.GetUserConfirmation(askUser) {
				log.Fatalf("file already exists and user chose not to delete it")
			}
			if err := os.Remove(config.Target); err != nil {
				log.Fatalf("unable to delete existing file: %s", err)
			}
		}
		err := image.CloneImage(config.Source, config.Target)
		if err != nil {
			log.Printf("Error: %s\n", err)
			return
		}
	}

	err = image.CopyPackageToImagePartitions(&config)
	if err != nil {
		log.Printf("Error: %s\n", err)
		return
	}

	log.Printf("All packages copied successfully\n")

	if newConfigFilePath != "" {
		err = configuration.UpdateConfigurationFile(config, newConfigFilePath)
		if err != nil {
			log.Printf("Error: %s\n", err)
		}
	}
}

func parseArguments(args []string) (configuration.Configuration, error) {
	var config configuration.Configuration
	flags := flag.NewFlagSet("package-to-image-placer", flag.ContinueOnError)

	configFile := flags.String("config", "", "Path to configuration file (non-interactive mode)")
	targetImage := flags.String("target", "", "Target image path (will be created).")
	sourceImage := flags.String("source", "", "Source image")
	noClone := flags.Bool("no-clone", false, "Do not clone source image. Target image must exist. If operation is not successful, may cause damage the image")
	overwrite := flags.Bool("overwrite", false, "Overwrite files in target image")
	targetDirectory := flags.String("target-dir", "", "Target directory in image (non-interactive mode)")
	packageDirectory := flags.String("package-dir", "./", "Default package directory, from which package finder starts (interactive mode)")
	showUsage := flags.Bool("h", false, "Show usage")

	err := flags.Parse(args)
	if err != nil {
		return config, err
	}

	if *showUsage {
		fmt.Printf("Usage:\n" +
			"Interactive: \t\tpackage-to-image-placer -target <target_image> [ -source <src_image> | -no-clone ] [ opts... ]\n" +
			"Non-interactive: \tpackage-to-image-placer -config <config_file> [ <override-opts> ]\n")
		flags.PrintDefaults()
		os.Exit(0)
	}

	interactiveRun := true
	if *configFile != "" {
		interactiveRun = false
		file, err := os.Open(*configFile)
		if err != nil {
			return config, fmt.Errorf("error opening config file: %v", err)
		}
		defer file.Close()

		decoder := json.NewDecoder(file)
		err = decoder.Decode(&config)
		if err != nil {
			return config, fmt.Errorf("error decoding config file: %v", err)
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

	config.InteractiveRun = interactiveRun
	return config, nil
}
