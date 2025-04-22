import subprocess
import os
import shutil
import json


def remove_dir(dir_path: str) -> None:
    """
    Remove a directory and its contents.

    Args:
        dir_path (str): The path to the directory to remove.
    """
    if not dir_path:
        raise ValueError("Directory path is empty or None.")

    if not os.path.exists(dir_path):
        return

    dir_path = os.path.abspath(dir_path)
    allowed_root = os.path.abspath("test_data")
    if os.path.commonpath([dir_path, allowed_root]) != allowed_root:
        raise ValueError("Refusing to remove directory outside of tests/test_data.")

    subprocess.run(["sudo", "rm", "-rf", dir_path], check=True)


def mkdir(dir_path: str) -> None:
    """
    Create a directory.

    Args:
        dir_path (str): The path to the directory to create.
    """
    subprocess.run(["sudo", "mkdir", "-p", dir_path], check=True)


def convert_size_to_bytes(size: str) -> int:
    """
    Convert a size string to bytes.

    Args:
        size (str): The size string to convert (e.g., '5MB').

    Returns:
        int: The size in bytes.
    """
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


def create_symlink(source: str, target: str) -> None:
    """
    Create a symbolic link.

    Args:
        source (str): The path to the source file or directory.
        target (str): The path to the target symbolic link.
    """
    source = os.path.abspath(source)
    if os.path.exists(target):
        os.remove(target)
    os.symlink(source, target)


def create_test_package(
    package_path: str, package_size: str, create_symlinks: bool = True, services: list[str] = None
) -> None:
    """
    Create a test package with the specified size and contents.

    Args:
        package_path (str): The path to the package zip file to create.
        package_size (str): The size of the package (e.g., '5MB').
        create_symlinks (bool): Whether to create symbolic links in the package directory.
        services (list[str]): A list of service files to include in the package.
    """
    if os.path.exists(package_path):
        print(f"Package {package_path} already exists. Removing...")
        remove_dir(package_path)

    package_size_bytes = convert_size_to_bytes(package_size)

    os.makedirs(package_path)

    with open(f"{package_path}/test_file", "wb") as f:
        f.write(os.urandom(package_size_bytes))

    if create_symlinks:
        os.makedirs(f"{package_path}/symlinks")
        create_symlink(f"{package_path}/test_file", f"{package_path}/symlinks/symlink")

    for service in services or []:
        service_target_path = os.path.join(package_path, os.path.basename(service))
        shutil.copy(service, service_target_path)

    subprocess.run(
        ["zip", "-ry", f"{os.path.abspath(package_path)}.zip", os.path.basename(package_path)],
        check=True,
        cwd=os.path.dirname(package_path),
    )


def add_services_dirs_to_partition(partition: str) -> None:
    """
    Add the required directories for services to a partition.

    Args:
        partition (str): The path to the partition to add the directories to.
    """
    services_dir = "/etc/systemd/system/"
    multi_user_target = "/etc/systemd/system/multi-user.target.wants/"
    mount_point = "test_data/services_mount_point"
    os.makedirs(mount_point, exist_ok=True)

    try:
        subprocess.run(["sudo", "mount", partition, mount_point], check=True)
        mkdir(f"{mount_point}{services_dir}")
        mkdir(f"{mount_point}{multi_user_target}")

    finally:
        unmount_disk(mount_point)


