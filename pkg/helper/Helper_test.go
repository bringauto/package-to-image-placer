package helper

import (
	"os"
	"testing"
)

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

}

func cleanup() {
}

func TestRemoveMountDirAndPackageName_1(t *testing.T) {
	path := "/mnt/test/dir1/dir1/package/dirInPackage1/dirInPackage2/file.txt"
	mountDir := "/mnt/test"
	packageDir := "/dir1/dir1/"
	packagePath := "/home/user/package/package.zip"
	expected := "/dirInPackage1/dirInPackage2/file.txt"
	result := RemoveMountDirAndPackageName(path, mountDir, packageDir, packagePath)
	if result != expected {
		t.Errorf("Expected %s, got %s", expected, result)
	}
}

func TestRemoveMountDirAndPackageName_2(t *testing.T) {
	path := "/mnt/test/dir1/dir1/package/dirInPackage1/dirInPackage2/file.txt"
	mountDir := "/mnt/test"
	packageDir := "dir1/dir1"
	packagePath := "/home/user/package/package.zip"
	expected := "/dirInPackage1/dirInPackage2/file.txt"
	result := RemoveMountDirAndPackageName(path, mountDir, packageDir, packagePath)
	if result != expected {
		t.Errorf("Expected %s, got %s", expected, result)
	}
}
