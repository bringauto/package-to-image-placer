import subprocess
import os
from time import sleep
from test_utils.test_utils import (
    run_package_to_image_placer,
    create_test_package,
    create_image,
    make_image_mountable,
    create_config,
    inspect_image,
)


def test_app_shows_help(package_to_image_placer_binary):
    """TODO"""
    result = subprocess.run([package_to_image_placer_binary, "-h"], capture_output=True, text=True)

    assert result.stderr != ""
    assert result.returncode == 0


def test_write_one_package(package_to_image_placer_binary):
    """TODO"""
    config = "test_data/test_config.json"
    img_in = "test_data/test_img.img.in"
    img_out = "test_data/test_img_out.img"
    package = "test_data/normal_package"
    package_zip = package + ".zip"
    partitions = [1]

    create_test_package(package, "10KB")
    create_image(img_in, "10MB", 1)
    make_image_mountable(img_in)

    create_config(config, img_in, img_out, [package_zip], partitions)

    result = run_package_to_image_placer(package_to_image_placer_binary, config=config)

    assert result.returncode == 0

    assert inspect_image(config)


def test_write_two_packages(package_to_image_placer_binary):
    """TODO"""
    config = "test_data/test_config.json"
    img_in = "test_data/test_img.img.in"
    img_out = "test_data/test_img_out.img"
    package_1 = "test_data/normal_package_1"
    package_2 = "test_data/normal_package_2"
    partitions = [1]

    create_test_package(package_1, "15KB")
    create_test_package(package_2, "5KB")
    create_image(img_in, "10MB", 1)
    make_image_mountable(img_in)

    create_config(config, img_in, img_out, [package_1 + ".zip", package_2 + ".zip"], partitions)

    result = run_package_to_image_placer(package_to_image_placer_binary, config=config)

    assert result.returncode == 0

    assert inspect_image(config)


def test_write_multiple_packages_to_multiple_partitions(package_to_image_placer_binary):
    """TODO"""
    config = "test_data/test_config.json"
    img_in = "test_data/test_img.img.in"
    img_out = "test_data/test_img_out.img"

    package_1 = "test_data/normal_package1"
    package_2 = "test_data/normal_package2"
    package_3 = "test_data/normal_package3"
    package_4 = "test_data/normal_package4"
    package_5 = "test_data/normal_package5"

    partitions = [1, 2, 3, 4]

    create_test_package(package_1, "10KB")
    create_test_package(package_2, "10KB")
    create_test_package(package_3, "10KB")
    create_test_package(package_4, "10KB")
    create_test_package(package_5, "10KB")

    create_image(img_in, "10MB", 5)
    make_image_mountable(img_in)

    create_config(
        config,
        img_in,
        img_out,
        [package_1 + ".zip", package_2 + ".zip", package_3 + ".zip", package_4 + ".zip", package_5 + ".zip"],
        partitions,
    )

    result = run_package_to_image_placer(package_to_image_placer_binary, config=config)

    assert result.returncode == 0

    assert inspect_image(config)


def test_write_one_package_target_override(package_to_image_placer_binary):
    """TODO"""
    config = "test_data/test_config.json"
    img_in = "test_data/test_img.img.in"
    img_out = "test_data/test_img_out.img"
    package = "test_data/normal_package"
    package_zip = package + ".zip"
    partitions = [1]

    create_test_package(package, "10KB")
    create_image(img_in, "10MB", 1)
    make_image_mountable(img_in)

    with open(img_out, "w") as f:
        f.write("Test data")

    sleep(5)

    create_config(config, img_in, img_out, [package_zip], partitions, overwrite=False)

    result = run_package_to_image_placer(package_to_image_placer_binary, config=config)

    assert result.returncode == 0

    assert inspect_image(config)

    # sleep(100)


