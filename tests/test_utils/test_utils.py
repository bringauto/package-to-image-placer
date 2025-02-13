import subprocess
import os
import shutil
import json
from random import random, randint
from time import sleep


def create_test_package(package_path: str, number_of_files: int) -> None:
    if os.path.exists(package_path):
        print(f"Package {package_path} already exists. Removing...")
        shutil.rmtree(package_path)

    os.makedirs(package_path)
    for i in range(number_of_files):
        with open(f"{package_path}/file_{i}", "w") as f:
            f.write(f"This is a test file {i}")

    subprocess.run(["zip", "-r", f"{package_path}.zip", package_path], check=True)


def create_disk_image(image_path: str, image_size: str, file_system: str) -> str:
    """
    Create a disk image with the specified size and file system.

    Args:
        image_path (str): The path where the disk image will be created.
        image_size (str): The size of the disk image.
        file_system (str): The file system type to be used for the disk image.

    Returns:
        str: The path to the created disk image.
    """

    if os.path.exists(image_path):
        os.remove(image_path)
    subprocess.run(["fallocate", "-l", f"{image_size}", image_path], check=True)
    if file_system == "vfat":
        subprocess.run(["mkfs.vfat", "-F", "32", image_path], check=True)
    else:
        subprocess.run(["mkfs", f"-t{file_system}", image_path], check=True)

    return image_path


# def create_config(config_path: str)


def unmount_disk(device_path: str) -> None:
    """
    Unmount a disk device.

    Args:
        device_path (str): The path to the device to unmount.
    """

    rc = subprocess.run(["sudo", "umount", device_path], check=False)
    while rc.returncode != 0:
        print(f"Failed to unmount {device_path}. Retrying...")
        sleep(0.1)
        rc = subprocess.run(["sudo", "umount", device_path], check=False)


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


def inspect_device(device_path: str, bootloader: str, rootfs: str, recovery: str = None) -> bool:
    """
    Inspect the content of a device.

    Args:
        device_path (str): The path to the device.
        bootloader (str): The path to the bootloader image.
        rootfs (str): The path to the rootfs image.
        recovery (str): The path to the recovery image.

    Returns:
        bool: True if the device content is similar to the provided images, False otherwise.
    """

    try:
        mount_device = subprocess.run(["sudo", "fdisk", "-l", device_path], capture_output=True, text=True)
        if mount_device.returncode != 0:
            raise Exception(f"Failed to inspect device: {mount_device.stderr}")

        partitions = []
        for line in mount_device.stdout.splitlines():
            if line.startswith(device_path):
                parts = line.split()
                partition_info = {
                    "name": parts[0],
                    "start": parts[1],
                    "end": parts[2],
                    "sectors": parts[3],
                    "size": parts[4],
                    "type": parts[5],
                }
                partitions.append(partition_info)

        # It should contains 4 partitions bootloader, A, B, recovery
        if len(partitions) != 4:
            raise Exception(f"Device {device_path} does not have 4 partitions")

        for partition in partitions:
            fs_check = subprocess.run(
                ["sudo", "blkid", "-o", "value", "-s", "TYPE", "-s", "PARTLABEL", partition["name"]],
                capture_output=True,
                text=True,
            )
            partition["fs_type"] = fs_check.stdout.splitlines()[0]
            partition["part_label"] = fs_check.stdout.splitlines()[1]
            if fs_check.returncode != 0:
                raise Exception(f"Failed to get filesystem info for {partition['name']}: {fs_check.stderr}")

            if not partition["fs_type"] in ["ext4"]:
                raise Exception(f"Partition {partition['name']} is not formatted as ext4")

            # Check the bootloader partition
            if partition["part_label"] == "msdos":
                if not is_partition_content_similar(partition["name"], bootloader):
                    raise Exception(f"Bootloader partition content is not similar to the provided bootloader image")

            # Check the rootfs partitions
            if partition["part_label"] in ["A", "B"]:
                if not is_partition_content_similar(partition["name"], rootfs):
                    raise Exception(f"Rootfs partition content is not similar to the provided rootfs image")

            # Check the recovery partition
            if partition["part_label"] == "recovery":
                if recovery is None:
                    if not is_partition_content_similar(partition["name"], rootfs):
                        raise Exception(f"Recovery partition content is not similar to the provided rootfs image")
                else:
                    if not is_partition_content_similar(partition["name"], recovery):
                        raise Exception(f"Recovery partition content is not similar to the provided recovery image")

    except Exception as e:
        print(f"Error inspecting device: {e}")
        return False

    return True


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
