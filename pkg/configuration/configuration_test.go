package configuration

import (
	"testing"
)

func TestValidateConfiguration_Success(t *testing.T) {
	config := Configuration{
		Source:           "../../testdata/testImage.img",
		Target:           "target.img",
		NoClone:          false,
		Packages:         []string{"package1", "package2"},
		PartitionNumbers: []int{1, 2},
		TargetDirectory:  "target/dir",
		ServiceNames:     []string{"service1", "service2"},
		Overwrite:        true,
		InteractiveRun:   false,
		PackageDir:       "package/dir",
	}

	err := ValidateConfiguration(config)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
}

func TestValidateConfiguration_MissingTarget(t *testing.T) {
	config := Configuration{
		Source:           "source.img",
		NoClone:          false,
		Packages:         []string{"package1", "package2"},
		PartitionNumbers: []int{1, 2},
		TargetDirectory:  "target/dir",
		ServiceNames:     []string{"service1", "service2"},
		Overwrite:        true,
		InteractiveRun:   false,
		PackageDir:       "package/dir",
	}

	err := ValidateConfiguration(config)
	if err == nil {
		t.Fatalf("expected error, got nil")
	}
}

func TestValidateConfiguration_SameSourceAndTarget(t *testing.T) {
	config := Configuration{
		Source:           "image.img",
		Target:           "image.img",
		NoClone:          false,
		Packages:         []string{"package1", "package2"},
		PartitionNumbers: []int{1, 2},
		TargetDirectory:  "target/dir",
		ServiceNames:     []string{"service1", "service2"},
		Overwrite:        true,
		InteractiveRun:   false,
		PackageDir:       "package/dir",
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
		Packages:         []string{"package1", "package2"},
		PartitionNumbers: []int{1, 2},
		TargetDirectory:  "target/dir",
		ServiceNames:     []string{"service1", "service2"},
		Overwrite:        true,
		InteractiveRun:   false,
		PackageDir:       "package/dir",
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
		Packages:         []string{"package1", "package2"},
		PartitionNumbers: []int{1, 2},
		TargetDirectory:  "target/dir",
		ServiceNames:     []string{"service1", "service2"},
		Overwrite:        true,
		InteractiveRun:   false,
		PackageDir:       "package/dir",
	}

	err := ValidateConfiguration(config)
	if err == nil {
		t.Fatalf("expected error, got nil")
	}
}