def test_write_one_package_no_clone(package_to_image_placer_binary):
    """FIXME"""
    config = "test_data/test_config.json"
    img_in = "test_data/test_img.img.in"
    img_out = "test_data/test_img_out.img"
    package = "test_data/normal_package"
    package_zip = package + ".zip"
    partitions = [1]

    create_test_package(package, "10KB")
    create_image(img_in, "10MB", 1)
    make_image_mountable(img_in)

    create_config(
        config, source=img_in, target=img_out, packages=[package_zip], partition_numbers=partitions, no_clone=True
    )

    result = run_package_to_image_placer(package_to_image_placer_binary, config=config)

    assert result.returncode == 0

    assert inspect_image(config)


# def test_app_finishes(target_disk_setup_binary):
#     """Test that the target disk setup application completes successfully when provided with valid arguments."""

#     bootloader = create_disk_image("test_data/bootloader.img", "100M", "ext4")
#     rootfs = create_disk_image("test_data/rootfs.img", "100M", "ext4")
#     recovery = create_disk_image("test_data/recovery.img", "100M", "ext4")
#     device = create_disk_image("test_data/sd32.img", "32G", "vfat")
#     loop_device = make_image_mountable(device)

#     result = run_target_disk_setup(
#         target_disk_setup_binary, bootloader, rootfs, loop_device, recovery=recovery, send_to_stdin="y"
#     )

#     assert result.returncode == 0


# def test_app_writes_to_disk(target_disk_setup_binary):
#     """Test that the target disk setup application completes successfully when provided with valid arguments (bootloader, rootfs) and checks if the data was written to the disk correctly."""

#     bootloader = create_disk_image("test_data/bootloader.img", "10M", "ext4")
#     rootfs = create_disk_image("test_data/rootfs.img", "50M", "ext4")
#     fill_disk_images_with_data([bootloader, rootfs])

#     device = create_disk_image("test_data/sd32.img", "32G", "vfat")
#     loop_device = make_image_mountable(device)

#     result = run_target_disk_setup(target_disk_setup_binary, bootloader, rootfs, loop_device, send_to_stdin="y")

#     assert result.returncode == 0

#     assert inspect_device(loop_device, bootloader, rootfs)


# def test_app_writes_to_disk_with_recovery(target_disk_setup_binary):
#     """Test that the target disk setup application completes successfully when provided with valid arguments (bootloader, rootfs, recovery) and checks if the data was written to the disk correctly."""

#     bootloader = create_disk_image("test_data/bootloader.img", "15M", "ext4")
#     rootfs = create_disk_image("test_data/rootfs.img", "60M", "ext4")
#     recovery = create_disk_image("test_data/recovery.img", "50M", "ext4")
#     fill_disk_images_with_data([bootloader, rootfs, recovery])

#     device = create_disk_image("test_data/sd32.img", "32G", "vfat")
#     loop_device = make_image_mountable(device)

#     result = run_target_disk_setup(
#         target_disk_setup_binary, bootloader, rootfs, loop_device, recovery=recovery, send_to_stdin="y"
#     )

#     assert result.returncode == 0

#     assert inspect_device(loop_device, bootloader, rootfs, recovery=recovery)


# def test_app_writes_to_disk_with_log_folder(target_disk_setup_binary):
#     """Test that the target disk setup application completes successfully when provided with valid arguments (bootloader, rootfs,v log_folder), checks if the data was written to the disk correctly and if non-empty log file was created."""

#     bootloader = create_disk_image("test_data/bootloader.img", "15M", "ext4")
#     rootfs = create_disk_image("test_data/rootfs.img", "60M", "ext4")
#     log_folder = "test_data"
#     fill_disk_images_with_data([bootloader, rootfs])

#     device = create_disk_image("test_data/sd32.img", "32G", "vfat")
#     loop_device = make_image_mountable(device)

#     result = run_target_disk_setup(
#         target_disk_setup_binary, bootloader, rootfs, loop_device, log_folder=log_folder, send_to_stdin="y"
#     )

#     log_file_path = f"{log_folder}/target_disk_setup.log"
#     assert os.path.exists(log_file_path)
#     # Check if the log file is not empty
#     with open(log_file_path, "r") as log_file:
#         assert log_file.read().strip() != ""

#     assert result.returncode == 0

#     assert inspect_device(loop_device, bootloader, rootfs)
