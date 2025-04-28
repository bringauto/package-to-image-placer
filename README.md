
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

⚠️ In non-interactive mode, if the target image already exists, it will be modified. If the operation fails, the target image will be removed to prevent an inconsistent state.

When passing arguments through the command line, it is recommended to use the `-name=value` format when the equal sign is used.

### Arguments

* `-source` - Path to the source image.
* `-target` - Path to the target image. The path will be created and can't be same as source image path.
  * If used with no-clone option this file must exist and will be changed.
* `-config` - Path to the config file. Sets Non-interactive mode.
* `-no-clone` - Do not clone the source image. The target image must exist. If the operation fails, it may leave the image in an inconsistent state.
* `-package-dir` - Initial directory for the package selection. Interactive mode only.
* `-log-path` - Directory for the log file. Default is the current directory (`.`). The log file will be created at `log-path/package-to-image-placer.log`.
* `-h` - Show usage.

> Command line arguments are overriding the config file values.

For the mock use case, please consult the [Use Case Example Documentation](doc/UseCaseExample.md).

## Tests

### Unit Tests

To run all unit tests, run:

```bash
go test ./...
```

### Integration Tests

To run integration tests, refer to the [tests/README.md](tests/README.md) file.

## Config File

The configuration for the image customization process can be generated through an interactive run. Alternatively, you can directly edit a configuration file to specify the packages to be applied, the target partitions for these changes, and the service files that should be activated.

The example configuration file is located at [example_config.json](./resources/example_config.json).

The structure of the configuration file is defined in JSON format as follows:

```json lines
{
  "source": "<source-image-path>",
  "target": "<target-image-path>",
  "no-clone": "<bool>",
  "packages": [
    {
     "package-path": "package-path.zip",
     "enable-services": "<bool>",
     "service-name-suffix": "<service-name-suffix>",
     "target-directory": "<target-directory>",
     "overwrite-files": [
       "<file-name-1>",
       "<file-name-2>"
     ]
    }
  ],
  "partition-numbers": [
    "<partition-number>"
  ],
  "configuration-packages": [
    {
     "package-path": "configuration-package-path.zip",
     "overwrite-files": [
       "<file-name-1>",
       "<file-name-2>"
     ]
    }
  ],
  "log-path": "<log-path>",
}
```

* The `source` or `target` in the case of `-no-clone` must be a valid image file using a GPT partition table and must have at least one partition with an Ext4 filesystem, as the tool can only write to this filesystem. If you want to enable services from the copied package, the destination partition must contain the directories `/etc/systemd/system/` and `/etc/systemd/system/multi-user.target.wants/`, where the service files will be copied.
* The `packages` and `configuration-packages` must be valid zip files containing the files to be copied to the image. Additionally, a package can contain a service file that can be activated in the image. The service file must be included in the package and must have a `.service` extension. If a configuration package contains a service file, it is processed as a normal file and is simply copied to the image, not activated as a service.
* The `service-name-suffix` is used to add a suffix to the service file name and thus avoid name conflicts. The suffix is added to the service file name in the image. For example, if the service file name is `my-service.service` and the suffix is `test`, the service file name in the image will be `my-service-test.service`. The suffix must not start with a hyphen.
* The `overwrite-files` paths are relative to their location within the package zip file. If a file already exists in the image and is not listed under `overwrite-files`, an error will occur. However, if the file is included in `overwrite-files`, it will be copied to the image, overwriting the existing file regardless of its presence.
* The `partition-numbers` must be valid partition numbers in the image. The partition numbers are 1-based, meaning the first partition is 1, the second is 2, and so on.
* Paths in the configuration file can be absolute or relative to the location of the configuration file.
* The difference between package and configuration packages is that the configuration packages are not placed in the specified directory with the package name, and are always placed into the root of the image and services from them cannot be activated.

## Services

The tool can activate service files in the image.
The service files are activated by copying them to `/etc/systemd/system/` and creating a symlink to the file in `/etc/systemd/system/multi-user.target.wants/`.

The paths in the image are updated based on the `WorkingDirectory` field, where the original WorkingDirectory is replaced with the new path in the target image.

### Service Requirements

The service file must:

* be in the package.
* service file name must end with `.service`.
* multiple services for the same package are not supported.
* service file suffix must not start with a hyphen.
* contain the following fields:
  * `ExecStart`
  * `User`
  * `RestartSec`
  * `WorkingDirectory`
  * `Type=simple`
  * `WantedBy=multi-user.target`

## Libguest Installation

Libquestfs is a library for modifying disk images. It is used to mount the disk image without root permissions.

### Debian Based

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

## Troubleshooting

### Libvirt Error

If you have error that look like this:

``` text
libguestfs: error: could not create appliance through libvirt. 
Original error from libvirt: internal error: 
process exited while connecting to monitor: 2025-03-03T14:51:53.981133Z qemu-kvm: -device {"driver":"scsi-hd","bus":"scsi0.0","channel":0,"scsi-id":0,"lun":0,"device_id":"drive-scsi0-0-0-0","drive":"libvirt-2-storage","id":"scsi0-0-0-0","bootindex":1,"write-cache":"on"}: Failed to get "write" lock
        Is another process using the image?
```

Make sure you have `kvm` enabled.
