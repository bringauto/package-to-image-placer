import subprocess
import pathlib
import os
from test_utils.test_utils import (
    run_package_to_image_placer,
    create_config,
    create_normal_package_config,
    create_test_package,
    create_image,
    make_image_mountable,
    inspect_image,
    create_configuration_package_config,
)


def test_01_invalid_file_paths(package_to_image_placer_binary):
    """Tests if the package_to_image_placer will fail when the paths are invalid"""

    r1 = run_package_to_image_placer(package_to_image_placer_binary, config="non_existent_file.json")
    assert r1.stderr != ""
    assert r1.returncode == 1

    r2 = run_package_to_image_placer(package_to_image_placer_binary, target="non_existent_file.img", no_clone=True)
    assert r2.stderr != ""
    assert r2.returncode == 1

    r3 = run_package_to_image_placer(
        package_to_image_placer_binary, source="non_existent_file.img", target="non_existent_file_2.img"
    )

    assert r3.stderr != ""
    assert r3.returncode == 1


def test_02_empty_image_file(package_to_image_placer_binary):
    """Tests if the package_to_image_placer will fail when the image file is empty"""
    config = "test_data/test_config.json"
    img_in = "test_data/test_img.img"
    img_out = "test_data/test_img_out.img"
    package = "test_data/normal_package"
    package_zip = package + ".zip"
    partition_numbers = [1]

    # create an empty file to simulate an empty image
    subprocess.run(["touch", img_in])

    create_test_package(package, "1KB")

    create_config(config, img_in, img_out, [create_normal_package_config(package_zip)], partition_numbers)
    result = run_package_to_image_placer(package_to_image_placer_binary, config=config)

    assert result.returncode == 1
    assert result.stderr != ""
    assert not pathlib.Path(img_out).exists()


def test_03_invalid_image_file(package_to_image_placer_binary):
    """Tests if the package_to_image_placer will fail when the image file is invalid"""
    config = "test_data/test_config.json"
    img_in = "test_data/test_img.img"
    img_out = "test_data/test_img_out.img"
    package = "test_data/normal_package"
    package_zip = package + ".zip"
    partition_numbers = [1]

    # Create a text file to simulate an invalid image
    with open(img_in, "w") as f:
        f.write("I'm just a text file")

    create_test_package(package, "2KB")

    create_config(config, img_in, img_out, [create_normal_package_config(package_zip)], partition_numbers)

    result = run_package_to_image_placer(package_to_image_placer_binary, config=config)

    assert result.returncode == 1
    assert result.stderr != ""
    assert not pathlib.Path(img_out).exists()


def test_04_without_specified_partitions(package_to_image_placer_binary):
    """Tests if the package_to_image_placer will fail when the partitions are not specified"""
    config = "test_data/test_config.json"
    img_in = "test_data/test_img.img"
    img_out = "test_data/test_img_out.img"
    package = "test_data/normal_package"
    package_zip = package + ".zip"
    partition_numbers = []

    create_test_package(package, "10KB")
    create_image(img_in, "10MB", 1)
    make_image_mountable(img_in)

    create_config(config, img_in, img_out, [create_normal_package_config(package_zip)], partition_numbers)

    result = run_package_to_image_placer(package_to_image_placer_binary, config=config)

    assert result.returncode == 1
    assert result.stderr != ""
    assert not pathlib.Path(img_out).exists()


def test_05_without_specified_package(package_to_image_placer_binary):
    """Tests if the package_to_image_placer will fail when the package is not specified"""
    config = "test_data/test_config.json"
    img_in = "test_data/test_img.img"
    img_out = "test_data/test_img_out.img"
    partition_numbers = [1]

    create_image(img_in, "10MB", 1)
    make_image_mountable(img_in)

    create_config(config, img_in, img_out, [], partition_numbers)

    result = run_package_to_image_placer(package_to_image_placer_binary, config=config)

    assert result.returncode == 1
    assert result.stderr != ""
    assert not pathlib.Path(img_out).exists()


def test_06_invalid_config_format(package_to_image_placer_binary):
    """Tests if the package_to_image_placer will fail when the config file is missing a field"""
    config = "test_data/test_config.json"
    img_in = "test_data/test_img.img.in"
    img_out = "test_data/test_img_out.img"
    package = "test_data/normal_package"
    package_zip = package + ".zip"
    partitions = [1]

    create_test_package(package, "10KB")
    create_image(img_in, "10MB", 1)
    make_image_mountable(img_in)

    def remove_from_config_and_run(remove_from_config: list[str], expected_result: bool):
        if pathlib.Path(img_out).exists():
            os.remove(img_out)

        create_config(
            config,
            img_in,
            img_out,
            [create_normal_package_config(package_zip)],
            partitions,
            remove_from_config=remove_from_config,
        )
        result = run_package_to_image_placer(package_to_image_placer_binary, config=config)

        if expected_result:
            assert result.returncode == 0
            assert inspect_image(config)
        else:
            assert result.returncode == 1
            assert not pathlib.Path(img_out).exists()

    remove_from_config_and_run(["source"], False)
    remove_from_config_and_run(["target"], False)
    remove_from_config_and_run(["packages"], False)
    remove_from_config_and_run(["partition-numbers"], False)
    remove_from_config_and_run(["no-clone"], True)
    remove_from_config_and_run(["log-path"], True)

    create_config(
        config,
        "Invalid path",
        img_in,
        [create_normal_package_config(package_zip)],
        partitions,
        no_clone=True,
        remove_from_config=["source"],
    )
    result = run_package_to_image_placer(package_to_image_placer_binary, config=config)
    assert result.returncode == 0
    assert inspect_image(config)


def test_07_write_package_and_nonexisting_configuration_package(package_to_image_placer_binary):
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
    assert result.returncode == 1
    assert not os.path.exists(img_out)


def test_08_write_nonexisting_package_and_configuration_package(package_to_image_placer_binary):
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
    assert result.returncode == 1
    assert not os.path.exists(img_out)
