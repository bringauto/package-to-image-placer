package configuration

import (
	"os"
	"package-to-image-placer/pkg/helper"
	"testing"
)

const package1 = "../../testdata/archives/example.zip"
const package2 = "../../testdata/archives/tooBig.zip"
const sourceImg = "../../testdata/testImage.img"

func TestMain(m *testing.M) {
	// Setup code here
	setup()

	// Run tests
	code := m.Run()

	// Cleanup code here
	cleanup()

	// Exit with the code from m.Run()
	os.Exit(code)
}

func setup() {
	err := helper.CopyFile(sourceImg, sourceImg+".in", 0666)
	if err != nil {
		panic(err)
	}
}

func cleanup() {
	os.Remove(sourceImg)
}

func TestValidateConfiguration_Success(t *testing.T) {
	config := Configuration{
		Source:           sourceImg,
		Target:           "target.img",
		NoClone:          false,
		Packages:         []string{package1, package2},
		PartitionNumbers: []int{1, 2},
		TargetDirectory:  "target/dir",
		ServiceNames:     []string{"service1", "service2"},
		Overwrite:        true,
		InteractiveRun:   false,
		PackageDir:       "package/dir",
		LogPath:          ".",
	}

	err := ValidateConfiguration(config)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
}

func TestValidateConfiguration_PackageNotExist(t *testing.T) {
	config := Configuration{
		Source:           sourceImg,
		Target:           "target.img",
		NoClone:          false,
		Packages:         []string{"nonexistent.zip"},
		PartitionNumbers: []int{1, 2},
		TargetDirectory:  "target/dir",
		ServiceNames:     []string{"service1", "service2"},
		Overwrite:        true,
		InteractiveRun:   false,
		PackageDir:       "package/dir",
		LogPath:          ".",
	}

	err := ValidateConfiguration(config)
	if err == nil {
		t.Fatalf("expected error, got nil")
	}
}

func TestValidateConfiguration_MissingTarget(t *testing.T) {
	config := Configuration{
		Source:           sourceImg,
		NoClone:          false,
		Packages:         []string{package1, package2},
		PartitionNumbers: []int{1, 2},
		TargetDirectory:  "target/dir",
		ServiceNames:     []string{"service1", "service2"},
		Overwrite:        true,
		InteractiveRun:   false,
		PackageDir:       "package/dir",
		LogPath:          ".",
	}

	err := ValidateConfiguration(config)
	if err == nil {
		t.Fatalf("expected error, got nil")
	}
}

func TestValidateConfiguration_SameSourceAndTarget(t *testing.T) {
	config := Configuration{
		Source:           sourceImg,
		Target:           sourceImg,
		NoClone:          false,
		Packages:         []string{package1, package2},
		PartitionNumbers: []int{1, 2},
		TargetDirectory:  "target/dir",
		ServiceNames:     []string{"service1", "service2"},
		Overwrite:        true,
		InteractiveRun:   false,
		PackageDir:       "package/dir",
		LogPath:          ".",
	}

	err := ValidateConfiguration(config)
	if err == nil {
		t.Fatalf("expected error, got nil")
	}
}

func TestValidateConfiguration_NoSourceAndNoClone(t *testing.T) {
	config := Configuration{
		Target:           "target.img",
		NoClone:          false,
		Packages:         []string{package1, package2},
		PartitionNumbers: []int{1, 2},
		TargetDirectory:  "target/dir",
		ServiceNames:     []string{"service1", "service2"},
		Overwrite:        true,
		InteractiveRun:   false,
		PackageDir:       "package/dir",
		LogPath:          ".",
	}

	err := ValidateConfiguration(config)
	if err == nil {
		t.Fatalf("expected error, got nil")
	}
}

func TestValidateConfiguration_NoCloneTargetDoesNotExist(t *testing.T) {
	config := Configuration{
		Target:           "nonexistent.img",
		NoClone:          true,
		Packages:         []string{package1, package2},
		PartitionNumbers: []int{1, 2},
		TargetDirectory:  "target/dir",
		ServiceNames:     []string{"service1", "service2"},
		Overwrite:        true,
		InteractiveRun:   false,
		PackageDir:       "package/dir",
		LogPath:          ".",
	}

	err := ValidateConfiguration(config)
	if err == nil {
		t.Fatalf("expected error, got nil")
	}
}
