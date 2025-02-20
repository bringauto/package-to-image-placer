# Package To Image Placer

This tool is used to place a package into a system image without the need to use `sudo`.
It is used to create a disk image with a package installed.
This way the target system can be used with new packages without the need to generate a whole image and know yocto.

It takes a package (archive) and a system image (disk image) as input and creates a new disk image with the package.
If the package contains any service files, they can be activated.

For working with the image, the tool uses `libguestfs` to mount the image's filesystems. This way, the tool can work with the image without root permissions.

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

## Run

For interactive mode, run:

```bash
./package-to-image-placer -source=<source_image_path> -target=<target_image_path> [ -package-dir=<package_dir> ... ]
```

For non-interactive mode, run:

```bash
./package-to-image-placer -config=<config_file_path> [ <overrides> ]
```

In non-interactive mode, if the target image already exists, it will be modified. If the operation fails, the target image will be removed to prevent an inconsistent state.

When passing arguments through the command line, it is recommended to use the `-name=value` format when the equal sign is used.

### Arguments

* `-source` - Path to the source image.
* `-target` - Path to the target image. The path will be created and can't be same as source image path.
  * If used with no-clone option this file must exist and will be changed.
* `-config` - Path to the config file. Sets Non-interactive mode.
* `-no-clone` - Do not clone the source image. The target image must exist. If the operation fails, it may leave the image in an inconsistent state.
* `-overwrite` - Overwrite files in target image if it exists.
* `-package-dir` - Initial directory for the package selection. Interactive mode only.
* `-target-dir` - Override target directory on the image from config. Non-interactive mode only.
* `-log-path` - Path to the log file. Default is `./package_to_image_placer.log`.
* `-h` - Show usage.

> Commandline arguments are overriding the config file values.

### Tests

To run all tests, run:

```bash
go test ./...
```

## Config file

The config can be generated in interactive run. The config file is used to set the package, target partition, and service files to activate.

Default config file is in [default-config.json](./resources/default-config.json).

Config structure is as follows:

```json lines
{
  "source": "<source-image-path>",
  "target": "<target-image-path>",
  "packages": [
    "<package-path.zip>"
  ],
  "partition-numbers": [ 
    <partition-number>
  ],
  "service_files": [
    "<service-file-name-with-suffix>"
  ],
  "target-directory": "<target-directory-on-image>",
  "log-path": "<log-path>",
  "no-clone": <bool>,
  "overwrite": <bool>
}
```

## Libguest installation

Libquestfs is a library for modifying disk images. It is used to mount the disk image without root permissions.

### Debian based

```bash
sudo apt-get install libguestfs-tools

# Add user kernel, which can be accessed without sudo. Other option is to give permissions to the existing kernel.
apt-get download linux-image-$(uname -r)
dpkg-deb -x linux-image-$(uname -r)*.deb ./
export SUPERMIN_KERNEL=./boot/vmlinuz-image-$(uname -r)
```

`install libguestfs-tools` will install the tools, but it needs read access to the kernel (e.g. `/boot/vmlinuz-*`) image. 
You can either create a new kernel image, that `supermin` will use (as shown above), or give read access to the existing one.
To test functionality or to find the used kernel image, run:

```bash
libguestfs-test-tool
```


### Fedora

```bash
sudo dnf install libguestfs-tools
```