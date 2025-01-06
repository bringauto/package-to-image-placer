package main

import (
	"flag"
	"fmt"
	"os"
	packagePlacer "package-to-image-placer/pkg"
	"package-to-image-placer/pkg/interaction"
)

func main() {
	s, _ := interaction.SelectTargetDirectory("/home/melkus/Bringauto")
	println(s)
	sourceImage := flag.String("source", "", "Source image")
	targetImage := flag.String("target", "", "Target image path")
	showUsage := flag.Bool("h", false, "Show usage")
	flag.Parse()

	if *showUsage {
		println("Usage:\n\tpackage-to-image-placer -source <source image> -target <target image>\n")
		flag.PrintDefaults()
		return
	}

	err := packagePlacer.AllDepsInstalled()
	if err != nil {
		fmt.Printf("Error: %s\n", err)
		os.Exit(1)
	}

	if *targetImage == "" || *sourceImage == "" {
		fmt.Printf("Missing arguments, start with -h to see arguments.\n")
		return
	}

	interaction.SetUpCommandline()
	selectedFiles, err := interaction.SelectFilesInDir(".")
	if err != nil {
		fmt.Printf("Error: %s\n", err)
		return
	}

	if len(selectedFiles) == 0 {
		fmt.Printf("No files selected\n")
		return
	}
	fmt.Printf("Selected files: %s\n", selectedFiles)

	if packagePlacer.DoesFileExists(*targetImage) {
		askUser := fmt.Sprintf("File %s already exists. Do you want to delete it?", *targetImage)
		if !interaction.GetUserConfirmation(askUser) {
			println("file already exists and user chose not to delete it")
			return
		}
		if err := os.Remove(*targetImage); err != nil {
			fmt.Printf("unable to delete existing file: %s", err)
			return
		}
	}
	imageData, err := packagePlacer.CloneImage(*sourceImage, *targetImage)
	if err != nil {
		fmt.Printf("Error: %s\n", err)
		return
	}

	targetPartitions, err := interaction.SelectPartitions(imageData.TargetDisk)
	// TODO Will only accept zip files, will unzip them.
	for _, partition := range targetPartitions {
		err = packagePlacer.UnzipPackageToImage(*targetImage, "test-image-folder", partition, "")
		if err != nil {
			println(err.Error())
			return
		}
	}
	return
}
