# Use Case Example

## Requirements for the Input Data

* The `source` or `target` in the case of `-no-clone` must be a valid image file using a GPT partition table and must have at least one partition with an Ext4 filesystem, as the tool can only write to this filesystem. If you want to enable services from the copied package, the destination partition must contain the directories `/etc/systemd/system/` and `/etc/systemd/system/multi-user.target.wants/`, where the service files will be copied.
* The `packages` and `configuration-packages` must be valid zip files containing the files to be copied to the image. Additionally, a package can contain a service file that can be activated in the image. The service file must be included in the package and must have a `.service` extension. If a configuration package contains a service file, it is processed as a normal file and is simply copied to the image, not activated as a service.
* The `partition-numbers` must be valid partition numbers in the image. The partition numbers are 1-based, meaning the first partition is 1, the second is 2, and so on.

## Example Use Case

There is a bash script, [create_test_image.sh](../resources/create_test_image.sh), that creates a test image with a GPT partition table and two partitions with Ext4 filesystems. Additionally, there is [example_package.zip](../resources/example_package.zip) and [example_config.json](../resources/example_config.json). The `example_config.json` file can be used to demonstrate how the package image placer works.

To demonstrate the tool, navigate to the [resources](../resources/) directory. First, create a test image using the bash script:

```bash
./create_test_image.sh
```

Then, run the tool with the example configuration file:

```bash
../package-to-image-placer -config example_config.json
```

The tool will copy the files from the package to the image and activate the service in the image. The output of the tool will be saved in the log file specified in the configuration file. The log file will contain information about the files that were copied to the image and the services that were activated.
