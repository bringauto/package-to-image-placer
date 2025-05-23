import subprocess
import os
from test_utils.test_utils import (
    run_package_to_image_placer,
    create_test_package,
    create_image,
    make_image_mountable,
    create_config,
    inspect_image,
    create_symlink,
    create_normal_package_config,
    create_configuration_package_config,
)


def test_01_app_shows_help(package_to_image_placer_binary):
    """Tests if the package_to_image_placer will show help message when -h flag is passed"""
    result = subprocess.run([package_to_image_placer_binary, "-h"], capture_output=True, text=True)

    assert result.stderr != ""
    assert result.returncode == 0


def test_02_write_one_package(package_to_image_placer_binary):
    """Test if the package_to_image_placer will write a package to an image"""
    config = "test_data/test_config.json"
    img_in = "test_data/test_img.img.in"
    img_out = "test_data/test_img_out.img"
    package = "test_data/normal_package"
    package_zip = package + ".zip"
    partitions = [1]

    create_test_package(package, "10KB")
    create_image(img_in, "10MB", 1)
    make_image_mountable(img_in)

    create_config(config, img_in, img_out, [create_normal_package_config(package_zip)], partitions)

    result = run_package_to_image_placer(package_to_image_placer_binary, config=config)

    assert result.returncode == 0

    assert inspect_image(config)


def test_03_write_two_packages(package_to_image_placer_binary):
    """Tests if the package_to_image_placer will write two packages to an image"""
    config = "test_data/test_config.json"
    img_in = "test_data/test_img.img.in"
    img_out = "test_data/test_img_out.img"
    package_1 = "test_data/normal_package_1"
    package_2 = "test_data/normal_package_2"
    partitions = [1]

    create_test_package(package_1, "15KB")
    create_test_package(package_2, "5KB")
    create_image(img_in, "10MB", 1)
    make_image_mountable(img_in)

    create_config(
        config,
        img_in,
        img_out,
        [create_normal_package_config(package_1 + ".zip"), create_normal_package_config(package_2 + ".zip")],
        partitions,
    )

    result = run_package_to_image_placer(package_to_image_placer_binary, config=config)

    assert result.returncode == 0

    assert inspect_image(config)


def test_04_write_multiple_packages_to_multiple_partitions(package_to_image_placer_binary):
    """Test if the package_to_image_placer will write multiple packages to multiple partitions"""
    config = "test_data/test_config.json"
    img_in = "test_data/test_img.img.in"
    img_out = "test_data/test_img_out.img"

    package_1 = "test_data/normal_package1"
    package_2 = "test_data/normal_package2"
    package_3 = "test_data/normal_package3"
    package_4 = "test_data/normal_package4"
    package_5 = "test_data/normal_package5"

    partitions = [1, 2, 3, 4]

    create_test_package(package_1, "10KB")
    create_test_package(package_2, "10KB")
    create_test_package(package_3, "10KB")
    create_test_package(package_4, "10KB")
    create_test_package(package_5, "10KB")

    create_image(img_in, "10MB", 5)
    make_image_mountable(img_in)

    create_config(
        config,
        img_in,
        img_out,
        [
            create_normal_package_config(package_1 + ".zip"),
            create_normal_package_config(package_2 + ".zip"),
            create_normal_package_config(package_3 + ".zip"),
            create_normal_package_config(package_4 + ".zip"),
            create_normal_package_config(package_5 + ".zip"),
        ],
        partitions,
    )

    result = run_package_to_image_placer(package_to_image_placer_binary, config=config)

    assert result.returncode == 0

    assert inspect_image(config)


def test_05_pass_package_as_symlink(package_to_image_placer_binary):
    """Test if the package_to_image_placer will write a package to an image when the package is a symlink"""
    config = "test_data/test_config.json"
    img_in = "test_data/test_img.img.in"
    img_out = "test_data/test_img_out.img"
    package = "test_data/normal_package"
    package_zip = package + ".zip"
    package_symlink = package + "_symlink.zip"
    partitions = [1]

    create_test_package(package, "10KB")
    create_symlink(package_zip, package_symlink)
    create_image(img_in, "10MB", 1)
    make_image_mountable(img_in)

    create_config(config, img_in, img_out, [create_normal_package_config(package_symlink)], partitions)

    result = run_package_to_image_placer(package_to_image_placer_binary, config=config)
    assert result.returncode == 0

    assert inspect_image(config)


def test_06_write_one_package_target_already_exists(package_to_image_placer_binary):
    """Test if the package_to_image_placer will write a package to an image when the target image already exists"""
    config = "test_data/test_config.json"
    img_in = "test_data/test_img.img.in"
    img_out = "test_data/test_img_out.img"
    package = "test_data/normal_package"
    package_zip = package + ".zip"
    partitions = [1]

    create_test_package(package, "10KB")
    create_image(img_in, "10MB", 1)
    make_image_mountable(img_in)

    with open(img_out, "w") as f:
        f.write("Test data")

    create_config(config, img_in, img_out, [create_normal_package_config(package_zip)], partitions)

    result = run_package_to_image_placer(package_to_image_placer_binary, config=config)

    assert result.returncode == 0

    assert inspect_image(config)


