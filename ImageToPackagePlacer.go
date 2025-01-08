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
	configFile := flag.String("config", "", "Path to configuration file (non-interactive mode)")
	targetImage := flag.String("target", "", "Target image path")
	sourceImage := flag.String("source", "", "Source image")
	noClone := flag.Bool("no-clone", false, "Do not clone source image. Target image must exist. If operation is not successful, may cause damage the image")
	overwrite := flag.Bool("overwrite", false, "Overwrite files in target image")
	targetDirectory := flag.String("target-dir", "", "Target directory in image")
	packageDirectory := flag.String("package-dir", "./", "Default package directory, from which package finder starts (interactive mode)")
	showUsage := flag.Bool("h", false, "Show usage")
	flag.Parse()

	if *showUsage {
		println("Usage:\n\tpackage-to-image-placer -source <source image> -target <target image>\n")
		flag.PrintDefaults()
		return
	}

	err := packagePlacer.AllDepsInstalled()
	if err != nil {
		log.Printf("Error: %s\n", err)
		os.Exit(1)
	}

	var config packagePlacer.Configuration
	interactiveRun := true
	if *configFile != "" {
		interactiveRun = false
		file, err := os.Open(*configFile)
		if err != nil {
			log.Fatalf("Error opening config file: %v", err)
		}
		defer file.Close()

		decoder := json.NewDecoder(file)
		err = decoder.Decode(&config)
		if err != nil {
			log.Fatalf("Error decoding config file: %v", err)
		}
	}

	// Override config with command-line arguments if they are provided
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

	if config.Target == "" {
		fmt.Printf("target image path is missing, start with -h to see arguments.\n")
		return
	}
	if config.Source == "" && !config.NoClone {
		fmt.Printf("Either 'source' or 'no-clone' must be defined, start with -h to see arguments.\n")
		return
	}
	if config.NoClone && !packagePlacer.DoesFileExists(config.Target) {
		fmt.Printf("Target image does not exist\n")
		return
	}

	if interactiveRun {
		interaction.SetUpCommandline()
		config.Packages, err = interaction.SelectFilesInDir(*packageDirectory)
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
		err := packagePlacer.CloneImage(config.Source, config.Target)
		if err != nil {
			log.Printf("Error: %s\n", err)
			return
		}
	}

	if interactiveRun {
		config.PartitionNumbers, err = interaction.SelectPartitions(config.Target)
		if err != nil {
			log.Printf("Error while selecting partitions: %s\n", err)
		}
	}
	for _, partition := range config.PartitionNumbers {
		log.Printf("Copying to partition: %d\n", partition)
		for _, archive := range config.Packages {
			// TODO passing config
			err := packagePlacer.UnzipPackageToImage(config.Target, archive, partition, config.TargetDirectory, config.Overwrite)
			if err != nil {
				log.Printf("Error: %s\n", err)
			}
		}
	}

	// TODO create new Config File?
	return
}
