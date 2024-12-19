package main

import (
	"flag"
	"fmt"
	"os"
	helper "package-to-image-placer/pkg"
)

func main() {
	err := helper.AllDepsInstalled()
	if err != nil {
		fmt.Printf("Error: %s\n", err)
		os.Exit(1)
	}
	sourceImage := flag.String("source", "", "Source image")
	targetImage := flag.String("target", "", "Target image")
	flag.Parse()
	// TODO check if all dependencies are installed
	if *targetImage == "" || *sourceImage == "" {
		fmt.Printf("Missing arguments, start with -h to see arguments.\n")
		return
	}

	if helper.DoesFileExists(*targetImage) {
		askUser := fmt.Sprintf("File %s already exists. Do you want to delete it? [Y|y to confirm]\n", *targetImage)
		if !getUserConfirmation(askUser) {
			println("file already exists and user chose not to delete it")
			return
		}
		if err := os.Remove(*targetImage); err != nil {
			fmt.Printf("unable to delete existing file: %s", err)
			return
		}
	}
	//	err := imageMounting.CloneImage(*sourceImage, *targetImage)
	//	// TODO Will only accept zip files, will unzip them.
	//	err = imageMounting.CopyFolderToImage(*targetImage, "test-image-folder", 2, "")
	//	if err != nil {
	//		println(err.Error())
	//		return
	//	}
	//	return
}

func getUserConfirmation(message string) bool {
	var b []byte = make([]byte, 1)
	fmt.Print(message)
	_, err := os.Stdin.Read(b)
	if err != nil {
		return false
	}

	return !(string(b) != "Y" && string(b) != "y")
}
