package service

import (
	"os"
	"package-to-image-placer/pkg/configuration"
	"package-to-image-placer/pkg/helper"
	"path/filepath"
	"testing"
)

var packageConfig = configuration.PackageConfig{
	PackagePath:       "../../testdata/archives/example_without_service.zip",
	EnableServices:    true,
	ServiceNameSuffix: "",
	TargetDirectory:   "target/dir",
	OverwriteFiles:    nil,
}

func TestAddService_Success(t *testing.T) {

	serviceFile, _ := filepath.Abs("../../testdata/service-mount/package/valid.service")
	mountDir, _ := filepath.Abs("../../testdata/service-mount")
	packageDir, _ := filepath.Abs("../../testdata/service-mount/package")

	// Renew service file. That will be changed during test
	err := helper.CopyFile(serviceFile, serviceFile+".in", os.FileMode(0666))
	if err != nil {
		t.Fatal(err.Error())
	}

	err = AddService(serviceFile, mountDir, packageDir, &packageConfig)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if enabled, err := isServiceEnabled(mountDir, filepath.Base("valid.service")); !enabled || err != nil {
		t.Fatalf("expected service to be enabled, got disabled")
	}

	os.Remove(filepath.Join(mountDir, "etc/systemd/system/multi-user.target.wants/valid.service"))
	os.Remove(filepath.Join(mountDir, "etc/systemd/system/valid.service"))
}

func TestAddService_MissingRequiredFields(t *testing.T) {
	serviceFile := "../../testdata/service-mount/package/missing-fields.service"
	mountDir := "../../testdata/service-mount"
	packageDir := "../../testdata/service-mount/package"

	err := AddService(serviceFile, mountDir, packageDir, &packageConfig)
	if err == nil {
		t.Fatalf("expected error, got nil")
	}
	println(err.Error())
}

func TestCheckRequiredServicesEnabled_True(t *testing.T) {
	mountDir := "../../testdata/service-mount"
	serviceNames := []string{"requires.service"}
	err := CheckRequiredServicesEnabled(mountDir, serviceNames)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
}

func TestCheckRequiredServicesEnabled_False(t *testing.T) {
	mountDir := "../testdata/service-mount"
	serviceNames := []string{"requires-invalid.service"}
	err := CheckRequiredServicesEnabled(mountDir, serviceNames)
	if err == nil {
		t.Fatalf("expected error, got nil")
	}
	println(err.Error())
}
