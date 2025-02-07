import subprocess
from test_utils.test_utils import run_package_to_image_placer


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
