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
    create_package_config,
)


def test_01_write_package_with_service(package_to_image_placer_binary):
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

    create_config(config, img_in, img_out, [create_package_config(package_zip, True)], partitions)

    result = run_package_to_image_placer(package_to_image_placer_binary, config=config)

    assert result.returncode == 0

    assert inspect_image(config)


def test_02_write_package_with_services(package_to_image_placer_binary):
    """Test if the package_to_image_placer will write a package with two services to an image, it should fail because package can contain only one service"""
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

    create_config(config, img_in, img_out, [create_package_config(package_zip, True)], partitions)

    result = run_package_to_image_placer(package_to_image_placer_binary, config=config)

    assert result.returncode == 1
    assert not os.path.exists(img_out)


def test_03_write_packages_with_services_with_override(package_to_image_placer_binary):
    """Test if the package_to_image_placer will write two packages with same service while the second one has override enabled"""
    config = "test_data/test_config.json"
    img_in = "test_data/test_img.img.in"
    img_out = "test_data/test_img_out.img"
    package1 = "test_data/normal_package"
    package2 = "test_data/normal_package2"
    package1_zip = package1 + ".zip"
    package2_zip = package2 + ".zip"
    service1 = "test_data/service1.service"
    partitions = [1, 3]

    create_service_file(service1)
    create_test_package(package1, "10KB", services=[service1])
    create_test_package(package2, "10KB", services=[service1])
    create_image(img_in, "10MB", 3)
    make_image_mountable(img_in)

    create_config(
        config,
        img_in,
        img_out,
        [
            create_package_config(package1_zip, True),
            create_package_config(package2_zip, True, overwrite_file=["/normal_package2/service1.service"]),
        ],
        partitions,
    )

    result = run_package_to_image_placer(package_to_image_placer_binary, config=config)

    assert result.returncode == 0
    assert inspect_image(config)


def test_04_write_packages_with_services_without_override(package_to_image_placer_binary):
    """Test if the package_to_image_placer will fail when attempting to write two packages with the same service to an image without override enabled"""
    config = "test_data/test_config.json"
    img_in = "test_data/test_img.img.in"
    img_out = "test_data/test_img_out.img"
    package1 = "test_data/normal_package"
    package2 = "test_data/normal_package2"
    package1_zip = package1 + ".zip"
    package2_zip = package2 + ".zip"
    service1 = "test_data/service1.service"
    partitions = [1, 3]

    create_service_file(service1)
    create_test_package(package1, "10KB", services=[service1])
    create_test_package(package2, "10KB", services=[service1])
    create_image(img_in, "10MB", 3)
    make_image_mountable(img_in)

    create_config(
        config,
        img_in,
        img_out,
        [
            create_package_config(package1_zip, True),
            create_package_config(package2_zip, True),
        ],
        partitions,
    )

    result = run_package_to_image_placer(package_to_image_placer_binary, config=config)

    assert result.returncode == 1
    assert not os.path.exists(img_out)


def test_05_write_multiple_packages_with_services_to_single_dir(package_to_image_placer_binary):
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

    for _, (service, package) in enumerate(zip(services, packages)):
        print(package)
        create_service_file(service)
        create_test_package(package, "10KB", services=[service])
        create_config(
            config,
            target=img_in_out,
            packages=[create_package_config(package + ".zip", True, target_directory=target_directory)],
            partition_numbers=partitions,
            no_clone=True,
        )
        result = run_package_to_image_placer(package_to_image_placer_binary, config=config)
        assert result.returncode == 0
        assert inspect_image(config)

    create_config(
        config,
        target=img_in_out,
        packages=[
            create_package_config(package + ".zip", True, target_directory=target_directory) for package in packages
        ],
        partition_numbers=partitions,
    )
    assert inspect_image(config)


def test_06_write_multiple_packages_with_services_to_multiple_dirs(package_to_image_placer_binary):
    """Write multiple packages with services to an image"""
    config = "test_data/test_config.json"
    img_in_out = "test_data/test_img.img.in"
    package_count = 5
    packages = [f"test_data/normal_package{i}" for i in range(package_count)]
    services = [f"test_data/service{i}.service" for i in range(package_count)]
    target_directories = [f"a/b/c{i}" for i in range(package_count)]
    partitions = [1, 2, 3]

    create_image(img_in_out, "30MB", 3)
    make_image_mountable(img_in_out)

    for service, package, target_directory in zip(services, packages, target_directories):
        print(package)
        create_service_file(service)
        create_test_package(package, "10KB", services=[service])
        create_config(
            config,
            target=img_in_out,
            packages=[create_package_config(package + ".zip", True, target_directory=target_directory)],
            partition_numbers=partitions,
            no_clone=True,
        )
        result = run_package_to_image_placer(package_to_image_placer_binary, config=config)
        assert result.returncode == 0
        assert inspect_image(config)

    create_config(
        config,
        target=img_in_out,
        packages=[
            create_package_config(package + ".zip", True, target_directory=target_directory)
            for package, target_directory in zip(packages, target_directories)
        ],
        partition_numbers=partitions,
    )
    assert inspect_image(config)


def test_07_package_service_with_suffix(package_to_image_placer_binary):
    """Creates package with service file with suffix."""
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

    create_config(
        config,
        img_in,
        img_out,
        [
            create_package_config(
                package_zip,
                True,
                service_name_suffix="test",
            )
        ],
        partitions,
    )

    result = run_package_to_image_placer(package_to_image_placer_binary, config=config)

    assert result.returncode == 0

    assert inspect_image(config)


def test_08_package_with_service_with_suffix_starting_with_hyphen(
    package_to_image_placer_binary,
):
    """Creates package with service file with suffix starting with hyphen. It should crash because it's not to start service file with hyphen"""
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

    create_config(
        config,
        img_in,
        img_out,
        [
            create_package_config(
                package_zip,
                True,
                service_name_suffix="-test",
            )
        ],
        partitions,
    )

    result = run_package_to_image_placer(package_to_image_placer_binary, config=config)

    assert result.returncode == 1

    assert not os.path.exists(img_out)
