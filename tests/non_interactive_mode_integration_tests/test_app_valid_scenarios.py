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
    crete_symlink,
)


def test_app_shows_help(package_to_image_placer_binary):
    """Tests if the package_to_image_placer will show help message when -h flag is passed"""
    result = subprocess.run([package_to_image_placer_binary, "-h"], capture_output=True, text=True)

    assert result.stderr != ""
    assert result.returncode == 0


def test_write_one_package(package_to_image_placer_binary):
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

    create_config(config, img_in, img_out, [package_zip], partitions)

    result = run_package_to_image_placer(package_to_image_placer_binary, config=config)

    assert result.returncode == 0

    assert inspect_image(config)


def test_write_two_packages(package_to_image_placer_binary):
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

    create_config(config, img_in, img_out, [package_1 + ".zip", package_2 + ".zip"], partitions)

    result = run_package_to_image_placer(package_to_image_placer_binary, config=config)

    assert result.returncode == 0

    assert inspect_image(config)


def test_write_multiple_packages_to_multiple_partitions(package_to_image_placer_binary):
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
        [package_1 + ".zip", package_2 + ".zip", package_3 + ".zip", package_4 + ".zip", package_5 + ".zip"],
        partitions,
    )

    result = run_package_to_image_placer(package_to_image_placer_binary, config=config)

    assert result.returncode == 0

    assert inspect_image(config)


def test_pass_package_as_symlink(package_to_image_placer_binary):
    """Test if the package_to_image_placer will write a package to an image when the package is a symlink"""
    config = "test_data/test_config.json"
    img_in = "test_data/test_img.img.in"
    img_out = "test_data/test_img_out.img"
    package = "test_data/normal_package"
    package_zip = package + ".zip"
    package_symlink = package + "_symlink.zip"
    partitions = [1]

    create_test_package(package, "10KB")
    crete_symlink(package_zip, package_symlink)
    create_image(img_in, "10MB", 1)
    make_image_mountable(img_in)

    create_config(config, img_in, img_out, [package_symlink], partitions)

    result = run_package_to_image_placer(package_to_image_placer_binary, config=config)
    assert result.returncode == 0

    assert inspect_image(config)


def test_write_one_package_target_override(package_to_image_placer_binary):
    """Test if the package_to_image_placer will write a package to an image when the target image already exists and the override flag is set, ensuring it overwrites the target image"""
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

    create_config(config, img_in, img_out, [package_zip], partitions, overwrite=True)

    result = run_package_to_image_placer(package_to_image_placer_binary, config=config)

    assert result.returncode == 0

    assert inspect_image(config)


def test_write_one_package_no_clone(package_to_image_placer_binary):
    """Test if the package_to_image_placer will write a package to an image without specifying a source image and with the no-clone flag set"""
    config = "test_data/test_config.json"
    img_in_out = "test_data/test_img.img.in"
    package = "test_data/normal_package"
    package_zip = package + ".zip"
    partitions = [1]

    create_test_package(package, "10KB")
    create_image(img_in_out, "10MB", 1)
    make_image_mountable(img_in_out)

    create_config(config, target=img_in_out, packages=[package_zip], partition_numbers=partitions, no_clone=True)

    result = run_package_to_image_placer(package_to_image_placer_binary, config=config)

    assert result.returncode == 0

    assert inspect_image(config)


def test_write_one_package_try_different_overwrite_flag(package_to_image_placer_binary):
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

    create_config(config, img_in, img_out_1, [package_zip], partitions)

    # write package to the image for the first time
    result = run_package_to_image_placer(package_to_image_placer_binary, config=config)
    assert result.returncode == 0
    assert inspect_image(config)

    # write package to the image for the second time (it should fail because override is not set)
    create_config(config, img_out_1, img_out_2, [package_zip], partitions)
    result = run_package_to_image_placer(package_to_image_placer_binary, config=config, overwrite=False)
    assert result.returncode == 1
    assert not os.path.exists(img_out_2)

    # write package to the image for the second time (it should pass because override is set)
    create_config(config, img_out_1, img_out_2, [package_zip], partitions)
    result = run_package_to_image_placer(package_to_image_placer_binary, config=config, overwrite=True)
    assert result.returncode == 0
    assert inspect_image(config)