def test_07_write_one_package_no_clone(package_to_image_placer_binary):
    """Test if the package_to_image_placer will write a package to an image without specifying a source image and with the no-clone flag set"""
    config = "test_data/test_config.json"
    img_in_out = "test_data/test_img.img.in"
    package = "test_data/normal_package"
    package_zip = package + ".zip"
    partitions = [1]

    create_test_package(package, "10KB")
    create_image(img_in_out, "10MB", 1)
    make_image_mountable(img_in_out)

    create_config(
        config,
        target=img_in_out,
        packages=[create_normal_package_config(package_zip)],
        partition_numbers=partitions,
        no_clone=True,
    )

    result = run_package_to_image_placer(package_to_image_placer_binary, config=config)

    assert result.returncode == 0

    assert inspect_image(config)


def test_08_write_one_package_try_different_overwrite_flag(package_to_image_placer_binary):
    """Test if the package_to_image_placer will write a package to an image when the target image already exists. First, it attempts to write without the overwrite flag, expecting the operation to fail. Then, it retries with the overwrite flag set, expecting the operation to succeed."""
    config = "test_data/test_config.json"
    img_in = "test_data/test_img.img.in"
    img_out_1 = "test_data/test_img_out_1.img"
    img_out_2 = "test_data/test_img_out_2.img"
    package = "test_data/normal_package"
    package_zip = package + ".zip"
    partitions = [1]

    create_test_package(package, "10KB")
    create_image(img_in, "10MB", 1)
    make_image_mountable(img_in)

    create_config(config, img_in, img_out_1, [create_normal_package_config(package_zip)], partitions)

    # write package to the image for the first time
    result = run_package_to_image_placer(package_to_image_placer_binary, config=config)
    assert result.returncode == 0
    assert inspect_image(config)

    # write package to the image for the second time (it should fail because override is not set)
    create_config(config, img_out_1, img_out_2, [create_normal_package_config(package_zip)], partitions)
    result = run_package_to_image_placer(package_to_image_placer_binary, config=config)
    assert result.returncode == 1
    assert not os.path.exists(img_out_2)

    # write package to the image for the second time (it should pass because override is set)
    create_config(
        config,
        img_out_1,
        img_out_2,
        [
            create_normal_package_config(
                package_zip, overwrite_file=["/normal_package/test_file", "/normal_package/symlinks/symlink"]
            )
        ],
        partitions,
    )
    result = run_package_to_image_placer(package_to_image_placer_binary, config=config)
    assert result.returncode == 0
    assert inspect_image(config)


def test_09_write_one_package_overwrite_config_no_clone_value_to_true(package_to_image_placer_binary):
    """Test if the package_to_image_placer will write a package to an image without specifying a source image and with the no-clone flag set to True in the arguments, overshadowing the value set in the config file"""
    config = "test_data/test_config.json"
    img_in_out = "test_data/test_img_out.img"
    package = "test_data/normal_package"
    package_zip = package + ".zip"
    partitions = [1]

    create_test_package(package, "10KB")
    create_image(img_in_out, "10MB", 1)
    make_image_mountable(img_in_out)

    create_config(
        config,
        target=img_in_out,
        packages=[create_normal_package_config(package_zip)],
        partition_numbers=partitions,
        no_clone=False,
    )

    result = run_package_to_image_placer(package_to_image_placer_binary, config=config, no_clone=True)
    assert result.returncode == 0
    assert inspect_image(config)


def test_10_write_one_package_overwrite_config_no_clone_value_to_false(package_to_image_placer_binary):
    """Test if the package_to_image_placer will fail when the output image already exists and the no-clone flag is set to False in the arguments, overshadowing the value set in the config file"""
    config = "test_data/test_config.json"
    img_in_out = "test_data/test_img_out.img"
    package = "test_data/normal_package"
    package_zip = package + ".zip"
    partitions = [1]

    create_test_package(package, "10KB")
    create_image(img_in_out, "10MB", 1)
    make_image_mountable(img_in_out)

    create_config(
        config,
        target=img_in_out,
        packages=[create_normal_package_config(package_zip)],
        partition_numbers=partitions,
        no_clone=True,
    )

    result = run_package_to_image_placer(package_to_image_placer_binary, config=config, no_clone=False)
    assert result.returncode == 1
    assert not inspect_image(config)


