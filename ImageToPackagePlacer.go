package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	packagePlacer "package-to-image-placer/pkg"
	"package-to-image-placer/pkg/interaction"
)

func main() {
	sourceImage := flag.String("source", "", "Source image")
	targetImage := flag.String("target", "", "Target image path")
	overwrite := flag.Bool("overwrite", false, "Overwrite files in target image")
	packageDirectory := flag.String("package-dir", "./", "Default package directory, from which package finder starts")
	noClone := flag.Bool("no-clone", false, "Do not clone source image. Target image must exist. If operation is not successful, may cause damage the image")
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

	if *targetImage == "" || (*sourceImage == "" && !*noClone) {
		fmt.Printf("Missing arguments, start with -h to see arguments.\n")
		return
	}
	if *noClone && !packagePlacer.DoesFileExists(*targetImage) {
		fmt.Printf("Target image does not exist\n")
		return
	}

	interaction.SetUpCommandline()
	selectedFiles, err := interaction.SelectFilesInDir(*packageDirectory)
	if err != nil {
		log.Printf("Error: %s\n", err)
		return
	}

	if len(selectedFiles) == 0 {
		log.Printf("No files selected\n")
		return
	}
	log.Printf("Selected files: %s\n", selectedFiles)

	if !*noClone {
		err := packagePlacer.CloneImage(*sourceImage, *targetImage)
		if err != nil {
			log.Printf("Error: %s\n", err)
			return
		}
	}
	targetPartitions, err := interaction.SelectPartitions(*targetImage)
	if err != nil {
		log.Printf("Error while selecting partitions: %s\n", err)
	}
	for _, partition := range targetPartitions {
		log.Printf("Copying to partition: %d\n", partition)
		for _, archive := range selectedFiles {
			err := packagePlacer.UnzipPackageToImage(*targetImage, archive, partition, "", *overwrite)
			if err != nil {
				log.Printf("Error: %s\n", err)
			}
		}
	}
	return
}
