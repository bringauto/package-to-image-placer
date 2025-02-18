import subprocess
import os
import shutil
import json
from random import random, randint
from time import sleep


def convert_size_to_bytes(size: str) -> int:
    """
    Convert a size string to bytes.

    Args:
        size (str): The size string to convert (e.g., '5MB').

    Returns:
        int: The size in bytes.
    """
    print(size)
    size_units = size[-2:].upper()
    size_value = int(size[:-2])

    if size_units == "KB":
        return size_value * 1024
    elif size_units == "MB":
        return size_value * 1024 * 1024
    elif size_units == "GB":
        return size_value * 1024 * 1024 * 1024
    else:
        raise ValueError("Unsupported size unit. Please use KB, MB, or GB.")


def create_test_package(package_path: str, package_size: str) -> None:
    if os.path.exists(package_path):
        print(f"Package {package_path} already exists. Removing...")
        shutil.rmtree(package_path)

    package_size_bytes = convert_size_to_bytes(package_size)

    os.makedirs(package_path)

    with open(f"{package_path}/test_file", "wb") as f:
        f.write(os.urandom(package_size_bytes))

    subprocess.run(
        ["zip", "-r", f"{os.path.abspath(package_path)}.zip", os.path.basename(package_path)],
        check=True,
        cwd=os.path.dirname(package_path),
    )


import subprocess
import os


import subprocess


def create_image(image_path: str, image_size: str, partitions_count: int) -> str:
    """
    Create a disk image with the specified size and partition table (DOS/MBR).

    Args:
        image_path (str): The path where the disk image will be created.
        image_size (str): The size of the disk image (e.g., '5MB').
        partitions_count (int): The number of partitions to create.

    Returns:
        str: The path to the created disk image.
    """
    if partitions_count > 10:
        raise ValueError("The maximum number of partitions is 10.")

    image_size_bytes = convert_size_to_bytes(image_size)

    # Create a blank disk image with the specified size
    subprocess.run(["fallocate", "-l", str(image_size_bytes), image_path], check=True)

    # Create a partition table (DOS/MBR or GPT)
    subprocess.run(["parted", image_path, "--script", "mklabel", "gpt"], check=True)

    # Calculate partition sizes and create partitions
    start = 0
    for i in range(1, partitions_count + 1):
        # Calculate partition size as a percentage of total disk space
        end = (i * 100) // partitions_count
        if i == partitions_count:
            end = 100  # Ensure the last partition takes up the remaining space
        subprocess.run(
            ["parted", image_path, "--script", "mkpart", "primary", f"{start}%", f"{end}%"],
            check=True,
        )
        start = end  # Update start for the next partition

    # Set up a loop device for the disk image
    loop_device = subprocess.run(
        ["sudo", "losetup", "--show", "-Pf", image_path], capture_output=True, text=True, check=True
    ).stdout.strip()

    # Format each partition with ext4
    for i in range(1, partitions_count + 1):
        subprocess.run(["sudo", "mkfs.ext4", "-F", f"{loop_device}p{i}"], check=True)

    # Detach loop device
    subprocess.run(["sudo", "losetup", "-d", loop_device], check=True)

    return image_path


def create_config(
    config_path: str,
    source: str,
    target: str,
    packages: list[str],
    partition_numbers: list[int],
    service_files: list[str] = [],
    target_directory: str = None,
    no_clone: bool = False,
    overwrite: bool = False,
) -> None:
    data = {
        "source": source,
        "target": target,
        "packages": packages,
        "partition-numbers": partition_numbers,
        "service_files": service_files,
        "target-directory": target_directory,
        "no-clone": no_clone,
        "overwrite": overwrite,
    }

    with open(config_path, "w") as f:
        json.dump(data, f)


def unmount_disk(device_path: str) -> None:
    """
    Unmount a disk device.

    Args:
        device_path (str): The path to the device to unmount.
    """
    # Check if the device is already mounted
    mounts = subprocess.check_output(["mount"]).decode()
    if device_path in mounts:
        subprocess.run(["sudo", "sync"], check=True)
        rc = subprocess.run(["sudo", "umount", device_path], check=False)
        while rc.returncode != 0:
            print(f"Failed to unmount {device_path}. Retrying...")
            sleep(0.1)
            rc = subprocess.run(["sudo", "umount", device_path], check=False)
    else:
        print(f"Device {device_path} is not mounted.")