def test_11_double_write_without_override(package_to_image_placer_binary):
    """Tests if the package_to_image_placer will fail when tha package is already written to the image and the overwrite flag is not set"""
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

    create_config(config_1, img_in, img_out_1, [create_normal_package_config(package_zip)], partitions)
    create_config(config_2, img_out_1, img_out_2, [create_normal_package_config(package_zip)], partitions)

    result = run_package_to_image_placer(package_to_image_placer_binary, config=config_1)

    assert result.returncode == 0
    assert inspect_image(config_1)

    # create file that should be overwritten
    with open(img_out_2, "wb") as f:
        f.write(b"Test data")
    assert os.path.exists(img_out_2)

    result = run_package_to_image_placer(package_to_image_placer_binary, config=config_2)

    assert result.returncode == 1
    # Check if the image was overwritten and removed
    assert not os.path.exists(img_out_2)

    # check if the source image was not damaged during second run
    assert inspect_image(config_1)


def test_12_write_to_custom_target_directory(package_to_image_placer_binary):
    """Test if the package_to_image_placer will write a package to an image in a custom target directory"""
    config = "test_data/test_config.json"
    img_in = "test_data/test_img.img.in"
    img_out = "test_data/test_img_out.img"
    package = "test_data/normal_package"
    package_zip = package + ".zip"
    partitions = [1]

    create_test_package(package, "10KB")
    create_image(img_in, "10MB", 1)
    make_image_mountable(img_in)

    target_directory = "custom_target_directory"
    create_config(
        config,
        img_in,
        img_out,
        [create_normal_package_config(package_zip, target_directory=target_directory)],
        partitions,
    )

    result = run_package_to_image_placer(package_to_image_placer_binary, config=config)

    assert result.returncode == 0
    assert inspect_image(config)


def test_13_write_only_configuration_package(package_to_image_placer_binary):
    """Test if the package_to_image_placer will write a package to an image when the package is a configuration package"""
    config = "test_data/test_config.json"
    img_in = "test_data/test_img.img.in"
    img_out = "test_data/test_img_out.img"
    conf_package = "test_data/configuration_package"
    conf_package_zip = conf_package + ".zip"
    partitions = [1]

    create_test_package(conf_package, "10KB")
    create_image(img_in, "10MB", 1)
    make_image_mountable(img_in)
    create_config(
        config,
        img_in,
        img_out,
        [],
        partitions,
        [create_configuration_package_config(conf_package_zip)],
    )

    result = run_package_to_image_placer(package_to_image_placer_binary, config=config)
    assert result.returncode == 0
    assert inspect_image(config)


def test_14_write_package_and_configuration_package(package_to_image_placer_binary):
    """Test if the package_to_image_placer will write a package and configuration package to an image"""
    config = "test_data/test_config.json"
    img_in = "test_data/test_img.img.in"
    img_out = "test_data/test_img_out.img"
    normal_package = "test_data/normal_package"
    normal_package_zip = normal_package + ".zip"
    conf_package = "test_data/configuration_package"
    conf_package_zip = conf_package + ".zip"
    partitions = [1]

    create_test_package(normal_package, "10KB")
    create_test_package(conf_package, "10KB")
    create_image(img_in, "10MB", 1)
    make_image_mountable(img_in)
    create_config(
        config,
        img_in,
        img_out,
        [create_normal_package_config(normal_package_zip)],
        partitions,
        [create_configuration_package_config(conf_package_zip)],
    )

    result = run_package_to_image_placer(package_to_image_placer_binary, config=config)
    assert result.returncode == 0
    assert inspect_image(config)


def test_15_write_multiple_different_packages(package_to_image_placer_binary):
    """Write five packages of both types to an image"""
    package_n = 5
    config = "test_data/test_config.json"
    img_in = "test_data/test_img.img.in"
    img_out = "test_data/test_img_out.img"
    normal_packages = [f"test_data/normal_package_{i}" for i in range(1, package_n + 1)]
    normal_package_zips = [f"{pkg}.zip" for pkg in normal_packages]
    conf_packages = [f"test_data/configuration_package_{i}" for i in range(1, package_n + 1)]
    conf_package_zips = [f"{pkg}.zip" for pkg in conf_packages]
    partitions = [1, 2]

    for normal_package in normal_packages:
        create_test_package(normal_package, "10KB")

    for conf_package in conf_packages:
        create_test_package(conf_package, "10KB")

    create_image(img_in, "10MB", 2)
    make_image_mountable(img_in)
    create_config(
        config,
        img_in,
        img_out,
        [create_normal_package_config(normal_package_zip) for normal_package_zip in normal_package_zips],
        partitions,
        [create_configuration_package_config(conf_package_zip) for conf_package_zip in conf_package_zips],
    )

    result = run_package_to_image_placer(package_to_image_placer_binary, config=config)
    assert result.returncode == 0
    assert inspect_image(config)
