import subprocess
import os
from test_utils.test_utils import (
    run_package_to_image_placer,
    create_test_package,
    create_image,
    make_image_mountable,
    create_config,
    inspect_image,
)


def test_write_one_big_package(package_to_image_placer_binary):
    """TODO"""
    config = "test_data/test_config.json"
    img_in = "test_data/test_img.img.in"
    img_out = "test_data/test_img_out.img"
    package = "test_data/normal_package"
    package_zip = package + ".zip"
    partitions = [1]

    create_test_package(package, "11MB")
    create_image(img_in, "10MB", 1)
    make_image_mountable(img_in)

    create_config(config, img_in, img_out, [package_zip], partitions)

    result = run_package_to_image_placer(package_to_image_placer_binary, config=config)

    assert result.returncode == 1
    assert not os.path.exists(img_out)


def test_write_two_big_packages(package_to_image_placer_binary):
    config = "test_data/test_config.json"
    img_in = "test_data/test_img.img.in"
    img_out = "test_data/test_img_out.img"
    package_1 = "test_data/normal_package_1"
    package_2 = "test_data/normal_package_2"

    partitions = [1]

    create_test_package(package_1, "10MB")
    create_test_package(package_2, "8MB")
    create_image(img_in, "17MB", 1)
    make_image_mountable(img_in)

    create_config(config, img_in, img_out, [package_1 + ".zip", package_2 + ".zip"], partitions)

    result = run_package_to_image_placer(package_to_image_placer_binary, config=config)

    assert result.returncode == 1
    assert not os.path.exists(img_out)


def test_write_one_package_to_small_partition(package_to_image_placer_binary):
    """TODO"""
    config = "test_data/test_config.json"
    img_in = "test_data/test_img.img.in"
    img_out = "test_data/test_img_out.img"
    package = "test_data/normal_package"
    package_zip = package + ".zip"
    partitions = [2]

    create_test_package(package, "10MB")
    create_image(img_in, "20MB", 2)
    make_image_mountable(img_in)

    create_config(config, img_in, img_out, [package_zip], partitions)

    result = run_package_to_image_placer(package_to_image_placer_binary, config=config)

    assert result.returncode == 1
    assert not os.path.exists(img_out)


def test_write_multiple_packages_to_partition(package_to_image_placer_binary):
    """TODO"""
    config = "test_data/test_config.json"
    img_in = "test_data/test_img.img.in"
    img_out = "test_data/test_img_out.img"

    packages = [f"test_data/normal_package_{i}" for i in range(1, 21)]
    partitions = [5]

    for packet in packages:
        create_test_package(packet, "50KB")

    create_image(img_in, "5MB", 5)
    make_image_mountable(img_in)

    create_config(
        config,
        img_in,
        img_out,
        [package + ".zip" for package in packages],
        partitions,
    )

    result = run_package_to_image_placer(package_to_image_placer_binary, config=config)

    assert result.returncode == 1
    assert not os.path.exists(img_out)


def test_double_write_without_override(package_to_image_placer_binary):
    """TODO"""
    config_1 = "test_data/test_config_1.json"
    config_2 = "test_data/test_config_2.json"
    img_in = "test_data/test_img.img.in"
    img_out_1 = "test_data/test_img_out_1.img"
    img_out_2 = "test_data/test_img_out_2.img"
    package = "test_data/normal_package"
    package_zip = package + ".zip"
    partitions = [1]

    create_test_package(package, "10KB")
    create_image(img_in, "10MB", 1)
    make_image_mountable(img_in)

    create_config(config_1, img_in, img_out_1, [package_zip], partitions)
    create_config(config_2, img_out_1, img_out_2, [package_zip], partitions)

    result = run_package_to_image_placer(package_to_image_placer_binary, config=config_1)

    assert result.returncode == 0
    assert inspect_image(config_1)

    result = run_package_to_image_placer(package_to_image_placer_binary, config=config_2)

    assert result.returncode == 1
    # check if the source image was not damaged during second run
    assert inspect_image(config_1)


# def test_double_write_without_override_check_if
