# Use Case Example

There is a bash script, [create_test_image.sh](../resources/create_test_image.sh), that creates a test image with a GPT partition table and two partitions with Ext4 filesystems. Additionally, there is [example_package.zip](../resources/example_package.zip) and [example_config.json](../resources/example_config.json). The `example_config.json` file can be used to demonstrate how the package image placer works.

To demonstrate the tool, navigate to the [resources](../resources/) directory. First, create a test image using the bash script:

```bash
./create_test_image.sh
```

Then, run the tool with the example configuration file (a prerequisite is that the test image is created and package-to-image-placer is built):

```bash
../package-to-image-placer -config example_config.json
```

The tool will copy the files from the *example_package.zip* to the image and activate the service in the image.
