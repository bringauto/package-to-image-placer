# Tests documentation

## Requirements

The tests share the same requirements as the program itself and additionally depend on standard Linux utilities and Python 3. Written in Python, they leverage pytest for execution. It is recommended to use a virtual environment for managing Python dependencies.

```bash
python3 -m venv venv
source venv/bin/activate
pip install -r requirements.txt
```

Before running the tests, ensure that the application can be built successfully. Follow the build instructions provided in the main documentation.

---
**⚠️ WARNING**  
To ensure the tests run correctly, please take note of the following requirements:  

1. **Root Privileges:** The application requires root privileges during testing. Ensure you have the **root password** available when prompted.
2. **Non-standard Test Behavior:** If tests are interrupted or exhibit non-standard behavior, you may need to manually unmount any devices used during the tests. In some cases, a system restart might be necessary to restore normal operation.

---

## Running tests

To run tests use the following command:

```bash
pytest
```

To run tests with additional options, you can use the following flags:

- `-s`: Show print statements in the test functions.
- `-v`: Show verbose output.
- `-k <expression>`: Run only tests that match the provided expression.

Example:

```bash
pytest -s -v -k "test_function_name"
```

In cases where tests are interrupted by user or crash, you may need to manually unmount any devices used during the tests.

To list all test cases, use the following command:

```bash
pytest --collect-only
```

## Package to Image Placer tests

### 1. Test suite: test_invalid_arguments

This test suite validates the application's behavior when provided with invalid arguments.

[ ] `Test case: test_01_invalid_file_paths` - Verifies that the application fails when provided with non-existent file paths.

[ ] `Test case: test_02_empty_image_file` - Ensures the application fails when the input image file is empty.

[ ] `Test case: test_03_invalid_image_file` - Confirms the application fails when the input image file is invalid (e.g., not a valid image format).

[ ] `Test case: test_04_without_specified_partitions` - Checks that the application fails when no partitions are specified in the configuration.

[ ] `Test case: test_05_without_specified_package` - Validates that the application fails when no package is specified in the configuration.

[ ] `Test case: test_06_invalid_config_format` - Tests the application's behavior when the configuration file is missing required fields.

[ ] `Test case: test_07_write_package_and_nonexisting_configuration_package` - Ensures the application fails when attempting to write a package and a non-existent configuration package to an image.

[ ] `Test case: test_08_write_nonexisting_package_and_configuration_package` - Confirms the application fails when attempting to write a non-existent package and a configuration package to an image.

---

### 2. Test suite: test_app_invalid_scenarios

This test suite tests the application with scenarios that should fail.

[ ] `Test case: test_01_write_one_big_package` - Tests if the package_to_image_placer will fail when the package is bigger than the image.

[ ] `Test case: test_02_write_two_big_packages` - Tests if the package_to_image_placer will fail when the sum of packages is bigger than the image.

[ ] `Test case: test_03_write_one_package_to_small_partition` - Tests if the package_to_image_placer will fail when the package is bigger than the partition.

[ ] `Test case: test_04_write_multiple_packages_to_partition` - Tests if the package_to_image_placer will fail when the sum of packages is bigger than the partition.

[ ] `Test case: test_05_write_package_with_invalid_overwrite` - Tests if the package_to_image_placer will fail when attempting to write a package with an invalid overwrite file.

[ ] `Test case: test_06_write_package_with_invalid_overwrite_in_config_package` - Tests if the package_to_image_placer will fail when attempting to write a config package with an invalid overwrite file.

---

### 3. Test suite: test_app_valid_scenarios

This test suite tests the application with scenarios that should pass.

[ ] `Test case: test_01_app_shows_help` - Tests if the package_to_image_placer will show help message when -h flag is passed.

[ ] `Test case: test_02_write_one_package` - Test if the package_to_image_placer will write a package to an image.

[ ] `Test case: test_03_write_two_packages` - Tests if the package_to_image_placer will write two packages to an image.

[ ] `Test case: test_04_write_multiple_packages_to_multiple_partitions` - Test if the package_to_image_placer will write multiple packages to multiple partitions.

[ ] `Test case: test_05_pass_package_as_symlink` - Test if the package_to_image_placer will write a package to an image when the package is a symlink.

[ ] `Test case: test_06_write_one_package_target_already_exists` - Test if the package_to_image_placer will write a package to an image when the target image already exists.

[ ] `Test case: test_07_write_one_package_no_clone` - Test if the package_to_image_placer will write a package to an image without specifying a source image and with the no-clone flag set.

[ ] `Test case: test_08_write_one_package_try_different_overwrite_flag` - Test if the package_to_image_placer will write a package to an image when the target image already exists. First, it attempts to write without the overwrite flag, expecting the operation to fail. Then, it retries with the overwrite flag set, expecting the operation to succeed.

[ ] `Test case: test_09_write_one_package_overwrite_config_no_clone_value_to_true` - Test if the package_to_image_placer will write a package to an image without specifying a source image and with the no-clone flag set to True in the arguments, overshadowing the value set in the config file.

[ ] `Test case: test_10_write_one_package_overwrite_config_no_clone_value_to_false` - Test if the package_to_image_placer will fail when the output image already exists and the no-clone flag is set to False in the arguments, overshadowing the value set in the config file.

[ ] `Test case: test_11_double_write_without_override` - Tests if the package_to_image_placer will fail when the package is already written to the image and the overwrite flag is not set.

[ ] `Test case: test_12_write_to_custom_target_directory` - Test if the package_to_image_placer will write a package to an image in a custom target directory.

[ ] `Test case: test_13_write_only_configuration_package` - Test if the package_to_image_placer will write a package to an image when the package is a configuration package.

[ ] `Test case: test_14_write_package_and_configuration_package` - Test if the package_to_image_placer will write a package and configuration package to an image.

[ ] `Test case: test_15_write_multiple_different_packages` - Write five packages of both types to an image.

---

### 4. Test suite: test_app_service_scenarios

This test suite tests the application while using packages with services.

[ ] `Test case: test_01_write_package_with_service` - Verifies that the package_to_image_placer writes a package with a service to an image.

[ ] `Test case: test_02_write_package_with_services` - Ensures the package_to_image_placer fails when attempting to write a package with multiple services to an image.

[ ] `Test case: test_03_write_packages_with_services_with_override` - Confirms that the package_to_image_placer writes two packages with the same service to an image when override is enabled.

[ ] `Test case: test_04_write_packages_with_services_without_override` - Validates that the package_to_image_placer fails when attempting to write two packages with the same service to an image without override enabled.

[ ] `Test case: test_05_write_multiple_packages_with_services_to_single_dir` - Tests if the package_to_image_placer writes multiple packages with services to an image in a single target directory.

[ ] `Test case: test_06_write_multiple_packages_with_services_to_multiple_dirs` - Tests if the package_to_image_placer writes multiple packages with services to an image in multiple target directories.

[ ] `Test case: test_07_package_service_with_suffix` - Verifies that the package_to_image_placer creates a package with a service file that includes a suffix.

[ ] `Test case: test_08_package_with_service_with_suffix_starting_with_hyphen` - Ensures the package_to_image_placer fails when creating a package with a service file that includes a suffix starting with a hyphen.

[ ] `Test case: test_09_try_to_write_service_when_package_do_not_contain_any_service` - Confirms that the package_to_image_placer fails when attempting to write a service for a package that does not contain any service files.
