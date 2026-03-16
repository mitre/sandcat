"""Tests for donut extension."""
import pytest

from plugins.sandcat.app.utility.base_extension import Extension


class TestDonut:
    @pytest.fixture
    def donut_ext(self):
        from app.extensions.donut.donut import Donut
        return Donut()

    @pytest.fixture
    def donut_load(self):
        from app.extensions.donut.donut import load
        return load

    def test_load_returns_instance(self, donut_load):
        ext = donut_load()
        from app.extensions.donut.donut import Donut
        assert isinstance(ext, Donut)

    def test_is_extension(self, donut_ext):
        assert isinstance(donut_ext, Extension)

    def test_files(self, donut_ext):
        expected = [
            ('donut.go', 'execute/donut'),
            ('dll_windows.go', 'execute/donut'),
            ('donut_windows.go', 'execute/donut'),
            ('donut_helper_windows.go', 'execute/donut'),
        ]
        assert donut_ext.files == expected

    def test_no_dependencies(self, donut_ext):
        assert donut_ext.dependencies == []

    def test_no_file_hooks(self, donut_ext):
        assert donut_ext.file_hooks == {}

    def test_all_files_in_same_package(self, donut_ext):
        packages = {p for _, p in donut_ext.files}
        assert packages == {'execute/donut'}

    def test_windows_specific_files(self, donut_ext):
        filenames = [f for f, _ in donut_ext.files]
        windows_files = [f for f in filenames if 'windows' in f]
        assert len(windows_files) == 3