def fill_disk_image_with_data(image_path: str, files_n: int = None) -> None:
    """
    Fills a disk image with random data files.

    Args:
        image_path (str): The path to the disk image file.
        files_n (int, optional): The number of random data files to create. If not specified, a random number between
                                 10 and 50 will be used.

    Raises:
        subprocess.CalledProcessError: If any of the subprocess commands fail.
    """
    return
    test_mount_point = os.path.abspath("test_data/mount_point")
    loop_device = make_image_mountable(image_path)
    if files_n is None:
        files_n = randint(10, 50)

    try:
        subprocess.run(["mkdir", "-p", test_mount_point], check=True)
        subprocess.run(["sudo", "mount", loop_device, test_mount_point], check=True)

        statvfs = os.statvfs(test_mount_point)
        available_size = statvfs.f_frsize * statvfs.f_bavail
        print(f"Available size in image {image_path}: {available_size} B")

        for i in range(files_n):
            # The size of the random data is between 0.5 and 0.9 of the image size and is divided by the number of files
            random_file_size = int(((random() / 10 * 4) + 0.5) * available_size) // files_n
            random_file_path = f"{test_mount_point}/random_data_{i}"

            subprocess.run(["sudo", "touch", random_file_path], check=True)
            subprocess.run(["sudo", "chmod", "666", random_file_path], check=True)
            with open(random_file_path, "wb") as f:
                f.write(os.urandom(random_file_size))

    finally:
        unmount_disk(test_mount_point)
        subprocess.run(["sudo", "rmdir", test_mount_point], check=True)
        subprocess.run(["sudo", "losetup", "-d", loop_device], check=True)


def fill_disk_images_with_data(image_paths: list[str], files_n: int = None) -> None:
    for image_path in image_paths:
        fill_disk_image_with_data(image_path, files_n)


def make_image_mountable(image_path: str) -> str:
    """
    Mounts a disk image as a loop device and returns the loop device path.

    Args:
        image_path (str): The path to the disk image file.

    Returns:
        str: The path to the created loop device.

    Raises:
        Exception: If the loop device creation fails.
    """

    loop_device = subprocess.run(["sudo", "losetup", "--show", "-Pf", image_path], capture_output=True, text=True)
    if loop_device.returncode != 0:
        raise Exception(f"Failed to create loop device: {loop_device.stderr}")

    return loop_device.stdout.strip()


def calculate_device_hash(device: str, chunk_size: int = 8192, device_coverage: float = 0.1) -> str:
    """
    Calculate the hash of a device.

    Args:
        device (str): The path to the device.
        chunk_size (int): The size of the chunks to read from the device.
        device_coverage (float): The percentage of the device to read.

    Returns:
        str: The hash of the device.
    """

    hash = xxhash.xxh64()
    device_size = int(subprocess.check_output(["sudo", "blockdev", "--getsize64", device]).strip())

    # Calculate the number of chunks to read when hashing device
    n_chunks = device_size * device_coverage // chunk_size
    chunk_jump = device_size // n_chunks
    offset = 0

    with subprocess.Popen(["sudo", "cat", device], stdout=subprocess.PIPE) as proc:
        while offset < device_size:
            chunk = proc.stdout.read(chunk_size)
            if not chunk:
                break
            hash.update(chunk)
            offset += chunk_jump

    return hash.hexdigest()


