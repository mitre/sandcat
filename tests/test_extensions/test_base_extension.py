"""Tests for app/utility/base_extension.py — Extension base class."""
import os
import shutil
import tempfile
from unittest.mock import AsyncMock, MagicMock, patch

import pytest

from app.utility.base_extension import Extension, ConfigFileException


# Concrete subclass for testing the ABC
class ConcreteExtension(Extension):
    def __init__(self, files=None, dependencies=None, file_hooks=None):
        super().__init__(files or [], dependencies=dependencies, file_hooks=file_hooks)


# ========================================================================
# __init__
# ========================================================================

class TestExtensionInit:
    def test_files_stored(self):
        ext = ConcreteExtension(files=[('a.go', 'pkg')])
        assert ext.files == [('a.go', 'pkg')]

    def test_default_dependencies(self):
        ext = ConcreteExtension()
        assert ext.dependencies == []

    def test_custom_dependencies(self):
        ext = ConcreteExtension(dependencies=['github.com/foo/bar'])
        assert ext.dependencies == ['github.com/foo/bar']

    def test_default_file_hooks(self):
        ext = ConcreteExtension()
        assert ext.file_hooks == {}

    def test_custom_file_hooks(self):
        hook = AsyncMock()
        ext = ConcreteExtension(file_hooks={'a.go': hook})
        assert ext.file_hooks == {'a.go': hook}


# ========================================================================
# check_go_dependencies
# ========================================================================

class TestCheckGoDependencies:
    def test_no_dependencies_returns_true(self):
        ext = ConcreteExtension()
        assert ext.check_go_dependencies('/fake') is True

    @patch('subprocess.run')
    def test_dependency_met(self, mock_run):
        mock_run.return_value = MagicMock(stdout=b'github.com/foo/bar\n')
        ext = ConcreteExtension(dependencies=['github.com/foo/bar'])
        assert ext.check_go_dependencies('/gocat') is True

    @patch('subprocess.run')
    def test_dependency_not_met(self, mock_run):
        mock_run.return_value = MagicMock(stdout=b'')
        ext = ConcreteExtension(dependencies=['github.com/foo/bar'])
        assert ext.check_go_dependencies('/gocat') is False

    @patch('subprocess.run')
    def test_multiple_deps_all_met(self, mock_run):
        mock_run.side_effect = [
            MagicMock(stdout=b'dep1\n'),
            MagicMock(stdout=b'dep2\n'),
        ]
        ext = ConcreteExtension(dependencies=['dep1', 'dep2'])
        assert ext.check_go_dependencies('/gocat') is True

    @patch('subprocess.run')
    def test_multiple_deps_one_fails(self, mock_run):
        mock_run.side_effect = [
            MagicMock(stdout=b'dep1\n'),
            MagicMock(stdout=b'wrong\n'),
        ]
        ext = ConcreteExtension(dependencies=['dep1', 'dep2'])
        assert ext.check_go_dependencies('/gocat') is False

    @patch('subprocess.run')
    def test_uses_correct_cwd(self, mock_run):
        import subprocess
        mock_run.return_value = MagicMock(stdout=b'dep\n')
        ext = ConcreteExtension(dependencies=['dep'])
        ext.check_go_dependencies('/my/gocat/dir')
        mock_run.assert_called_with(
            'go list "dep"', shell=True, cwd='/my/gocat/dir',
            stdout=subprocess.PIPE, stderr=subprocess.DEVNULL
        )


# ========================================================================
# copy_module_files
# ========================================================================

