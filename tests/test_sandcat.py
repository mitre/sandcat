import ast
import glob
import os

import pytest

yaml = pytest.importorskip("yaml")


PLUGIN_DIR = os.path.abspath(os.path.join(os.path.dirname(__file__), '..'))
ABILITIES_DIR = os.path.join(PLUGIN_DIR, 'data', 'abilities')
PAYLOADS_DIR = os.path.join(PLUGIN_DIR, 'payloads')
APP_DIR = os.path.join(PLUGIN_DIR, 'app')

REQUIRED_ABILITY_FIELDS = {'id', 'name', 'tactic', 'technique'}


class TestHookModule:
    """Tests that hook.py loads and registers routes."""

    def test_hook_module_loads(self):
        hook_path = os.path.join(PLUGIN_DIR, 'hook.py')
        assert os.path.isfile(hook_path), 'hook.py not found'
        with open(hook_path, encoding='utf-8') as f:
            source = f.read()
        tree = ast.parse(source)
        top_level_names = [
            node.name
            for node in ast.walk(tree)
            if isinstance(node, (ast.FunctionDef, ast.AsyncFunctionDef))
        ]
        assert 'enable' in top_level_names, 'hook.py must define an enable() function'

    def test_hook_has_name_and_description(self):
        hook_path = os.path.join(PLUGIN_DIR, 'hook.py')
        with open(hook_path, encoding='utf-8') as f:
            source = f.read()
        tree = ast.parse(source)
        assigned_names = [
            node.targets[0].id
            for node in ast.walk(tree)
            if isinstance(node, ast.Assign) and isinstance(node.targets[0], ast.Name)
        ]
        assert 'name' in assigned_names, 'hook.py should assign a name variable'
        assert 'description' in assigned_names, 'hook.py should assign a description variable'

    def test_hook_registers_routes(self):
        hook_path = os.path.join(PLUGIN_DIR, 'hook.py')
        with open(hook_path, encoding='utf-8') as f:
            source = f.read()
        assert 'add_route' in source or 'add_static' in source, (
            'hook.py should register at least one route'
        )

    def test_hook_registers_special_payloads(self):
        hook_path = os.path.join(PLUGIN_DIR, 'hook.py')
        with open(hook_path, encoding='utf-8') as f:
            source = f.read()
        assert 'add_special_payload' in source, (
            'hook.py should register special payloads for dynamic compilation'
        )


class TestSandService:
    """Tests that SandService defines expected class structure and methods."""

    def test_sand_svc_module_is_valid_python(self):
        path = os.path.join(APP_DIR, 'sand_svc.py')
        assert os.path.isfile(path), 'sand_svc.py not found'
        with open(path, encoding='utf-8') as f:
            try:
                ast.parse(f.read())
            except SyntaxError as e:
                pytest.fail(f'sand_svc.py has syntax error: {e}')

    def test_sand_svc_defines_service_class(self):
        path = os.path.join(APP_DIR, 'sand_svc.py')
        with open(path, encoding='utf-8') as f:
            tree = ast.parse(f.read())
        class_names = [node.name for node in ast.walk(tree) if isinstance(node, ast.ClassDef)]
        assert 'SandService' in class_names, 'sand_svc.py should define SandService class'

    def test_sand_svc_has_compile_methods(self):
        path = os.path.join(APP_DIR, 'sand_svc.py')
        with open(path, encoding='utf-8') as f:
            tree = ast.parse(f.read())
        method_names = [
            node.name
            for node in ast.walk(tree)
            if isinstance(node, (ast.FunctionDef, ast.AsyncFunctionDef))
        ]
        assert 'dynamically_compile_executable' in method_names, (
            'SandService should have dynamically_compile_executable method'
        )
        assert 'dynamically_compile_library' in method_names, (
            'SandService should have dynamically_compile_library method'
        )

    def test_sand_svc_accepts_services_parameter(self):
        """Test that SandService.__init__ accepts a services parameter."""
        path = os.path.join(APP_DIR, 'sand_svc.py')
        with open(path, encoding='utf-8') as f:
            source = f.read()
        tree = ast.parse(source)
        # Find __init__ method in SandService
        for node in ast.walk(tree):
            if isinstance(node, ast.ClassDef) and node.name == 'SandService':
                for item in node.body:
                    if isinstance(item, (ast.FunctionDef, ast.AsyncFunctionDef)) and item.name == '__init__':
                        # Verify it accepts services parameter
                        args = [a.arg for a in item.args.args]
                        assert 'services' in args, (
                            'SandService.__init__ should accept services parameter'
                        )
                        break


