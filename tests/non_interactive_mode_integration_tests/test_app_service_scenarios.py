import os
from time import sleep
from test_utils.test_utils import (
    run_package_to_image_placer,
    create_test_package,
    create_image,
    make_image_mountable,
    create_config,
    inspect_image,
    create_service_file,
)


def test_write_package_with_service(package_to_image_placer_binary):
    """Test if the package_to_image_placer will write a package with a service to an image"""
    config = "test_data/test_config.json"
    img_in = "test_data/test_img.img.in"
    img_out = "test_data/test_img_out.img"
    package = "test_data/normal_package"
    package_zip = package + ".zip"
    service = "test_data/service.service"
    partitions = [1]

    create_service_file(service)
    create_test_package(package, "10KB", services=[service])
    create_image(img_in, "10MB", 1)
    make_image_mountable(img_in)

    create_config(config, img_in, img_out, [package_zip], partitions, service_names=[os.path.basename(service)])

    result = run_package_to_image_placer(package_to_image_placer_binary, config=config)

    assert result.returncode == 0

    assert inspect_image(config)


def test_write_package_with_services(package_to_image_placer_binary):
    """Test if the package_to_image_placer will write a package with two services to an image"""
    config = "test_data/test_config.json"
    img_in = "test_data/test_img.img.in"
    img_out = "test_data/test_img_out.img"
    package = "test_data/normal_package"
    package_zip = package + ".zip"
    service1 = "test_data/service1.service"
    service2 = "test_data/service2.service"
    partitions = [1]

    create_service_file(service1)
    create_service_file(service2)
    create_test_package(package, "10KB", services=[service1, service2])
    create_image(img_in, "10MB", 1)
    make_image_mountable(img_in)

    create_config(
        config,
        img_in,
        img_out,
        [package_zip],
        partitions,
        service_names=[os.path.basename(service1), os.path.basename(service2)],
    )

    result = run_package_to_image_placer(package_to_image_placer_binary, config=config)

    assert result.returncode == 0
    assert inspect_image(config)


def test_write_packages_with_services_with_override(package_to_image_placer_binary):
    """Test if the package_to_image_placer will write two packages with two services to an image when override is set to True"""
    config = "test_data/test_config.json"
    img_in = "test_data/test_img.img.in"
    img_out = "test_data/test_img_out.img"
    package1 = "test_data/normal_package"
    package2 = "test_data/normal_package2"
    package1_zip = package1 + ".zip"
    package2_zip = package2 + ".zip"
    service1 = "test_data/service1.service"
    service2 = "test_data/service2.service"
    partitions = [1, 3]

    create_service_file(service1)
    create_service_file(service2)
    create_test_package(package1, "10KB", services=[service1, service2])
    create_test_package(package2, "10KB", services=[service1, service2])
    create_image(img_in, "10MB", 3)
    make_image_mountable(img_in)

    create_config(
        config,
        img_in,
        img_out,
        [package1_zip, package2_zip],
        partitions,
        service_names=[os.path.basename(service1), os.path.basename(service2)],
        overwrite=True,
    )

    result = run_package_to_image_placer(package_to_image_placer_binary, config=config)

    assert result.returncode == 0
    assert inspect_image(config)


def test_write_packages_with_services_without_override(package_to_image_placer_binary):
    """Test if the package_to_image_placer will fail when attempting to write two packages with the same service to an image without override enabled"""
    config = "test_data/test_config.json"
    img_in = "test_data/test_img.img.in"
    img_out = "test_data/test_img_out.img"
    package1 = "test_data/normal_package"
    package2 = "test_data/normal_package2"
    package1_zip = package1 + ".zip"
    package2_zip = package2 + ".zip"
    service = "test_data/service1.service"
    partitions = [1]

    create_service_file(service)
    create_test_package(package1, "10KB", services=[service])
    create_image(img_in, "10MB", 1)
    make_image_mountable(img_in)

    create_config(
        config,
        img_in,
        img_out,
        [package1_zip, package2_zip],
        partitions,
        service_names=[os.path.basename(service)],
    )

    result = run_package_to_image_placer(package_to_image_placer_binary, config=config)

    assert result.returncode == 1
    assert not os.path.exists(img_out)


def test_write_multiple_packages_with_services(package_to_image_placer_binary):
    """Write multiple packages with services to an image"""
    config = "test_data/test_config.json"
    img_in_out = "test_data/test_img.img.in"
    package_count = 5
    packages = [f"test_data/normal_package{i}" for i in range(package_count)]
    services = [f"test_data/service{i}.service" for i in range(package_count)]
    target_directory = "a/b/c"
    partitions = [1, 2, 3]

    create_image(img_in_out, "30MB", 3)
    make_image_mountable(img_in_out)

    for service, package in zip(services, packages):
        print(package)
        create_service_file(service)
        create_test_package(package, "10KB", services=[service])
        create_config(
            config,
            target=img_in_out,
            packages=[package + ".zip"],
            partition_numbers=partitions,
            service_names=[os.path.basename(service)],
            no_clone=True,
            target_directory=target_directory,
        )
        result = run_package_to_image_placer(package_to_image_placer_binary, config=config)
        assert result.returncode == 0
        assert inspect_image(config)

    create_config(
        config,
        target=img_in_out,
        packages=[package + ".zip" for package in packages],
        partition_numbers=partitions,
        service_names=[os.path.basename(service) for service in services],
        target_directory=target_directory,
    )
    assert inspect_image(config)


def test_package_with_multiple_services(package_to_image_placer_binary):
    """Creates package with multiple service files. It should crash because it's not supported"""
    return


def test_package_service_with_suffix(package_to_image_placer_binary):
    """Creates package with service file with suffix."""
    return


def test_package_with_service_with_suffix_starting_with_hyphen(
    package_to_image_placer_binary,
):
    """Creates package with service file with suffix starting with hyphen. It should crash because it's not to start service file with hyphen"""
    return
