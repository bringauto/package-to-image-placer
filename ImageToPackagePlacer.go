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
	err := parseArguments(os.Args[1:])
	if err != nil {
		log.Fatalf("Error parsing arguments: %v", err)
	}
	err = configuration.ValidateConfiguration()
	if err != nil {
		log.Fatalf("Configuration validation error: %v", err)
	}
	logFile, err := setupLogFile(configuration.Config.LogPath)
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
	var packages []string
	if configuration.Config.InteractiveRun {
		header_base := "Choose standard package to copy."
		packages, err = user.SelectFilesInDir(configuration.Config.PackageDir, header_base)
		log.Printf("Selected standard packages: %v\n", packages)
		if err != nil {
			log.Printf("Error: %s\n", err)
			return
		}
		for _, pkg := range packages {
			packageConfig := configuration.PackageConfig{PackagePath: pkg}
			configuration.Config.Packages = append(configuration.Config.Packages, packageConfig)
		}

		header_base = "Choose configuration package to copy."
		packages, err = user.SelectFilesInDir(configuration.Config.PackageDir, header_base)
		log.Printf("Selected configuration packages: %v\n", packages)
		if err != nil {
			log.Printf("Error: %s\n", err)
			return
		}
		for _, pkg := range packages {
			packageConfig := configuration.ConfigurationPackage{PackagePath: pkg}
			configuration.Config.ConfigurationPackages = append(configuration.Config.ConfigurationPackages, packageConfig)
		}

		if len(configuration.Config.Packages) == 0 && len(configuration.Config.ConfigurationPackages) == 0 {
			log.Printf("No packages selected. Exiting...\n")
			return
		}

		imagePath := ""
		if configuration.Config.NoClone {
			imagePath = configuration.Config.Target
		} else {
			imagePath = configuration.Config.Source
		}

		err = helper.ValidSourceImage(imagePath)
		if err != nil {
			helper.RemoveInvalidOutputImage(configuration.Config.Target, configuration.Config.NoClone)
			log.Fatalf("Error: %s\n", err)
		}

		configuration.Config.PartitionNumbers, err = user.SelectPartitions(imagePath)
		if err != nil {
			helper.RemoveInvalidOutputImage(configuration.Config.Target, configuration.Config.NoClone)
			log.Fatalf("Error while selecting partitions: %s\n", err)
		}

		if user.GetUserConfirmation("Do you want to save the configuration?") {
			newConfigFilePath, err = configuration.CreateConfigurationFile(configuration.Config)
			if err != nil {
				log.Printf("Error: %s\n", err)
			}
		}

	}

	// Create a slice of package paths from configuration.Config.Packages and configuration.Config.ConfigurationPackages
	var standardPackagePaths []string
	var configurationPackagePaths []string

	for _, pkg := range configuration.Config.Packages {
		standardPackagePaths = append(standardPackagePaths, pkg.PackagePath)
	}
	for _, pkg := range configuration.Config.ConfigurationPackages {
		configurationPackagePaths = append(configurationPackagePaths, pkg.PackagePath)
	}
	log.Printf("\n%d standard packages: \n\t%v\n%d configuration packages: \n\t%v\nwill be copied to partitions: %v\n", len(standardPackagePaths), strings.Join(standardPackagePaths, "\n\t"), len(configurationPackagePaths), strings.Join(configurationPackagePaths, "\n\t"), configuration.Config.PartitionNumbers)

	if configuration.Config.InteractiveRun && !user.GetUserConfirmation("Do you want to continue?") {
		log.Printf("Operation cancelled by user\n")
		return
	}

	if !configuration.Config.NoClone {
		if helper.DoesFileExists(configuration.Config.Target) {
			askUser := fmt.Sprintf("File %s already exists. Do you want to delete it?", configuration.Config.Target)
			if configuration.Config.InteractiveRun && !user.GetUserConfirmation(askUser) {
				log.Fatalf("file already exists and user chose not to delete it")
			}
			if err := os.Remove(configuration.Config.Target); err != nil {
				log.Fatalf("unable to delete existing file: %s", err)
			}
		}
		err := image.CloneImage(configuration.Config.Source, configuration.Config.Target)
		if err != nil {
			helper.RemoveInvalidOutputImage(configuration.Config.Target, configuration.Config.NoClone)
			log.Fatalf("Error: %s\n", err)
		}
	}

	err = image.CopyPackagesToImagePartitions()
	if err != nil {
		helper.RemoveInvalidOutputImage(configuration.Config.Target, configuration.Config.NoClone)
		log.Fatalf("Error: %s\n", err)
		return
	}

	log.Printf("All packages copied successfully\n")

	if newConfigFilePath != "" {
		err = configuration.UpdateConfigurationFile(configuration.Config, newConfigFilePath)
		if err != nil {
			log.Fatalf("Error: %s\n", err)
		}
	}
}

func parseArguments(args []string) error {
	flags := flag.NewFlagSet("package-to-image-placer", flag.ContinueOnError)
	configFile := flags.String("config", "", "Path to configuration file (non-interactive mode)")
	targetImage := flags.String("target", "", "Target image path (will be created).")
	sourceImage := flags.String("source", "", "Source image")
	noClone := flags.Bool("no-clone", false, "Do not clone source image. Target image must exist. If operation is not successful, may cause damage the image")
	packageDir := flags.String("package-dir", "./", "Default package directory, from which package finder starts (interactive mode)")
	logPath := flags.String("log-path", "./", "Path to log file")
	showUsage := flags.Bool("h", false, "Show usage")

	err := flags.Parse(args)
	if err != nil {
		return err
	}

	if *showUsage {
		fmt.Printf("Usage:\n" +
			"Interactive: \t\tpackage-to-image-placer -target <target_image> [ -source <src_image> | -no-clone ] [ opts... ]\n" +
			"Non-interactive: \tpackage-to-image-placer -configuration.Config <config_file> [ <override-opts> ]\n")
		flags.PrintDefaults()
		os.Exit(0)
	}

	if *configFile != "" {
		file, err := os.Open(*configFile)
		if err != nil {
			return fmt.Errorf("error opening configuration.Config file: %v", err)
		}
		defer file.Close()

		decoder := json.NewDecoder(file)
		err = decoder.Decode(&configuration.Config)
		if err != nil {
			return fmt.Errorf("error decoding configuration.Config file: %v", err)
		}

		configuration.Config.InteractiveRun = false
		configuration.Config.ConfigFile = *configFile
		configuration.ConvertRelativePathsToWorkingDir()
	}

	if *sourceImage != "" {
		configuration.Config.Source = *sourceImage
	}
	if *targetImage != "" {
		configuration.Config.Target = *targetImage
	}
	if *packageDir != "" {
		configuration.Config.PackageDir = *packageDir
	}
	if *logPath != "./" {
		configuration.Config.LogPath = *logPath
	}

	// Check if the overwrite flag has been set
	noCloneSet := false
	flags.Visit(func(f *flag.Flag) {
		if f.Name == "no-clone" {
			noCloneSet = true
		}
	})
	if noCloneSet {
		configuration.Config.NoClone = *noClone
	}
	return nil
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
