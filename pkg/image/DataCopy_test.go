package image

import (
	"os"
	"package-to-image-placer/pkg/configuration"
	"package-to-image-placer/pkg/helper"
	"testing"
)

const exampleArchive = "../../testdata/archives/example.zip"
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

func createDefaultConfig() configuration.Configuration {
	return configuration.Configuration{
		Target:           testImage,
		NoClone:          true,
		Packages:         []string{exampleArchive},
		PartitionNumbers: []int{1},
		TargetDirectory:  "target/dir",
		ServiceNames:     []string{},
		Overwrite:        false,
		InteractiveRun:   false,
	}
}

// If this test doesn't pass, all other (failing tests) are without any significance
func TestMountPartitionAndCopyPackage_Success(t *testing.T) {
	cleanup()
	setup()
	config := createDefaultConfig()

	err := MountPartitionAndCopyPackages(partitionNumber, &config)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
}

func TestMountPartitionAndCopyPackage_ArchiveSizeTooBig(t *testing.T) {
	packagePath := "../../testdata/archives/tooBig.zip"
	config := createDefaultConfig()
	config.Packages = []string{packagePath}

	err := MountPartitionAndCopyPackages(partitionNumber, &config)
	if err == nil {
		t.Fatalf("expected error, got nil")
	}
}

func TestMountPartitionAndCopyPackage_InvalidPartition(t *testing.T) {
	config := createDefaultConfig()

	err := MountPartitionAndCopyPackages(-1, &config)
	if err == nil {
		t.Fatalf("expected error, got nil")
	}
}

func TestMountPartitionAndCopyPackage_InvalidPackagePath(t *testing.T) {
	packagePath := "doesNotExist.zip"
	config := createDefaultConfig()
	config.Packages = []string{packagePath}

	err := MountPartitionAndCopyPackages(partitionNumber, &config)
	if err == nil {
		t.Fatalf("expected error, got nil")
	}
}

func TestMountPartitionAndCopyPackage_NotAllServicesActivated(t *testing.T) {
	cleanup()
	setup()
	config := createDefaultConfig()
	config.ServiceNames = []string{"unavailable.service"}

	err := MountPartitionAndCopyPackages(partitionNumber, &config)
	if err == nil {
		t.Fatalf("expected error, got nil")
	}
}

func TestMountPartitionAndCopyPackage_FailExistNoOverwrite(t *testing.T) {
	cleanup()
	setup()
	config := createDefaultConfig()
	config.Overwrite = false
	err := MountPartitionAndCopyPackages(partitionNumber, &config)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	err = MountPartitionAndCopyPackages(partitionNumber, &config)
	if err == nil {
		t.Fatalf("expected error, got nil")
	}
}

func TestMountPartitionAndCopyPackage_SuccessOverwrite(t *testing.T) {
	cleanup()
	setup()
	config := createDefaultConfig()
	config.Overwrite = true
	err := MountPartitionAndCopyPackages(partitionNumber, &config)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	err = MountPartitionAndCopyPackages(partitionNumber, &config)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
}

func TestMountPartitionAndCopyPackage_TargetDirectoryOutOfMount(t *testing.T) {
	config := createDefaultConfig()
	config.TargetDirectory = "../../"

	err := MountPartitionAndCopyPackages(partitionNumber, &config)
	if err == nil {
		t.Fatalf("expected error, got nil")
	} else if err.Error() != "target directory is not within the mounted partition" {
		t.Fatalf("expected error message 'target directory is not within the mounted partition', got %v", err)
	}
}
