"""Tests for shared extension."""
import pytest

from plugins.sandcat.app.utility.base_extension import Extension


class TestShared:
    @pytest.fixture
    def shared_ext(self):
        from app.extensions.shared.shared import Shared
        return Shared()

    @pytest.fixture
    def shared_load(self):
        from app.extensions.shared.shared import load
        return load

    def test_load_returns_instance(self, shared_load):
        ext = shared_load()
        from app.extensions.shared.shared import Shared
        assert isinstance(ext, Shared)

    def test_is_extension(self, shared_ext):
        assert isinstance(shared_ext, Extension)

    def test_files(self, shared_ext):
        assert shared_ext.files == [('shared.go', 'shared')]

    def test_no_dependencies(self, shared_ext):
        assert shared_ext.dependencies == []

    def test_file_hooks_registered(self, shared_ext):
        assert 'shared.go' in shared_ext.file_hooks

    def test_additional_exports_starts_empty(self, shared_ext):
        assert shared_ext.additional_exports == []


class TestSharedHookSetAdditionalExports:
    @pytest.fixture
    def shared_ext(self):
        from app.extensions.shared.shared import Shared
        return Shared()

    @pytest.mark.asyncio
    async def test_no_exports_returns_original(self, shared_ext):
        data = 'some code // ADDITIONAL EXPORTS PLACEHOLDER'
        result = await shared_ext.hook_set_additional_exports(data)
        assert result == data

    @pytest.mark.asyncio
    async def test_single_export(self, shared_ext):
        shared_ext.additional_exports = ['MyFunc']
        data = 'code\n// ADDITIONAL EXPORTS PLACEHOLDER\nmore'
        result = await shared_ext.hook_set_additional_exports(data)
        assert '//export MyFunc' in result
        assert 'func MyFunc()' in result
        assert 'VoidFunc()' in result

    @pytest.mark.asyncio
    async def test_multiple_exports(self, shared_ext):
        shared_ext.additional_exports = ['FuncA', 'FuncB']
        data = '// ADDITIONAL EXPORTS PLACEHOLDER'
        result = await shared_ext.hook_set_additional_exports(data)
        assert '//export FuncA' in result
        assert '//export FuncB' in result
        assert 'func FuncA()' in result
        assert 'func FuncB()' in result

    @pytest.mark.asyncio
    async def test_exports_cleared_after_hook(self, shared_ext):
        shared_ext.additional_exports = ['MyFunc']
        data = '// ADDITIONAL EXPORTS PLACEHOLDER'
        await shared_ext.hook_set_additional_exports(data)
        assert shared_ext.additional_exports == []

    @pytest.mark.asyncio
    async def test_sanitized_export_names(self, shared_ext):
        shared_ext.additional_exports = ['bad-func!name']
        data = '// ADDITIONAL EXPORTS PLACEHOLDER'
        result = await shared_ext.hook_set_additional_exports(data)
        assert '//export bad_func_name' in result


class TestSharedSanitizeExportFunc:
    def test_alphanumeric_unchanged(self):
        from app.extensions.shared.shared import Shared
        assert Shared.sanitize_export_func('MyFunc123') == 'MyFunc123'

    def test_underscores_preserved(self):
        from app.extensions.shared.shared import Shared
        assert Shared.sanitize_export_func('my_func') == 'my_func'

    def test_special_chars_replaced(self):
        from app.extensions.shared.shared import Shared
        assert Shared.sanitize_export_func('bad-func!name') == 'bad_func_name'

    def test_spaces_replaced(self):
        from app.extensions.shared.shared import Shared
        assert Shared.sanitize_export_func('a b c') == 'a_b_c'

    def test_empty_string(self):
        from app.extensions.shared.shared import Shared
        assert Shared.sanitize_export_func('') == ''


class TestSharedCopyModuleFiles:
    @pytest.fixture
    def shared_ext(self):
        from app.extensions.shared.shared import Shared
        return Shared()

    @pytest.mark.asyncio
    async def test_headers_with_additional_exports(self, shared_ext, tmp_dir):
        import os
        src = os.path.join(tmp_dir, 'gocat-extensions', 'shared')
        dest = os.path.join(tmp_dir, 'gocat', 'shared')
        os.makedirs(src)
        with open(os.path.join(src, 'shared.go'), 'w') as f:
            f.write('// ADDITIONAL EXPORTS PLACEHOLDER')

        headers = {'additional_exports': 'FuncA,FuncB'}
        await shared_ext.copy_module_files(tmp_dir, headers=headers)

        with open(os.path.join(dest, 'shared.go')) as f:
            content = f.read()
        assert '//export FuncA' in content
        assert '//export FuncB' in content

    @pytest.mark.asyncio
    async def test_headers_without_exports(self, shared_ext, tmp_dir):
        import os
        src = os.path.join(tmp_dir, 'gocat-extensions', 'shared')
        os.makedirs(src)
        with open(os.path.join(src, 'shared.go'), 'w') as f:
            f.write('placeholder code')

        await shared_ext.copy_module_files(tmp_dir, headers={})
        assert shared_ext.additional_exports == []

    @pytest.mark.asyncio
    async def test_no_headers(self, shared_ext, tmp_dir):
        import os
        src = os.path.join(tmp_dir, 'gocat-extensions', 'shared')
        os.makedirs(src)
        with open(os.path.join(src, 'shared.go'), 'w') as f:
            f.write('code')

        await shared_ext.copy_module_files(tmp_dir, headers=None)
        # Should not crash

    @pytest.mark.asyncio
    async def test_empty_export_string_ignored(self, shared_ext, tmp_dir):
        import os
        src = os.path.join(tmp_dir, 'gocat-extensions', 'shared')
        os.makedirs(src)
        with open(os.path.join(src, 'shared.go'), 'w') as f:
            f.write('code')

        headers = {'additional_exports': ''}
        await shared_ext.copy_module_files(tmp_dir, headers=headers)
        assert shared_ext.additional_exports == []