class TestBaseExtension:
    """Tests that base_extension.py Extension class does not use shell=True unsafely."""

    def _get_source(self):
        path = os.path.join(APP_DIR, 'utility', 'base_extension.py')
        with open(path, encoding='utf-8') as f:
            return f.read()

    def test_base_extension_exists(self):
        path = os.path.join(APP_DIR, 'utility', 'base_extension.py')
        assert os.path.isfile(path), 'base_extension.py not found'

    def test_base_extension_defines_extension_class(self):
        source = self._get_source()
        tree = ast.parse(source)
        class_names = [node.name for node in ast.walk(tree) if isinstance(node, ast.ClassDef)]
        assert 'Extension' in class_names, 'base_extension.py should define Extension class'

    def test_base_extension_no_shell_true(self):
        source = self._get_source()
        tree = ast.parse(source)
        for node in ast.walk(tree):
            if isinstance(node, ast.Call):
                func = node.func
                # Check for subprocess calls with shell=True
                is_subprocess_call = False
                if isinstance(func, ast.Attribute):
                    if isinstance(func.value, ast.Name) and func.value.id == 'subprocess':
                        is_subprocess_call = True
                if is_subprocess_call:
                    for kw in node.keywords:
                        if kw.arg == 'shell':
                            if isinstance(kw.value, ast.Constant) and kw.value.value is True:
                                line = getattr(node, 'lineno', '?')
                                pytest.fail(
                                    f'base_extension.py line {line}: subprocess call uses '
                                    f'shell=True which is a security risk'
                                )


class TestPayloads:
    """Tests that payloads directory has expected files."""

    EXPECTED_PAYLOADS = [
        'sandcat.go-linux',
        'sandcat.go-darwin',
        'sandcat.go-windows',
    ]

    def test_payloads_directory_exists(self):
        assert os.path.isdir(PAYLOADS_DIR), 'payloads directory not found'

    def test_payloads_not_empty(self):
        files = os.listdir(PAYLOADS_DIR)
        assert len(files) > 0, 'payloads directory is empty'

    @pytest.mark.parametrize('payload_name', EXPECTED_PAYLOADS)
    def test_expected_payload_exists(self, payload_name):
        path = os.path.join(PAYLOADS_DIR, payload_name)
        assert os.path.isfile(path), f'Expected payload {payload_name} not found'

    def test_sandcat_elfload_payload_exists(self):
        path = os.path.join(PAYLOADS_DIR, 'sandcat-elfload.py')
        assert os.path.isfile(path), 'sandcat-elfload.py payload not found'


class TestAbilitiesYAML:
    """Tests that sandcat abilities YAML are valid."""

    @staticmethod
    def _collect_yaml_files():
        pattern = os.path.join(ABILITIES_DIR, '**', '*.yml')
        return glob.glob(pattern, recursive=True)

    def test_abilities_directory_exists(self):
        assert os.path.isdir(ABILITIES_DIR), 'abilities directory not found'

    def test_at_least_one_ability_exists(self):
        files = self._collect_yaml_files()
        assert len(files) > 0, 'No ability YAML files found'

    def test_all_abilities_are_parseable(self):
        for yml_file in self._collect_yaml_files():
            with open(yml_file, encoding='utf-8') as f:
                try:
                    data = yaml.safe_load(f)
                except yaml.YAMLError as e:
                    pytest.fail(f'Failed to parse {yml_file}: {e}')
                assert data is not None, f'{yml_file} is empty'

    def test_all_abilities_have_required_fields(self):
        for yml_file in self._collect_yaml_files():
            with open(yml_file, encoding='utf-8') as f:
                data = yaml.safe_load(f)
            if not isinstance(data, list):
                data = [data]
            for ability in data:
                for field in REQUIRED_ABILITY_FIELDS:
                    assert field in ability, (
                        f'{yml_file}: ability missing required field "{field}"'
                    )


class TestSandcatElfloadSecurity:
    """Tests that sandcat-elfload.py has timeout on requests calls."""

    def _get_source(self):
        path = os.path.join(PAYLOADS_DIR, 'sandcat-elfload.py')
        with open(path, encoding='utf-8') as f:
            return f.read()

    def test_elfload_is_valid_python(self):
        source = self._get_source()
        try:
            ast.parse(source)
        except SyntaxError as e:
            pytest.fail(f'sandcat-elfload.py has syntax error: {e}')

    def test_elfload_requests_have_timeout(self):
        source = self._get_source()
        tree = ast.parse(source)
        requests_methods = {'get', 'post', 'put', 'delete', 'patch', 'head'}
        missing = []
        for node in ast.walk(tree):
            if isinstance(node, ast.Call):
                func = node.func
                is_requests_call = False
                if isinstance(func, ast.Attribute) and func.attr in requests_methods:
                    if isinstance(func.value, ast.Name) and func.value.id == 'requests':
                        is_requests_call = True
                if is_requests_call:
                    keyword_names = [kw.arg for kw in node.keywords if kw.arg is not None]
                    if 'timeout' not in keyword_names:
                        line = getattr(node, 'lineno', '?')
                        missing.append(f'line {line}: requests.{func.attr}()')
        if missing:
            pytest.fail(
                f'sandcat-elfload.py has requests calls without timeout: {"; ".join(missing)}'
            )
