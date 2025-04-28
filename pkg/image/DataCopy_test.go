package image

import (
	"os"
	"package-to-image-placer/pkg/configuration"
	"package-to-image-placer/pkg/helper"
	"strings"
	"testing"
)

var package1 = configuration.PackageConfig{
	PackagePath:       "../../testdata/archives/example_without_service.zip",
	EnableServices:    false,
	ServiceNameSuffix: "",
	TargetDirectory:   "target/dir",
	OverwriteFiles:    nil,
}

const partitionNumber = 1
const testImage = "../../testdata/testImage.img"

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
	err := helper.CopyFile(testImage, testImage+".in", 0666)
	if err != nil {
		panic(err)
	}
}

func cleanup() {
	os.Remove(testImage)
}

func createDefaultConfig() {
	configuration.Config = configuration.Configuration{
		Target:           testImage,
		NoClone:          true,
		Packages:         []configuration.PackageConfig{package1},
		PartitionNumbers: []int{1},
		InteractiveRun:   false,
	}
}

// If this test doesn't pass, all other (failing tests) are without any significance
func TestMountPartitionAndCopyPackage_Success(t *testing.T) {
	cleanup()
	setup()
	createDefaultConfig()

	err := MountPartitionAndCopyPackages(partitionNumber, true)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
}

func TestMountPartitionAndCopyPackage_ArchiveSizeTooBig(t *testing.T) {
	packagePath := "../../testdata/archives/tooBig.zip"
	createDefaultConfig()
	configuration.Config.Packages[0].PackagePath = packagePath

	err := MountPartitionAndCopyPackages(partitionNumber, true)
	if err == nil {
		t.Fatalf("expected error, got nil")
	}
}

func TestMountPartitionAndCopyPackage_InvalidPartition(t *testing.T) {
	createDefaultConfig()

	err := MountPartitionAndCopyPackages(-1, true)
	if err == nil {
		t.Fatalf("expected error, got nil")
	}
}

func TestMountPartitionAndCopyPackage_InvalidPackagePath(t *testing.T) {
	createDefaultConfig()
	configuration.Config.Packages[0].PackagePath = "doesNotExist.zip"

	err := MountPartitionAndCopyPackages(partitionNumber, true)
	if err == nil {
		t.Fatalf("expected error, got nil")
	}
}

func TestMountPartitionAndCopyPackage_NotAllServicesActivated(t *testing.T) {
	cleanup()
	setup()
	createDefaultConfig()
	configuration.Config.Packages[0].EnableServices = true

	err := MountPartitionAndCopyPackages(partitionNumber, true)
	if err == nil {
		t.Fatalf("expected error, got nil")
	}
}

func TestMountPartitionAndCopyPackage_FailExistNoOverwrite(t *testing.T) {
	cleanup()
	setup()
	createDefaultConfig()
	configuration.Config.Packages[0].OverwriteFiles = nil
	err := MountPartitionAndCopyPackages(partitionNumber, true)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	err = MountPartitionAndCopyPackages(partitionNumber, true)
	if err == nil {
		t.Fatalf("expected error, got nil")
	}
}

func TestMountPartitionAndCopyPackage_SuccessOverwrite(t *testing.T) {
	cleanup()
	setup()
	createDefaultConfig()
	err := MountPartitionAndCopyPackages(partitionNumber, true)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	configuration.Config.Packages[0].OverwriteFiles = []string{"/example/a/b/c/file"}

	err = MountPartitionAndCopyPackages(partitionNumber, true)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
}

func TestMountPartitionAndCopyPackage_NonExistingOverwrite(t *testing.T) {
	cleanup()
	setup()
	createDefaultConfig()
	err := MountPartitionAndCopyPackages(partitionNumber, true)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	configuration.Config.Packages[0].OverwriteFiles = []string{"/example/a/b/c/file1"}

	err = MountPartitionAndCopyPackages(partitionNumber, true)
	if err == nil {
		t.Fatalf("expected error, got nil")
	}
}

func TestMountPartitionAndCopyPackage_TargetDirectoryOutOfMount(t *testing.T) {
	createDefaultConfig()
	configuration.Config.Packages[0].TargetDirectory = "../../"
	err := MountPartitionAndCopyPackages(partitionNumber, true)
	if err == nil {
		t.Fatalf("expected error, got nil")
	} else if !strings.Contains(err.Error(), "target directory is not within the mounted partition") {
		t.Fatalf("expected error message 'target directory is not within the mounted partition', got %v", err)
	}
}
