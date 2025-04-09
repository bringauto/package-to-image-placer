package configuration

import (
	"os"
	"package-to-image-placer/pkg/helper"
	"testing"
)

var package1 = PackageConfig{
	PackagePath:       "../../testdata/archives/example_without_service.zip",
	EnableServices:    true,
	ServiceNameSuffix: "example",
	TargetDirectory:   "target/dir",
	OverwriteFiles:    []string{"file1.txt", "file2.txt"},
}

var package2 = PackageConfig{
	PackagePath:       "../../testdata/archives/example_with_service.zip",
	EnableServices:    false,
	ServiceNameSuffix: "tooBig",
	TargetDirectory:   "target/dir",
	OverwriteFiles:    []string{"file3.txt", "file4.txt"},
}

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
	Config = Configuration{
		Source:           sourceImg,
		Target:           "target.img",
		NoClone:          false,
		Packages:         []PackageConfig{package1, package2},
		PartitionNumbers: []int{1, 2},
		InteractiveRun:   false,
		PackageDir:       "package/dir",
		LogPath:          "./",
	}

	err := ValidateConfiguration()
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
}

func TestValidateConfiguration_PackageNotExist(t *testing.T) {
	var nonexistentPackage = PackageConfig{
		PackagePath:       "nonexistent.zip",
		EnableServices:    true,
		ServiceNameSuffix: "example",
		TargetDirectory:   "target/dir",
		OverwriteFiles:    []string{"file1.txt", "file2.txt"},
	}

	Config = Configuration{
		Source:           sourceImg,
		Target:           "target.img",
		NoClone:          false,
		Packages:         []PackageConfig{nonexistentPackage},
		PartitionNumbers: []int{1, 2},
		InteractiveRun:   false,
		PackageDir:       "package/dir",
		LogPath:          "./",
	}

	err := ValidateConfiguration()
	if err == nil {
		t.Fatalf("expected error, got nil")
	}
}

func TestValidateConfiguration_MissingTarget(t *testing.T) {
	Config = Configuration{
		Source:           sourceImg,
		NoClone:          false,
		Packages:         []PackageConfig{package1, package2},
		PartitionNumbers: []int{1, 2},
		InteractiveRun:   false,
		PackageDir:       "package/dir",
		LogPath:          "./",
	}

	err := ValidateConfiguration()
	if err == nil {
		t.Fatalf("expected error, got nil")
	}
}

func TestValidateConfiguration_SameSourceAndTarget(t *testing.T) {
	Config = Configuration{
		Source:           sourceImg,
		Target:           sourceImg,
		NoClone:          false,
		Packages:         []PackageConfig{package1, package2},
		PartitionNumbers: []int{1, 2},
		InteractiveRun:   false,
		PackageDir:       "package/dir",
		LogPath:          "./",
	}

	err := ValidateConfiguration()
	if err == nil {
		t.Fatalf("expected error, got nil")
	}
}

func TestValidateConfiguration_NoSourceAndNoClone(t *testing.T) {
	Config = Configuration{
		Target:           "target.img",
		NoClone:          false,
		Packages:         []PackageConfig{package1, package2},
		PartitionNumbers: []int{1, 2},
		InteractiveRun:   false,
		PackageDir:       "package/dir",
		LogPath:          "./",
	}

	err := ValidateConfiguration()
	if err == nil {
		t.Fatalf("expected error, got nil")
	}
}

func TestValidateConfiguration_NoCloneTargetDoesNotExist(t *testing.T) {
	Config = Configuration{
		Target:           "nonexistent.img",
		NoClone:          true,
		Packages:         []PackageConfig{package1, package2},
		PartitionNumbers: []int{1, 2},
		InteractiveRun:   false,
		PackageDir:       "package/dir",
		LogPath:          "./",
	}

	err := ValidateConfiguration()
	if err == nil {
		t.Fatalf("expected error, got nil")
	}
}

func TestValidateConfiguration_InvalidLogPath(t *testing.T) {
	Config = Configuration{
		Source:           sourceImg,
		Target:           "target.img",
		NoClone:          false,
		Packages:         []PackageConfig{package1, package2},
		PartitionNumbers: []int{1, 2},
		InteractiveRun:   false,
		PackageDir:       "package/dir",
		LogPath:          "/invalid/log/path",
	}

	err := ValidateConfiguration()
	if err == nil {
		t.Fatalf("expected error, got nil")
	}
}