def create_image(image_path: str, image_size: str, partitions_count: int) -> str:
    """
    Create a disk image with the specified size and partition table (DOS/MBR).

    Args:
        image_path (str): The path where the disk image will be created.
        image_size (str): The size of the disk image (e.g., '5MB').
        partitions_count (int): The number of partitions to create. Partition size will be calculated based on this value and the total disk size.

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
        partition_name = f"{loop_device}p{i}"
        subprocess.run(["sudo", "mkfs.ext4", "-F", partition_name], check=True)
        add_services_dirs_to_partition(partition_name)

    # Detach loop device
    subprocess.run(["sudo", "losetup", "-d", loop_device], check=True)

    return image_path


def create_normal_package_config(
    package_path: str,
    enable_services: bool = False,
    service_name_suffix: str = "",
    target_directory: str = "/",
    overwrite_file: list[str] = None,
) -> dict:
    """
    Create a package configuration dictionary.

    Args:
        package_path (str): The path to the package zip file.
        enable_services (bool, optional): Whether to enable services. Defaults to False.
        service_name_suffix (str, optional): The suffix for the service names. Defaults to an empty string.
        target_directory (str, optional): The target directory for the package. Defaults to "/".
        overwrite_file (list[str], optional): A list of files to overwrite in the package. Defaults to None.

    Returns:
        dict: The package configuration dictionary.
    """
    package_config = {
        "package-path": os.path.abspath(package_path),
        "enable-services": enable_services,
        "service-name-suffix": service_name_suffix,
        "target-directory": target_directory,
        "overwrite-files": overwrite_file if overwrite_file is not None else [],
    }
    return package_config


def create_configuration_package_config(
    package_path: str,
    overwrite_file: list[str] = None,
) -> dict:
    """
    Create a configuration package configuration file.
    Args:
        package_path (str): The path to the package zip file.
        overwrite_file (list[str]): A list of files to overwrite in the package.
    Returns:
        dict: The configuration package configuration.
    """
    package_config = {
        "package-path": os.path.abspath(package_path),
        "overwrite-files": overwrite_file if overwrite_file is not None else [],
    }
    return package_config


def create_config(
    config_path: str,
    source: str = "",
    target: str = "",
    packages: list[dict] = None,
    partition_numbers: list[int] = None,
    configuration_packages: list[str] = None,
    no_clone: bool = False,
    log_path: str = "",
    remove_from_config: list[str] = None,
) -> None:
    """
    Create a configuration file for the package-to-image-placer.

    Args:
        config_path (str): The path to the configuration file to create.
        source (str): The path to the source image.
        target (str): The path to the target image.
        packages (list[dict]): A list of package configurations.
        configuration_packages (list[str]): A list of configuration packages.
        partition_numbers (list[int]): A list of partition numbers.
        no_clone (bool): Whether to clone the source image.
        log_path (str): The path to the log file.
        remove_from_config (list[str]): A list of keys to remove from the configuration.
    """

    data = {
        "source": os.path.abspath(source) if source else "",
        "target": os.path.abspath(target) if target else "",
        "packages": packages if packages is not None else [],
        "configuration-packages": configuration_packages if configuration_packages is not None else [],
        "partition-numbers": partition_numbers if partition_numbers is not None else [],
        "no-clone": no_clone,
        "log-path": os.path.abspath(log_path) if log_path else "",
    }

    for key in remove_from_config or []:
        data.pop(key, None)

    if os.path.exists(config_path):
        print(f"Config file {config_path} already exists. Removing...")
        os.remove(config_path)

    with open(config_path, "w") as f:
        json.dump(data, f, indent=4)

    print(data)


def create_service_file(service_file_path: str) -> None:
    """
    Create a service file for testing.

    Args:
        service_file_path (str): The path to the service file to create.
    """
    service_content = """
[Unit]
Description=My Custom Testing Service
After=network.target

[Service]
ExecStart=test_file arguments
Type=simple
User=root
WorkingDirectory=.
Restart=always
RestartSec=1

[Install]
WantedBy=multi-user.target

