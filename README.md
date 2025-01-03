# Package To Image Placer

## Requirements

* golang >= 1.23
* libguestfs
    * See [libguest installation] section for install instructions.
* zip
* ssty

## Libguest installation

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