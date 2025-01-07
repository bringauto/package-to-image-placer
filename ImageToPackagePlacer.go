package main

import (
	"flag"
	"fmt"
	"github.com/diskfs/go-diskfs"
	"github.com/sirupsen/logrus"
	"os"
	packagePlacer "package-to-image-placer/pkg"
	"package-to-image-placer/pkg/interaction"
)

func main() {
	sourceImage := flag.String("source", "", "Source image")
	targetImage := flag.String("target", "", "Target image path")
	overwrite := flag.Bool("overwrite", false, "Overwrite files in target image")
	packageDirectory := flag.String("package-dir", "./", "Default package directory, from which package finder starts")
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
	selectedFiles, err := interaction.SelectFilesInDir(*packageDirectory)
	if err != nil {
		fmt.Printf("Error: %s\n", err)
		return
	}

	if len(selectedFiles) == 0 {
		fmt.Printf("No files selected\n")
		return
	}
	fmt.Printf("Selected files: %s\n", selectedFiles)

	//if packagePlacer.DoesFileExists(*targetImage) {
	//	askUser := fmt.Sprintf("File %s already exists. Do you want to delete it?", *targetImage)
	//	if !interaction.GetUserConfirmation(askUser) {
	//		println("file already exists and user chose not to delete it")
	//		return
	//	}
	//	if err := os.Remove(*targetImage); err != nil {
	//		fmt.Printf("unable to delete existing file: %s", err)
	//		return
	//	}
	//}
	// TODO way to skip cloning
	//imageData, err := packagePlacer.CloneImage(*sourceImage, *targetImage)
	//if err != nil {
	//	fmt.Printf("Error: %s\n", err)
	//	return
	//}
	d, _ := diskfs.Open(*targetImage)
	targetPartitions, err := interaction.SelectPartitions(d)
	// TODO Will only accept zip files, will unzip them.
	for _, partition := range targetPartitions {
		logrus.Printf("Copying to partition: %d\n", partition)
		for _, archive := range selectedFiles {
			err := packagePlacer.UnzipPackageToImage(*targetImage, archive, partition, "", *overwrite)
			if err != nil {
				fmt.Printf("Error: %s\n", err)
			}
		}
	}
	return
}