"""

    if os.path.exists(service_file_path):
        print(f"Service file {service_file_path} already exists. Removing...")
        os.remove(service_file_path)

    with open(service_file_path, "w") as f:
        f.write(service_content)


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
        attempts = 0
        rc = subprocess.run(["sudo", "umount", device_path], check=False)
        while rc.returncode != 0 and attempts < 5:
            attempts += 1
            print(f"Failed to unmount {device_path}. Retrying ({attempts}/5)...")
            rc = subprocess.run(["sudo", "umount", device_path], check=False)
        if rc.returncode != 0:
            raise RuntimeError(f"Unable to unmount {device_path} after {attempts} attempts")
    else:
        print(f"Device {device_path} is not mounted.")


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


def compare_directories(dir1: str, dir2: str) -> bool:
    """
    Compare two directories recursively.

    Args:
        dir1 (str): The path to the first directory.
        dir2 (str): The path to the second directory.

    Returns:
        bool: True if the directories are the same, False otherwise.
    """
    files1 = []
    files2 = []
    dirs1 = []
    dirs2 = []

    if not os.path.exists(dir1) or not os.path.exists(dir2):
        print(f"One of the directories does not exist: {dir1}, {dir2}")
        return False

    if not os.path.isdir(dir1) or not os.path.isdir(dir2):
        print(f"One of the paths is not a directory: {dir1}, {dir2}")
        return False

    with os.scandir(dir1) as entries:
        for entry in entries:
            if entry.is_file():
                files1.append(entry.name)
            elif entry.is_dir():
                dirs1.append(entry.name)

    with os.scandir(dir2) as entries:
        for entry in entries:
            if entry.is_file():
                files2.append(entry.name)
            elif entry.is_dir():
                dirs2.append(entry.name)

    files1 = [f for f in files1 if not f.endswith(".service")]
    files2 = [f for f in files2 if not f.endswith(".service")]
    files1.sort()
    files2.sort()

    if len(files1) != len(files2):
        print(f"Number of files differ: {len(files1)} vs {len(files2)}")
        return False

    for f1, f2 in zip(files1, files2):
        rc = subprocess.run(
            ["diff", "-q", os.path.join(dir1, f1), os.path.join(dir2, f2)], check=False, capture_output=True
        )
        if rc.returncode != 0:
            print(
                f"Differences found between {os.path.join(dir1, f1)} and {os.path.join(dir2, f2)}: {rc.stdout.strip()}"
            )
            return False

    dirs1.sort()
    dirs2.sort()
    for d1, d2 in zip(dirs1, dirs2):
        if not compare_directories(os.path.join(dir1, d1), os.path.join(dir2, d2)):
            return False

    return True


def is_package_installed(package: dict, mount_point: str, configuration_package=False) -> bool:
    """
    TODO;
    """
    package_path = package["package-path"]
    package_dir = package.get("target-directory", "/")
    print(f"Checking package {package_path} installation...")
    unzip_package_dir = os.path.abspath("test_data/unzip_package")
    if os.path.exists(unzip_package_dir):
        remove_dir(unzip_package_dir)
    os.makedirs(unzip_package_dir, exist_ok=True)

    subprocess.run(["sudo", "unzip", "-o", package_path, "-d", unzip_package_dir], check=True)

    if package_dir.startswith("/"):
        package_dir = package_dir[1:]
    packet_name = os.path.basename(package_path).split(".")[0]
    mount_package_dir = os.path.join(mount_point, package_dir, packet_name)
    if configuration_package:
        unzip_package_dir = os.path.join(unzip_package_dir, packet_name)

    print(package_dir, packet_name)
    print("Diffing: ", unzip_package_dir, mount_package_dir)

    if not compare_directories(unzip_package_dir, mount_package_dir):
        return False

    if not package.get("enable-services", False):
        return True

    service_files = [
        os.path.join(root, file)
        for root, _, files in os.walk(unzip_package_dir)
        for file in files
        if file.endswith(".service")
    ]

    if len(service_files) != 1:
        print(f"Found {len(service_files)} service files in the package. And there should be one.")
        return False

    service = service_files[0]

    if package.get("enable-services"):

        if package["service-name-suffix"] != "":
            service = ".".join(service.split(".")[:-1]) + "-" + package["service-name-suffix"] + ".service"

        print(f"Checking if service {service} is enabled...")

        return is_service_enabled(service, mount_point)

    return True


def is_service_enabled(service_name: str, mount_point: str) -> bool:
    """
    Check if a service is enabled in the target image.

    Args:
        service_name (str): The name of the service file.
        mount_point (str): The path to the mount point.

        Returns:
        bool: True if the service is enabled, False otherwise.
    """
    system_services_dir = "etc/systemd/system/"
    multi_user_target_dir = "etc/systemd/system/multi-user.target.wants/"

    service_name = os.path.basename(service_name)
    service_path = os.path.join(mount_point, system_services_dir, service_name)

    if not os.path.exists(service_path):
        print(f"Service file {service_name} not found.")
        return False

    service_link_path = os.path.join(mount_point, multi_user_target_dir, service_name)
    if not os.path.exists(service_link_path):
        print(f"Service link {service_link_path} not found.")
        return False

    if not os.path.islink(service_link_path):
        print(f"Service link {service_link_path} is not a symlink.")
        return False

    link_target = os.path.abspath(
        os.path.join(mount_point, system_services_dir, os.path.basename(os.readlink(service_link_path)))
    )
    if link_target != service_path:
        print(f"Service link: {service_link_path} target is: {link_target} instead of: {service_path}.")
        return False

    with open(service_path, "r") as f:
        required_fields = {
            "ExecStart=",
            "WorkingDirectory=",
            "User=",
            "RestartSec=",
            "Type=simple",
            "WantedBy=multi-user.target",
        }

        for line in f:
            if line.startswith("ExecStart="):
                exec_start = line.split("=")[1].split()[0].strip().strip("/")
                if not os.path.exists(os.path.join(mount_point, exec_start)):
                    print(f"Service {service_name} executable not found: {exec_start}")
                    return False

                args_count = len(line.split("=")[1].split())
                if args_count != 2:
                    print(f"Service {service_name} has not exactly one argument: {args_count}")
                    return False

            elif line.startswith("WorkingDirectory="):
                working_dir = line.split("=")[1].strip().strip("/")
                if not os.path.exists(os.path.join(mount_point, working_dir)):
                    print(f"Service {service_name} working directory not found: {working_dir}")
                    return False

            required_fields = {f for f in required_fields if not line.startswith(f)}

        if required_fields:
            print(f"Service {service_name} is missing required fields: {required_fields}")
            return False

    print(f"Service {service_name} is enabled and properly configured.")
    return True


def inspect_image(config_path: str) -> bool:
    """
    Inspect the target image to verify that the packages and services are installed correctly.

    Args:
        config_path (str): The path to the configuration file.

    Returns:
        bool: True if the image passes the inspection, False otherwise.
    """

    with open(config_path, "r") as f:
        config = json.load(f)

    image_path = config.get("target")
    partitions = config.get("partition-numbers")
    packages = config.get("packages")
    configuration_packages = config.get("configuration-packages")

    if not os.path.exists(image_path):
        print(f"Image {image_path} does not exist.")
        return False

    loop_device = make_image_mountable(image_path)
    test_mount_point = os.path.abspath("test_data/inspect_mount_point")
    test_passed = True
    try:
        mkdir(test_mount_point)
        for partition in partitions:
            partition_path = f"{loop_device}p{partition}"
            subprocess.run(["sudo", "mount", partition_path, test_mount_point], check=True)
            print(f"Partition {partition} mounted at {test_mount_point}")
            for package in packages:
                print(f"Partition: {partition}, Package: {package}")
                if not is_package_installed(package, test_mount_point):
                    test_passed = False
                    print(f"Package {package} not properly installed.")
                    break
            for package in configuration_packages:
                print(f"Partition: {partition}, Configuration Package: {package}")
                if not is_package_installed(package, test_mount_point, True):
                    test_passed = False
                    print(f"Configuration package {package} not properly installed.")
                    break

            unmount_disk(test_mount_point)

    finally:
        remove_dir(test_mount_point)
        subprocess.run(["sudo", "losetup", "-d", loop_device], check=True)

    return test_passed


def run_package_to_image_placer(
    package_to_image_placer_binary: str,
    source: str = None,
    target: str = None,
    config: str = None,
    no_clone: bool = None,
    package_dir: str = None,
    log_path: str = ".",
    send_to_stdin: str = None,
    result_list: list = None,
) -> subprocess.CompletedProcess:
    """
    Run the package-to-image-placer application with the specified parameters.

    Args:
        package_to_image_placer_binary (str): The path to the package-to-image-placer binary.
        source (str, optional): The path to the source image.
        target (str, optional): The path to the target image.
        config (str, optional): The path to the configuration file.
        no_clone (bool, optional): Whether to clone the source image.
        package_dir (str, optional): The directory containing the packages to install.
        log_path (str, optional): The path to the log file. Defaults to ".".
        send_to_stdin (str, optional): The input to send to the process.
        result_list (list, optional): A list to append the result to.

    Returns:
        subprocess.CompletedProcess: The result of the process execution.
    """

    parameters = [package_to_image_placer_binary]

    if source is not None:
        parameters.append(f"-source={source}")

    if target is not None:
        parameters.append(f"-target={target}")

    if config is not None:
        parameters.append(f"-config={config}")

    if no_clone is not None:
        parameters.append(f"-no-clone={no_clone}")

    if package_dir is not None:
        parameters.append(f"-package-dir={package_dir}")

    if log_path is not None:
        parameters.append(f"-log-path={log_path}")

    print(parameters)

    result = subprocess.Popen(
        parameters,
        stdin=subprocess.PIPE,
        stdout=subprocess.PIPE,
        stderr=subprocess.PIPE,
        text=True,
    )

    try:
        if send_to_stdin is not None:
            for char in send_to_stdin:
                result.stdin.write(char + "\n")
                result.stdin.flush()

    except Exception as e:
        print(f"Failed to send input to the process: {e}")

    stdout, stderr = result.communicate()

    # this outputs can be inspected when running pytest with -s flag
    print(stdout)
    print(stderr)

    if result_list is not None:
        result_list.append(result)

    return result
