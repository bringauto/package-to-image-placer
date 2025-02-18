import pytest
import subprocess
import os


@pytest.fixture(scope="session")
def package_to_image_placer_binary():
    """Compile the Go application binary."""
    package_to_image_placer_binary = "../package-to-image-placer"
    source_file = "../ImageToPackagePlacer.go"

    subprocess.run(["go", "build", "-o", package_to_image_placer_binary, source_file], check=True)

    yield package_to_image_placer_binary  # Pass the binary path to tests


@pytest.fixture(autouse=True)
def clean_up_between_tests():
    """Set up the environment before each test is run and clean up after each test is run."""
    test_data_dir = "test_data"
    os.makedirs(test_data_dir, exist_ok=True)

    yield

    subprocess.run(["sudo", "losetup", "-D"], check=True)
    # Clean up
    subprocess.run(["sudo", "rm", "-rf", test_data_dir], check=True)


@pytest.fixture(scope="session", autouse=True)
def setup_environment():
    """Set up the environment before any tests are run."""
    print("Checking if all system utilities are installed...")
    try:
        pass
        # subprocess.run(["go", "version"], check=True, capture_output=True, text=True)
        # subprocess.run(["dd", "--version"], check=True, capture_output=True, text=True)
        # subprocess.run(["mkfs", "--version"], check=True, capture_output=True, text=True)
        # subprocess.run(["mkfs.vfat", "--help"], check=True, capture_output=True, text=True)
        # subprocess.run(["fdisk", "--version"], check=True, capture_output=True, text=True)
        # subprocess.run(["losetup", "--version"], check=True, capture_output=True, text=True)
        # subprocess.run(["mount", "--version"], check=True, capture_output=True, text=True)
        # subprocess.run(["umount", "--version"], check=True, capture_output=True, text=True)
        # subprocess.run(["sync", "--version"], check=True, capture_output=True, text=True)
        # subprocess.run(["fallocate", "--version"], check=True, capture_output=True, text=True)
        # subprocess.run(["mkdir", "--version"], check=True, capture_output=True, text=True)
        # subprocess.run(["rm", "--version"], check=True, capture_output=True, text=True)
        # subprocess.run(["cat", "--version"], check=True, capture_output=True, text=True)
        # subprocess.run(["blockdev", "--version"], check=True, capture_output=True, text=True)
        # subprocess.run(["rmdir", "--version"], check=True, capture_output=True, text=True)
    except FileNotFoundError:
        pytest.exit("Please make sure all required system utilities are installed.")

    yield
