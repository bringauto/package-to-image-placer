import subprocess
import os
from test_utils.test_utils import (
    create_disk_image,
    make_image_mountable,
    run_target_disk_setup,
    inspect_device,
    fill_disk_images_with_data,
)


def test_app_shows_help(package_to_image_placer_binary):
    """TODO"""
    result = subprocess.run([package_to_image_placer_binary, "-h"], capture_output=True, text=True)

    assert result.stderr != ""
    assert result.returncode == 0


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
