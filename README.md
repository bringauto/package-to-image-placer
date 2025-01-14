# Package To Image Placer

This tool is used to place a package into a system image. It is used to create a disk image with a package installed.
This way the target system can be used with new packages without the need to generate a whole image and know yocto.

It takes a package (archive) and a system image (disk image) as input and creates a new disk image with the package.
If the package contains any service files, they can be activated.

For working with the image, the tool uses `libguestfs` to mount the image's filesystems.

The tool supports interactive mode, which allows the user to select the package, target partition, and the service files to activate.
Then it can generate a config file for the tool to use in non-interactive mode.

## Requirements

* golang >= 1.23
* libguestfs
    * See [libguest installation] section for install instructions.
* ssty

## Usage

### Build

```bash
go get package-to-image-placer
go build
```

### Run
For interactive mode, run:

```bash
./package-to-image-placer -source <source_image_path> -target <target_image_path> [ -package-dir <package_dir> ... ]
```

For non-interactive mode, run:

```bash
./package-to-image-placer -config <config_file_path> [ <overrides> ]
```

#### Arguments

* `-source` - Path to the source image.
* `-target` - Path to the target image. The path will be created. 
  * If used with no-clone option this file must exist and will be changed.
* `-config` - Path to the config file. Sets Non-interactive mode.
* `-no-clone` - Do not clone the source image. The target image must exist.
* `-overwrite` - Overwrite the target image if it exists.
* `-package-dir` - Initial directory for the package selection. Interactive mode only.
* `-target-dir` - Override target directory on the image from config. Non-interactive mode only.
* `-h` - Show usage.

> Commandline arguments are overriding the config file values.


## Libguest installation

Libquestfs is a library for modifying disk images. It is used to mount the disk image without root permissions.

### Debian based

```bash
sudo apt-get install libguestfs-tools
```

This will install the tools, but it needs read access to the kernel (e.g. `/boot/vmlinuz-*`) image. You can either
create a new kernel image, that `supermin` will use, or give read access to the existing one.
To test functionality or to find the used kernel image, run:

```bash
libguestfs-test-tool
```

Creating a new kernel image:

```bash
apt-get download linux-image-$(uname -r)
dpkg-deb -x linux-image-$(uname -r)*.deb ./
export SUPERMIN_KERNEL=./boot/vmlinuz-image-$(uname -r)
```

### Fedora

```bash
sudo dnf install libguestfs-tools
```