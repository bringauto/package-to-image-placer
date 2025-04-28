#!/bin/bash

# Define the image file name and size
image_file="testing_image.img"
image_size_mb=8
partition1_size_mb=3
partition2_size_mb=3
image_size_bytes=$((image_size_mb * 1024 * 1024))

# Create an empty file
echo "Creating an empty image file: $image_file (${image_size_mb}MB)"
truncate -s "$image_size_bytes" "$image_file"

# Create a loopback device
echo "Setting up loopback device..."
loop_device=$(sudo losetup --find --show "$image_file")
if [ -z "$loop_device" ]; then
  echo "Error: Failed to set up loopback device."
  exit 1
fi
echo "Loopback device: $loop_device"

# Create GPT and partitions
echo "Creating GPT partition table using sgdisk..."
sudo sgdisk --zap-all "$loop_device"
sudo sgdisk --new=1:0:+${partition1_size_mb}M --typecode=1:8300 "$loop_device"
sudo sgdisk --new=2:0:+${partition2_size_mb}M --typecode=2:8300 "$loop_device"
sudo sgdisk --print "$loop_device"

# Reload partition table
sudo partprobe "$loop_device"
if [ $? -ne 0 ]; then
  echo "Error: Failed to create partition table using sgdisk."
  sudo losetup -d "$loop_device"
  rm "$image_file"
  exit 1
fi


echo "Verifying partitions..."
partition1="${loop_device}p1"
partition2="${loop_device}p2"
if [ ! -e "$partition1" ] || [ ! -e "$partition2" ]; then
  echo "Error: Partitions not found. Please check the loopback device."
  sudo losetup -d "$loop_device"
  rm "$image_file"
  exit 1
fi

echo "Partition 1: $partition1"
echo "Partition 2: $partition2"

# Format the partitions as Ext4
echo "Formatting partitions as Ext4..."
sudo mkfs.ext4 -F -O ^has_journal "$partition1"
sudo mkfs.ext4 -F -O ^has_journal "$partition2"

if [ $? -ne 0 ]; then
  echo "Error: Failed to format partitions."
  sudo losetup -d "$loop_device"
  rm "$image_file"
  exit 1
fi


# Add directories for services
echo "Mounting partitions..."
mount_dir1=$(mktemp -d)
mount_dir2=$(mktemp -d)

sudo mount "$partition1" "$mount_dir1"
sudo mount "$partition2" "$mount_dir2"

if [ $? -ne 0 ]; then
    echo "Error: Failed to mount partitions."
    sudo umount "$mount_dir1" "$mount_dir2" 2>/dev/null
    rmdir "$mount_dir1" "$mount_dir2" 2>/dev/null
    sudo losetup -d "$loop_device"
    rm "$image_file"
    exit 1
fi

echo "Adding '/etc/systemd/system/multi-user.target.wants/' to each partition..."
sudo mkdir -p "$mount_dir1/etc/systemd/system/multi-user.target.wants/"
sudo mkdir -p "$mount_dir2/etc/systemd/system/multi-user.target.wants/"

# Unmount the partitions
echo "Unmounting partitions..."
sudo umount "$mount_dir1"
sudo umount "$mount_dir2"

if [ $? -ne 0 ]; then
    echo "Error: Failed to unmount partitions."
    sudo losetup -d "$loop_device"
    rm "$image_file"
    exit 1
fi

# Clean up temporary mount directories
rmdir "$mount_dir1" "$mount_dir2"

echo "Image '$image_file' created and partitioned successfully."
echo "Partitions are: $partition1 and $partition2"

echo "Detaching loopback device..."
sudo losetup -d "$loop_device"

exit 0