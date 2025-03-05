# Tests documentation

## Requirements

The tests share the same requirements as the program itself and additionally depend on standard Linux utilities and Python 3. Written in Python, they leverage pytest for execution. It is recommended to use a virtual environment for managing Python dependencies.

```bash
python3 -m venv venv
source venv/bin/activate
pip install -r requirements.txt
```

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

This test suite tests the application with invalid arguments.
[ ] `Test case: test_invalid_file_paths` - Tests if the package_to_image_placer will fail when the paths are invalid.

[ ] `Test case: test_empty_image_file` - Tests if the package_to_image_placer will fail when the image file is empty.

[ ] `Test case: test_invalid_image_file` - Tests if the package_to_image_placer will fail when the image file is invalid.

[ ] `Test case: test_without_specified_partitions` - Tests if the package_to_image_placer will fail when the partitions are not specified.

[ ] `Test case: test_without_specified_package` - Tests if the package_to_image_placer will fail when the package is not specified.

[ ] `Test case: test_invalid_config_format` - Tests if the package_to_image_placer will fail when the config file is missing a field.

---

### 2. Test suite: test_app_invalid_scenarios

This test suite tests the application with scenarios that should fail.

[ ] `Test case: test_write_one_big_package` - Tests if the package_to_image_placer will fail when the package is bigger than the image.

[ ] `Test case: test_write_two_big_packages` - Tests if the package_to_image_placer will fail when the sum of packages is bigger than the image.

[ ] `Test case: test_write_one_package_to_small_partition` - Tests if the package_to_image_placer will fail when the package is bigger than the partition.

[ ] `Test case: test_write_multiple_packages_to_partition` - Tests if the package_to_image_placer will fail when the sum of packages is bigger than the partition.

---

### 3. Test suite: test_app_valid_scenarios

This test suite tests the application with scenarios that should pass.

[ ] `Test case: test_app_show_help` - Tests if the package_to_image_placer will show help message when -h flag is passed.

[ ] `Test case: test_write_one_package` - Test if the package_to_image_placer will write a package to an image.

[ ] `Test case: test_write_two_packages` - Tests if the package_to_image_placer will write two packages to an image.

[ ] `Test case: test_write_multiple_packages_to_multiple_partitions` - Test if the package_to_image_placer will write multiple packages to multiple partitions.

[ ] `Test case: test_pass_package_as_symlink` - Test if the package_to_image_placer will write a package to an image when the package is a symlink.

[ ] `Test case: test_write_one_package_target_override` - Test if the package_to_image_placer will write a package to an image when the target image already exists and the override flag is set, ensuring it overwrites the target image.

[ ] `Test case: test_write_one_package_no_clone` - Test if the package_to_image_placer will write a package to an image without specifying a source image and with the no-clone flag set.

[ ] `Test case: test_write_one_package_try_different_overwrite_flag` - Test if the package_to_image_placer will write a package to an image when the target image already exists. First, it attempts to write without the overwrite flag, expecting the operation to fail. Then, it retries with the overwrite flag set, expecting the operation to succeed.

[ ] `Test case: test_write_one_package_overwrite_config_overwrite_value_to_false` - Test if the package_to_image_placer will write a package to an image when the target image already exists. The overwrite flag is set to true in the config file but is overshadowed by the overwrite argument set to false, ensuring the operation fails.

[ ] `Test case: test_write_one_package_overwrite_config_no_clone_value_to_true` - Test if the package_to_image_placer will write a package to an image without specifying a source image and with the no-clone flag set to True in the arguments, overshadowing the value set in the config file.

[ ] `Test case: test_write_one_package_overwrite_config_no_clone_value_to_false` - Test if the package_to_image_placer will fail when the output image already exists and the no-clone flag is set to False in the arguments, overshadowing the value set in the config file.

[ ] `Test case: test_double_write_without_override` - Tests if the package_to_image_placer will fail when tha package is already written to the image and the overwrite flag is not set.

[ ] `Test case: test_write_to_custom_target_directory` - Test if the package_to_image_placer will write a package to an image in a custom target directory.

---

### 4. Test suite: test_app_service_scenarios

This test suite tests the application while using packages with services.

[ ] `Test case: test_write_one_package_with_service` - Test if the package_to_image_placer will write a package with a service to an image.

[ ] `Test case: test_write_package_with_services` - Test if the package_to_image_placer will write a package with two services to an image.

[ ] `Test case: test_write_packages_with_services_with_override` - Test if the package_to_image_placer will write two packages with two services to an image when override is set to True.

[ ] `Test case: test_write_packages_with_services_without_override` - Test if the package_to_image_placer will fail when attempting to write two packages with the same service to an image without override enabled.

[ ] `Test case: test_write_multiple_packages_with_services` - Write multiple packages with services to an image.
