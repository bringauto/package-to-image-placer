package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"package-to-image-placer/pkg/configuration"
	"package-to-image-placer/pkg/helper"
	"package-to-image-placer/pkg/image"
	"package-to-image-placer/pkg/user"
	"path/filepath"
	"strings"
)

func main() {
	config, err := parseArguments(os.Args[1:])
	if err != nil {
		log.Fatalf("Error parsing arguments: %v", err)
	}
	err = configuration.ValidateConfiguration(config)
	if err != nil {
		log.Fatalf("Configuration validation error: %v", err)
	}
	logFile, err := setupLogFile(config.LogPath)
	if err != nil {
		log.Fatalf("Error setting up log file: %v", err)
	}
	defer closeLogFile(logFile)

	err = helper.AllDepsInstalled()
	if err != nil {
		log.Printf("Error: %s\n", err)
		os.Exit(1)
	}

	newConfigFilePath := ""
	if config.InteractiveRun {
		user.SetUpCommandline()
		defer user.CleanUpCommandLine()
		config.Packages, err = user.SelectFilesInDir(config.PackageDir)
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

		err = helper.ValidSourceImage(imagePath)
		if err != nil {
			user.CleanUpCommandLine()
			log.Fatalf("Error: %s\n", err)
		}

		config.PartitionNumbers, err = user.SelectPartitions(imagePath)
		if err != nil {
			user.CleanUpCommandLine()
			log.Fatalf("Error while selecting partitions: %s\n", err)
		}

		if user.GetUserConfirmation("\nDo you want to save the configuration?") {
			newConfigFilePath, err = configuration.CreateConfigurationFile(config)
			if err != nil {
				log.Printf("Error: %s\n", err)
			}
		}

	}

	log.Printf("Packages: \n\t%v\n\twill be copied to partitions: %v\n", strings.Join(config.Packages, "\n\t"), config.PartitionNumbers)
	if config.InteractiveRun && !user.GetUserConfirmation("Do you want to continue?") {
		log.Printf("Operation cancelled by user\n")
		return
	}

	if !config.NoClone {
		if helper.DoesFileExists(config.Target) {
			askUser := fmt.Sprintf("File %s already exists. Do you want to delete it?", config.Target)
			if config.InteractiveRun && !user.GetUserConfirmation(askUser) {
				user.CleanUpCommandLine()
				log.Fatalf("file already exists and user chose not to delete it")
			}
			if err := os.Remove(config.Target); err != nil {
				user.CleanUpCommandLine()
				log.Fatalf("unable to delete existing file: %s", err)
			}
		}
		err := image.CloneImage(config.Source, config.Target)
		if err != nil {
			log.Fatalf("Error: %s\n", err)
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
	logPath := flags.String("log-path", "./", "Path to log file")
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
	//if *overwrite {
	config.Overwrite = *overwrite
	//}
	if *targetDirectory != "" {
		config.TargetDirectory = *targetDirectory
	}
	config.NoClone = *noClone
	config.PackageDir = *packageDirectory
	config.LogPath = *logPath

	config.InteractiveRun = interactiveRun
	return config, nil
}

func setupLogFile(path string) (*os.File, error) {
	logFile, err := os.OpenFile(filepath.Join(path, "package_to_image_placer.log"), os.O_RDWR|os.O_CREATE|os.O_APPEND, 0666)
	if err != nil {
		return nil, err
	}
	mw := io.MultiWriter(os.Stdout, logFile)
	log.SetOutput(mw)
	return logFile, nil
}

func closeLogFile(logFile *os.File) {
	logFile.Close()
}