class TestCopyModuleFiles:
    @pytest.mark.asyncio
    async def test_copy_single_file(self, tmp_dir):
        base = tmp_dir
        src = os.path.join(base, 'gocat-extensions', 'pkg')
        dest = os.path.join(base, 'gocat', 'pkg')
        os.makedirs(src)
        with open(os.path.join(src, 'test.go'), 'w') as f:
            f.write('package main')

        ext = ConcreteExtension(files=[('test.go', 'pkg')])
        result = await ext.copy_module_files(base)
        assert result is True
        assert os.path.exists(os.path.join(dest, 'test.go'))
        with open(os.path.join(dest, 'test.go')) as f:
            assert f.read() == 'package main'

    @pytest.mark.asyncio
    async def test_copy_wildcard(self, tmp_dir):
        base = tmp_dir
        src = os.path.join(base, 'gocat-extensions', 'pkg')
        dest = os.path.join(base, 'gocat', 'pkg')
        os.makedirs(src)
        os.makedirs(dest)
        for name in ['a.go', 'b.go']:
            with open(os.path.join(src, name), 'w') as f:
                f.write(f'// {name}')

        ext = ConcreteExtension(files=[('*', 'pkg')])
        await ext.copy_module_files(base)
        assert os.path.exists(os.path.join(dest, 'a.go'))
        assert os.path.exists(os.path.join(dest, 'b.go'))

    @pytest.mark.asyncio
    async def test_copy_creates_dest_dir(self, tmp_dir):
        base = tmp_dir
        src = os.path.join(base, 'gocat-extensions', 'newpkg')
        os.makedirs(src)
        with open(os.path.join(src, 'test.go'), 'w') as f:
            f.write('pkg')

        ext = ConcreteExtension(files=[('test.go', 'newpkg')])
        await ext.copy_module_files(base)
        assert os.path.isdir(os.path.join(base, 'gocat', 'newpkg'))

    @pytest.mark.asyncio
    async def test_file_hook_applied(self, tmp_dir):
        base = tmp_dir
        src = os.path.join(base, 'gocat-extensions', 'pkg')
        os.makedirs(src)
        with open(os.path.join(src, 'test.go'), 'w') as f:
            f.write('PLACEHOLDER')

        async def hook(data):
            return data.replace('PLACEHOLDER', 'REPLACED')

        ext = ConcreteExtension(files=[('test.go', 'pkg')], file_hooks={'test.go': hook})
        await ext.copy_module_files(base)
        with open(os.path.join(base, 'gocat', 'pkg', 'test.go')) as f:
            assert f.read() == 'REPLACED'

    @pytest.mark.asyncio
    async def test_multiple_files(self, tmp_dir):
        base = tmp_dir
        for pkg in ['pkg1', 'pkg2']:
            src = os.path.join(base, 'gocat-extensions', pkg)
            os.makedirs(src)
            with open(os.path.join(src, 'file.go'), 'w') as f:
                f.write(pkg)

        ext = ConcreteExtension(files=[('file.go', 'pkg1'), ('file.go', 'pkg2')])
        await ext.copy_module_files(base)
        for pkg in ['pkg1', 'pkg2']:
            assert os.path.exists(os.path.join(base, 'gocat', pkg, 'file.go'))


# ========================================================================
# remove_module_files
# ========================================================================

class TestRemoveModuleFiles:
    def test_remove_single_file(self, tmp_dir):
        pkg_dir = os.path.join(tmp_dir, 'gocat', 'pkg')
        os.makedirs(pkg_dir)
        fpath = os.path.join(pkg_dir, 'test.go')
        with open(fpath, 'w') as f:
            f.write('data')

        ext = ConcreteExtension(files=[('test.go', 'pkg')])
        ext.remove_module_files(tmp_dir)
        assert not os.path.exists(fpath)

    def test_remove_nonexistent_file_no_error(self, tmp_dir):
        pkg_dir = os.path.join(tmp_dir, 'gocat', 'pkg')
        os.makedirs(pkg_dir)
        ext = ConcreteExtension(files=[('missing.go', 'pkg')])
        ext.remove_module_files(tmp_dir)  # should not raise

    def test_remove_wildcard_preserves_load_go(self, tmp_dir):
        pkg_dir = os.path.join(tmp_dir, 'gocat', 'pkg')
        os.makedirs(pkg_dir)
        for name in ['load.go', 'ext.go', 'other.go']:
            with open(os.path.join(pkg_dir, name), 'w') as f:
                f.write('data')

        ext = ConcreteExtension(files=[('*', 'pkg')])
        ext.remove_module_files(tmp_dir)
        assert os.path.exists(os.path.join(pkg_dir, 'load.go'))
        assert not os.path.exists(os.path.join(pkg_dir, 'ext.go'))
        assert not os.path.exists(os.path.join(pkg_dir, 'other.go'))


# ========================================================================
# _unstage_folder
# ========================================================================

class TestUnstageFolder:
    def test_preserves_load_go(self, tmp_dir):
        for name in ['load.go', 'a.go', 'b.go']:
            with open(os.path.join(tmp_dir, name), 'w') as f:
                f.write('x')
        Extension._unstage_folder(tmp_dir)
        assert os.path.exists(os.path.join(tmp_dir, 'load.go'))
        assert not os.path.exists(os.path.join(tmp_dir, 'a.go'))

    def test_ignores_directories(self, tmp_dir):
        subdir = os.path.join(tmp_dir, 'subdir')
        os.makedirs(subdir)
        Extension._unstage_folder(tmp_dir)
        assert os.path.isdir(subdir)


# ========================================================================
# install_dependencies
# ========================================================================

class TestInstallDependencies:
    def test_default_returns_false(self):
        ext = ConcreteExtension()
        assert ext.install_dependencies() is False


# ========================================================================
# ConfigFileException
# ========================================================================

class TestConfigFileException:
    def test_is_exception(self):
        assert issubclass(ConfigFileException, Exception)

    def test_message(self):
        exc = ConfigFileException('test message')
        assert str(exc) == 'test message'
