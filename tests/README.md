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

1. **Disk Space:** Ensure that your system has **at least 100GB of free disk space** available before running the tests. Insufficient space may cause tests to fail or behave unexpectedly.  
2. **Root Privileges:** The application runs with root privileges during testing. Therefore, you must provide the **root password** when prompted.
3. **Close VSCode and other IDEs:** Ensure that VSCode or any other IDEs are closed before running the tests. These applications can consume significant amounts of RAM when loading testing files, potentially causing the system to freeze or crash.

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

## 1. Create disk on target device tests

### 1.1. Test suite: test_app_valid_scenarios

This test suit tests if the application runs correctly with valid arguments.

- test that the packages fits on disk
- symlink to the package
