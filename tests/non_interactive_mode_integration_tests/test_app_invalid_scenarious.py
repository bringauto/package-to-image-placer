import os
from test_utils.test_utils import (
    run_package_to_image_placer,
    create_test_package,
    create_image,
    make_image_mountable,
    create_config,
    create_normal_package_config,
    create_configuration_package_config,
)


def test_01_write_one_big_package(package_to_image_placer_binary):
    """Tests if the package_to_image_placer will fail when the package is bigger than the image"""
    config = "test_data/test_config.json"
    img_in = "test_data/test_img.img.in"
    img_out = "test_data/test_img_out.img"
    package = "test_data/normal_package"
    package_zip = package + ".zip"
    partitions = [1]

    create_test_package(package, "11MB")
    create_image(img_in, "10MB", 1)
    make_image_mountable(img_in)

    create_config(config, img_in, img_out, [create_normal_package_config(package_zip)], partitions)

    result = run_package_to_image_placer(package_to_image_placer_binary, config=config)

    assert result.returncode == 1
    assert not os.path.exists(img_out)


def test_02_write_two_big_packages(package_to_image_placer_binary):
    """Tests if the package_to_image_placer will fail when the sum of packages is bigger than the image"""
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

    create_config(
        config,
        img_in,
        img_out,
        [create_normal_package_config(package_1 + ".zip"), create_normal_package_config(package_2 + ".zip")],
        partitions,
    )

    result = run_package_to_image_placer(package_to_image_placer_binary, config=config)

    assert result.returncode == 1
    assert not os.path.exists(img_out)


def test_03_write_one_package_to_small_partition(package_to_image_placer_binary):
    """Tests if the package_to_image_placer will fail when the package is bigger than the partition"""
    config = "test_data/test_config.json"
    img_in = "test_data/test_img.img.in"
    img_out = "test_data/test_img_out.img"
    package = "test_data/normal_package"
    package_zip = package + ".zip"
    partitions = [2]

    create_test_package(package, "10MB")
    create_image(img_in, "20MB", 2)
    make_image_mountable(img_in)

    create_config(config, img_in, img_out, [create_normal_package_config(package_zip)], partitions)

    result = run_package_to_image_placer(package_to_image_placer_binary, config=config)

    assert result.returncode == 1
    assert not os.path.exists(img_out)


def test_04_write_multiple_packages_to_partition(package_to_image_placer_binary):
    """Tests if the package_to_image_placer will fail when the sum of packages is bigger than the partition"""
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
        config, img_in, img_out, [create_normal_package_config(package + ".zip") for package in packages], partitions
    )

    result = run_package_to_image_placer(package_to_image_placer_binary, config=config)

    assert result.returncode == 1
    assert not os.path.exists(img_out)


def test_05_write_package_with_invalid_overwrite(package_to_image_placer_binary):
    """Test if the package_to_image_placer will fail when attempting to write a package with an invalid overwrite file"""
    config = "test_data/test_config.json"
    img_in = "test_data/test_img.img.in"
    img_out = "test_data/test_img_out.img"
    package = "test_data/normal_package"
    package_zip = package + ".zip"
    partitions = [1]

    create_test_package(package, "10KB")
    create_image(img_in, "10MB", 1)
    make_image_mountable(img_in)

    create_config(
        config,
        img_in,
        img_out,
        [create_normal_package_config(package_zip, overwrite_file=["/nonexisting_file"])],
        partitions,
    )

    result = run_package_to_image_placer(package_to_image_placer_binary, config=config)

    assert result.returncode == 1

    assert not os.path.exists(img_out)


def test_06_write_package_with_invalid_overwrite_in_config_package(package_to_image_placer_binary):
    """Test if the package_to_image_placer will fail when attempting to write a config package with an invalid overwrite file"""
    config = "test_data/test_config.json"
    img_in = "test_data/test_img.img.in"
    img_out = "test_data/test_img_out.img"
    package = "test_data/normal_package"
    package_zip = package + ".zip"
    partitions = [1]

    create_test_package(package, "10KB")
    create_image(img_in, "10MB", 1)
    make_image_mountable(img_in)

    create_config(
        config,
        img_in,
        img_out,
        [],
        partitions,
        configuration_packages=[create_configuration_package_config(package_zip, overwrite_file=["/nonexisting_file"])],
    )

    result = run_package_to_image_placer(package_to_image_placer_binary, config=config)

    assert result.returncode == 1

    assert not os.path.exists(img_out)