def is_partition_content_similar(partition: str, img: str) -> bool:
    """
    Compare the content of a partition with the content of an image.

    Args:
        partition (str): The path to the partition.
        img (str): The path to the image.

    Returns:
        bool: True if the content of the partition is similar to the content of the image, False otherwise.
    """

    print(f"Comparing {partition} with {img}")
    test_result = True
    loop_device = make_image_mountable(img)
    test_partition_mount_point = os.path.abspath("test_data/partition_mount_point")
    test_img_mount_point = os.path.abspath("test_data/img_mount_point")

    try:
        subprocess.run(["mkdir", "-p", test_partition_mount_point], check=True)
        subprocess.run(["mkdir", "-p", test_img_mount_point], check=True)
        subprocess.run(["sudo", "mount", partition, test_partition_mount_point], check=True)
        subprocess.run(["sudo", "mount", loop_device, test_img_mount_point], check=True)

        partition_files = []
        for root, _, files in os.walk(test_partition_mount_point):
            for name in files:
                partition_files.append(os.path.join(root, name))

        img_files = []
        for root, _, files in os.walk(test_img_mount_point):
            for name in files:
                img_files.append(os.path.join(root, name))

        partition_files = sorted(partition_files)
        img_files = sorted(img_files)
        if len(partition_files) != len(img_files):
            print(f"Number of files in partition and img are different: {len(partition_files)} != {len(img_files)}")
            test_result = False

        for f1, f2 in zip(partition_files, img_files):
            if not are_files_same(f1, f2):
                test_result = False
                break

    finally:
        subprocess.run(["sudo", "sync"], check=True)
        unmount_disk(test_partition_mount_point)
        unmount_disk(test_img_mount_point)
        subprocess.run(["sudo", "rmdir", test_partition_mount_point], check=True)
        subprocess.run(["sudo", "rmdir", test_img_mount_point], check=True)
        subprocess.run(["sudo", "losetup", "-d", loop_device], check=True)
    return test_result


def is_package_installed(package_path: str, mount_point: str, package_dir: str) -> bool:
    unzip_package_dir = os.path.abspath("test_data/unzip_package")
    subprocess.run(["mkdir", "-p", unzip_package_dir], check=True)

    subprocess.run(["sudo", "unzip", package_path, "-d", unzip_package_dir], check=True)

    mount_package_dir = os.path.join(mount_point, package_dir)

    return True


def inspect_image(config_path: str) -> bool:
    """TODO"""
    with open(config_path, "r") as f:
        config = json.load(f)

    image_path = config.get("target")
    partitions = config.get("partition-numbers")
    target_directory = config.get("target-directory")
    packages = config.get("packages")

    if not target_directory:
        target_directory = "."
    print(target_directory)

    if not os.path.exists(image_path):
        print(f"Image {image_path} does not exist.")
        return False

    loop_device = make_image_mountable(image_path)
    test_mount_point = os.path.abspath("test_data/inspect_mount_point")
    test_passed = True
    try:
        subprocess.run(["mkdir", "-p", test_mount_point], check=True)
        for partition in partitions:
            partition_path = f"{loop_device}p{partition}"
            subprocess.run(["sudo", "mount", partition_path, test_mount_point], check=True)
            print(f"Partition {partition} mounted at {test_mount_point}")
            for package in packages:
                if not is_package_installed(package, test_mount_point, target_directory):
                    test_passed = False
                    break
            unmount_disk(test_mount_point)

    finally:
        unmount_disk(test_mount_point)
        subprocess.run(["sudo", "rmdir", test_mount_point], check=True)
        subprocess.run(["sudo", "losetup", "-d", loop_device], check=True)

    return test_passed


def run_package_to_image_placer(
    package_to_image_placer_binary: str,
    source: str = None,
    target: str = None,
    config: bool = False,
    no_clone: bool = False,
    overwrite: bool = False,
    package_dir: bool = False,
    target_dir: bool = False,
    send_to_stdin: str = "",
    result_list: list = None,
) -> subprocess.CompletedProcess:

    parameters = [package_to_image_placer_binary]

    if source:
        parameters.append(f"-source")
        parameters.append(source)

    if target:
        parameters.append(f"-target")
        parameters.append(target)

    if config:
        parameters.append("-config")
        parameters.append(config)

    if no_clone:
        parameters.append("-no-clone")

    if overwrite:
        parameters.append("-overwrite")

    if package_dir:
        parameters.append("-package-dir")

    if target_dir:
        parameters.append("-target-dir")

    print(parameters)

    result = subprocess.Popen(
        parameters,
        stdin=subprocess.PIPE,
        stdout=subprocess.PIPE,
        stderr=subprocess.PIPE,
        text=True,
    )

    try:
        for char in send_to_stdin:
            result.stdin.write(char + "\n")
            result.stdin.flush()

    except Exception as e:
        print(f"Failed to send input to the process: {e}")

    stdout, stdin = result.communicate()

    # this outputs can be inspected when running pytest with -s flag
    print(stdout)
    print(stdin)

    if result_list is not None:
        result_list.append(result)

    return result
