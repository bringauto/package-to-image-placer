import subprocess
import pathlib
import os
from test_utils.test_utils import run_package_to_image_placer, create_config, create_test_package


def test_invalid_file_paths(package_to_image_placer_binary):
    """TODO"""
    test_data = "test_data/"

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


def test_empty_image_file(package_to_image_placer_binary):
    """TODO"""
    config = "test_data/test_config.json"
    img_in = "test_data/test_img.img"
    img_out = "test_data/test_img_out.img"
    package = "test_data/normal_package"
    package_zip = package + ".zip"
    partition_numbers = [1]

    subprocess.run(["touch", img_in])

    create_test_package(package, "1KB")

    create_config(config, img_in, img_out, [package_zip], partition_numbers)

    result = run_package_to_image_placer(package_to_image_placer_binary, config=config)

    assert result.returncode == 1
    assert result.stderr != ""
    assert not pathlib.Path(img_out).exists()


def test_invalid_image_file(package_to_image_placer_binary):
    """TODO"""
    # return
    config = "test_data/test_config.json"
    img_in = "test_data/test_img.img"
    img_out = "test_data/test_img_out.img"
    package = "test_data/normal_package"
    package_zip = package + ".zip"
    partition_numbers = [1]

    # Create an empty file to simulate an invalid image
    with open(img_in, "w") as f:
        f.write("I'm just a text file")

    if pathlib.Path(img_out).exists():
        os.remove(img_out)

    create_test_package(package, "2KB")

    create_config(config, img_in, img_out, [package_zip], partition_numbers)

    result = run_package_to_image_placer(package_to_image_placer_binary, config=config)

    assert result.returncode == 1
    assert result.stderr != ""
    assert not pathlib.Path(img_out).exists()