def test_write_one_package_overwrite_config_overwrite_value_to_false(package_to_image_placer_binary):
    """Test if the package_to_image_placer will write a package to an image when the target image already exists. The overwrite flag is set to true in the config file but is overshadowed by the overwrite argument set to false, ensuring the operation fails."""
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

    create_config(config, img_in, img_out_1, [package_zip], partitions)

    # write package to the image for the first time
    result = run_package_to_image_placer(package_to_image_placer_binary, config=config)
    assert result.returncode == 0
    assert inspect_image(config)

    # write package to the image for the second time (it should fail because override is set to false from arguments and it should overshadow the value from the config file)
    create_config(config, img_out_1, img_out_2, [package_zip], partitions, overwrite=True)
    result = run_package_to_image_placer(package_to_image_placer_binary, config=config, overwrite=False)
    assert result.returncode == 1
    assert not os.path.exists(img_out_2)


def test_write_one_package_overwrite_config_no_clone_value_to_true(package_to_image_placer_binary):
    """Test if the package_to_image_placer will write a package to an image without specifying a source image and with the no-clone flag set to True in the arguments, overshadowing the value set in the config file"""
    config = "test_data/test_config.json"
    img_in_out = "test_data/test_img_out.img"
    package = "test_data/normal_package"
    package_zip = package + ".zip"
    partitions = [1]

    create_test_package(package, "10KB")
    create_image(img_in_out, "10MB", 1)
    make_image_mountable(img_in_out)

    create_config(config, target=img_in_out, packages=[package_zip], partition_numbers=partitions, no_clone=False)

    result = run_package_to_image_placer(package_to_image_placer_binary, config=config, no_clone=True)
    assert result.returncode == 0
    assert inspect_image(config)


def test_write_one_package_overwrite_config_no_clone_value_to_false(package_to_image_placer_binary):
    """Test if the package_to_image_placer will fail when the output image already exists and the no-clone flag is set to False in the arguments, overshadowing the value set in the config file"""
    config = "test_data/test_config.json"
    img_in_out = "test_data/test_img_out.img"
    package = "test_data/normal_package"
    package_zip = package + ".zip"
    partitions = [1]

    create_test_package(package, "10KB")
    create_image(img_in_out, "10MB", 1)
    make_image_mountable(img_in_out)

    create_config(config, target=img_in_out, packages=[package_zip], partition_numbers=partitions, no_clone=True)

    result = run_package_to_image_placer(package_to_image_placer_binary, config=config, no_clone=False)
    assert result.returncode == 1
    assert not inspect_image(config)


def test_double_write_without_override(package_to_image_placer_binary):
    """Tests if the package_to_image_placer will not write a package to an image when the target image already exists and the overwrite flag is not set"""
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
    create_config(config_2, img_out_1, img_out_2, [package_zip], partitions, overwrite=False)

    result = run_package_to_image_placer(package_to_image_placer_binary, config=config_1)

    assert result.returncode == 0
    assert inspect_image(config_1)

    # crete file that should not be overwritten
    test_text = "Lorem ipsum"
    with open(img_out_2, "wb") as f:
        f.write(test_text.encode())
    assert os.path.exists(img_out_2)

    result = run_package_to_image_placer(package_to_image_placer_binary, config=config_2)

    assert result.returncode == 1

    # check if the file was not overwritten
    assert os.path.exists(img_out_2)
    with open(img_out_2, "rb") as f:
        assert f.read().decode() == test_text

    # check if the source image was not damaged during second run
    assert inspect_image(config_1)
