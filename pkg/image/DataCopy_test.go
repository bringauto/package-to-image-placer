package image

import (
	"package-to-image-placer/pkg/configuration"
	"testing"
)

const exampleArchive = "../../testdata/archives/example.zip"
const partitionNumber = 1

func createDefaultConfig() configuration.Configuration {
	return configuration.Configuration{
		Source:           "../../testdata/testImage.img",
		Target:           "../../testdata/testImage.img",
		NoClone:          false,
		Packages:         []string{},
		PartitionNumbers: []int{1},
		TargetDirectory:  "target/dir",
		ServiceNames:     []string{},
		Overwrite:        true,
		InteractiveRun:   false,
	}
}

// If this test doesn't pass, all other (failing tests) are without any significance
func TestMountPartitionAndCopyPackage_Success(t *testing.T) {
	config := createDefaultConfig()

	err := MountPartitionAndCopyPackage(partitionNumber, exampleArchive, &config)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
}

func TestMountPartitionAndCopyPackage_ArchiveSizeTooBig(t *testing.T) {
	packagePath := "../../testdata/archives/tooBig.zip"
	config := createDefaultConfig()

	err := MountPartitionAndCopyPackage(partitionNumber, packagePath, &config)
	if err == nil {
		t.Fatalf("expected error, got nil")
	}
}

func TestMountPartitionAndCopyPackage_InvalidPartition(t *testing.T) {
	config := createDefaultConfig()

	err := MountPartitionAndCopyPackage(-1, exampleArchive, &config)
	if err == nil {
		t.Fatalf("expected error, got nil")
	}
}

func TestMountPartitionAndCopyPackage_InvalidPackagePath(t *testing.T) {
	packagePath := "doesNotExist.zip"
	config := createDefaultConfig()

	err := MountPartitionAndCopyPackage(partitionNumber, packagePath, &config)
	if err == nil {
		t.Fatalf("expected error, got nil")
	}
}

func TestMountPartitionAndCopyPackage_NotAllServicesActivated(t *testing.T) {
	config := createDefaultConfig()
	config.ServiceNames = []string{"unavailable.service"}

	err := MountPartitionAndCopyPackage(partitionNumber, exampleArchive, &config)
	if err == nil {
		t.Fatalf("expected error, got nil")
	}
}

func TestMountPartitionAndCopyPackage_FailExistNoOverwrite(t *testing.T) {
	config := createDefaultConfig()
	config.Overwrite = false

	err := MountPartitionAndCopyPackage(partitionNumber, exampleArchive, &config)
	if err == nil {
		t.Fatalf("expected error, got nil")
	}
}

func TestMountPartitionAndCopyPackage_InvalidTargetDirectory(t *testing.T) {
	config := createDefaultConfig()
	config.TargetDirectory = "../../"

	err := MountPartitionAndCopyPackage(partitionNumber, exampleArchive, &config)
	if err == nil {
		t.Fatalf("expected error, got nil")
	} else if err.Error() != "target directory is not within the mounted partition" {
		t.Fatalf("expected error message 'target directory is not within the mounted partition', got %v", err)
	}

}
