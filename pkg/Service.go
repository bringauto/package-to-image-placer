package package_to_image_placer

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

func AddService(serviceFiles []string, mountDir string) error {
	fmt.Println("Found service files:", strings.Join(serviceFiles, ", "))
	for _, serviceFile := range serviceFiles {
		// TODO ask for each service
		// TODO check all fields
		// TODO edit paths
		destPath := filepath.Join(mountDir, "etc/systemd/system", filepath.Base(serviceFile))
		err := copyFile(destPath, serviceFile, 0644)
		if err != nil {
			return fmt.Errorf("failed to copy service file: %v", err)
		}

		// TODO find if multi-user.target.wants is the correct target
		symlinkPath := filepath.Join(mountDir, "/etc/systemd/system/multi-user.target.wants", filepath.Base(serviceFile))
		err = os.Symlink(filepath.Join("..", filepath.Base(serviceFile)), symlinkPath)
		if err != nil {
			return fmt.Errorf("failed to create symlink: %v", err)
		}

		fmt.Println("Added service file:", destPath)
	}

	return nil
}

func FindServiceFiles(folderPath string) ([]string, error) {
	var serviceFiles []string

	err := filepath.Walk(folderPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() && strings.HasSuffix(info.Name(), ".service") {
			serviceFiles = append(serviceFiles, path)
		}
		return nil
	})

	if err != nil {
		return nil, fmt.Errorf("error walking the path %s: %v", folderPath, err)
	}
	return serviceFiles, nil
}
