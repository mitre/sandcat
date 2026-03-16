"""Tests for execute extensions: native, native_aws, shellcode, shells."""
import pytest

from plugins.sandcat.app.utility.base_extension import Extension


# ========================================================================
# Native Executor
# ========================================================================

class TestNativeExecutor:
    @pytest.fixture
    def native_ext(self):
        from app.extensions.execute.native.native import NativeExecutor
        return NativeExecutor()

    @pytest.fixture
    def native_load(self):
        from app.extensions.execute.native.native import load
        return load

    def test_load_returns_instance(self, native_load):
        ext = native_load()
        from app.extensions.execute.native.native import NativeExecutor
        assert isinstance(ext, NativeExecutor)

    def test_is_extension(self, native_ext):
        assert isinstance(native_ext, Extension)

    def test_files(self, native_ext):
        expected = [
            ('native.go', 'execute/native'),
            ('*', 'execute/native/discovery'),
            ('util.go', 'execute/native/util'),
        ]
        assert native_ext.files == expected

    def test_no_dependencies(self, native_ext):
        assert native_ext.dependencies == []

    def test_no_file_hooks(self, native_ext):
        assert native_ext.file_hooks == {}

    def test_includes_wildcard_for_discovery(self, native_ext):
        wildcard_entries = [(f, p) for f, p in native_ext.files if f == '*']
        assert len(wildcard_entries) == 1
        assert wildcard_entries[0][1] == 'execute/native/discovery'


# ========================================================================
# Native AWS Executor
# ========================================================================

class TestNativeAwsExecutor:
    @pytest.fixture
    def aws_ext(self):
        from app.extensions.execute.native.native_aws import NativeAwsExecutor
        return NativeAwsExecutor()

    @pytest.fixture
    def aws_load(self):
        from app.extensions.execute.native.native_aws import load
        return load

    def test_load_returns_instance(self, aws_load):
        ext = aws_load()
        from app.extensions.execute.native.native_aws import NativeAwsExecutor
        assert isinstance(ext, NativeAwsExecutor)

    def test_is_extension(self, aws_ext):
        assert isinstance(aws_ext, Extension)

    def test_files(self, aws_ext):
        expected = [
            ('native.go', 'execute/native'),
            ('*', 'execute/native/aws'),
            ('util.go', 'execute/native/util'),
        ]
        assert aws_ext.files == expected

    def test_dependencies(self, aws_ext):
        assert 'github.com/aws/aws-sdk-go' in aws_ext.dependencies
        assert 'github.com/aws/aws-sdk-go/aws' in aws_ext.dependencies

    def test_no_file_hooks(self, aws_ext):
        assert aws_ext.file_hooks == {}

    def test_includes_wildcard_for_aws(self, aws_ext):
        wildcard_entries = [(f, p) for f, p in aws_ext.files if f == '*']
        assert len(wildcard_entries) == 1
        assert wildcard_entries[0][1] == 'execute/native/aws'


# ========================================================================
# Shellcode
# ========================================================================

class TestShellcode:
    @pytest.fixture
    def shellcode_ext(self):
        from app.extensions.execute.shellcode.shellcode import Shellcode
        return Shellcode()

    @pytest.fixture
    def shellcode_load(self):
        from app.extensions.execute.shellcode.shellcode import load
        return load

    def test_load_returns_instance(self, shellcode_load):
        ext = shellcode_load()
        from app.extensions.execute.shellcode.shellcode import Shellcode
        assert isinstance(ext, Shellcode)

    def test_is_extension(self, shellcode_ext):
        assert isinstance(shellcode_ext, Extension)

    def test_files_wildcard(self, shellcode_ext):
        assert shellcode_ext.files == [('*', 'execute/shellcode')]

    def test_no_dependencies(self, shellcode_ext):
        assert shellcode_ext.dependencies == []

    def test_no_file_hooks(self, shellcode_ext):
        assert shellcode_ext.file_hooks == {}


# ========================================================================
# Shells
# ========================================================================

class TestShells:
    @pytest.fixture
    def shells_ext(self):
        from app.extensions.execute.shells.shells import Shells
        return Shells()

    @pytest.fixture
    def shells_load(self):
        from app.extensions.execute.shells.shells import load
        return load

    def test_load_returns_instance(self, shells_load):
        ext = shells_load()
        from app.extensions.execute.shells.shells import Shells
        assert isinstance(ext, Shells)

    def test_is_extension(self, shells_ext):
        assert isinstance(shells_ext, Extension)

    def test_files(self, shells_ext):
        expected = [
            ('osascript.go', 'execute/shells'),
            ('powershell_core.go', 'execute/shells'),
            ('python.go', 'execute/shells'),
        ]
        assert shells_ext.files == expected

    def test_no_dependencies(self, shells_ext):
        assert shells_ext.dependencies == []

    def test_no_file_hooks(self, shells_ext):
        assert shells_ext.file_hooks == {}

    def test_all_shell_types_present(self, shells_ext):
        filenames = [f for f, _ in shells_ext.files]
        assert 'osascript.go' in filenames
        assert 'powershell_core.go' in filenames
        assert 'python.go' in filenames
