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
    create_service_file,
)


def test_write_package_with_service(package_to_image_placer_binary):
    """TODO"""
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
    """TODO"""
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
    """TODO"""
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
    """TODO"""
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
