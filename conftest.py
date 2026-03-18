"""Shared fixtures and mock services for sandcat plugin tests."""
import os
import sys
import shutil
import tempfile
from unittest.mock import AsyncMock, MagicMock

import pytest


# ---------------------------------------------------------------------------
# Fake caldera service stubs so that imports from app.utility.base_service,
# app.utility.base_world, and app.service.auth_svc never touch real code.
# ---------------------------------------------------------------------------

class _FakeBaseService:
    """Minimal stand-in for app.utility.base_service.BaseService."""

    def create_logger(self, name):
        import logging
        return logging.getLogger(name)


class _FakeBaseWorld:
    """Minimal stand-in for app.utility.base_world.BaseWorld."""
    _config = {}

    @classmethod
    def get_config(cls, prop=None, name=None):
        return cls._config.get(prop, '')

    @classmethod
    def set_config(cls, prop, value):
        cls._config[prop] = value

    @classmethod
    def clear_config(cls):
        cls._config.clear()


def _noop_decorator(fn):
    return fn


def _for_all_public_methods(decorator):
    def wrapper(cls):
        return cls
    return wrapper


# ---------------------------------------------------------------------------
# Session-scoped autouse fixture: inject fake modules into sys.modules once
# per test session so sandcat code can be imported without a real Caldera
# installation present.
# ---------------------------------------------------------------------------

@pytest.fixture(scope='session', autouse=True)
def _inject_fake_modules():
    """Populate sys.modules with thin stubs for Caldera framework modules."""
    repo_root = os.path.dirname(os.path.abspath(__file__))

    # Build fake module stubs
    base_service_mod = type(sys)('app.utility.base_service')
    base_service_mod.BaseService = _FakeBaseService

    base_world_mod = type(sys)('app.utility.base_world')
    base_world_mod.BaseWorld = _FakeBaseWorld

    auth_svc_mod = type(sys)('app.service.auth_svc')
    auth_svc_mod.check_authorization = _noop_decorator
    auth_svc_mod.for_all_public_methods = _for_all_public_methods

    jinja2_mod = type(sys)('aiohttp_jinja2')
    jinja2_mod.template = lambda name: (lambda fn: fn)

    # Ensure parent packages exist as proper namespace stubs with real paths
    pkg_paths = {
        'app': [os.path.join(repo_root, 'app')],
        'app.utility': [os.path.join(repo_root, 'app', 'utility')],
        'app.service': [],
        'app.extensions': [os.path.join(repo_root, 'app', 'extensions')],
        'app.extensions.contact': [os.path.join(repo_root, 'app', 'extensions', 'contact')],
        'app.extensions.donut': [os.path.join(repo_root, 'app', 'extensions', 'donut')],
        'app.extensions.execute': [os.path.join(repo_root, 'app', 'extensions', 'execute')],
        'app.extensions.execute.native': [os.path.join(repo_root, 'app', 'extensions', 'execute', 'native')],
        'app.extensions.execute.shellcode': [os.path.join(repo_root, 'app', 'extensions', 'execute', 'shellcode')],
        'app.extensions.execute.shells': [os.path.join(repo_root, 'app', 'extensions', 'execute', 'shells')],
        'app.extensions.proxy': [os.path.join(repo_root, 'app', 'extensions', 'proxy')],
        'app.extensions.shared': [os.path.join(repo_root, 'app', 'extensions', 'shared')],
    }
    for pkg, paths in pkg_paths.items():
        if pkg not in sys.modules:
            mod = type(sys)(pkg)
            mod.__path__ = paths
            sys.modules[pkg] = mod

    sys.modules['app.utility.base_service'] = base_service_mod
    sys.modules['app.utility.base_world'] = base_world_mod
    sys.modules['app.service.auth_svc'] = auth_svc_mod
    sys.modules['aiohttp_jinja2'] = jinja2_mod

    # Make the plugin package importable as plugins.sandcat
    if 'plugins' not in sys.modules:
        plugins_mod = type(sys)('plugins')
        plugins_mod.__path__ = [os.path.dirname(repo_root)]
        sys.modules['plugins'] = plugins_mod

    if 'plugins.sandcat' not in sys.modules:
        sandcat_mod = type(sys)('plugins.sandcat')
        sandcat_mod.__path__ = [repo_root]
        sys.modules['plugins.sandcat'] = sandcat_mod

    if 'plugins.sandcat.app' not in sys.modules:
        ps_app = type(sys)('plugins.sandcat.app')
        ps_app.__path__ = [os.path.join(repo_root, 'app')]
        sys.modules['plugins.sandcat.app'] = ps_app

    if 'plugins.sandcat.app.utility' not in sys.modules:
        ps_util = type(sys)('plugins.sandcat.app.utility')
        ps_util.__path__ = [os.path.join(repo_root, 'app', 'utility')]
        sys.modules['plugins.sandcat.app.utility'] = ps_util

    # Add repo root to sys.path so plain `from app.…` works for extension files
    if repo_root not in sys.path:
        sys.path.insert(0, repo_root)

    yield


# ---------------------------------------------------------------------------
# Fixtures
# ---------------------------------------------------------------------------

@pytest.fixture
def fake_base_world():
    """Provide and auto-clear the FakeBaseWorld config store."""
    _FakeBaseWorld.clear_config()
    yield _FakeBaseWorld
    _FakeBaseWorld.clear_config()


@pytest.fixture
def mock_services():
    """Return a dict of mock caldera services suitable for SandService / SandGuiApi."""
    file_svc = MagicMock()
    file_svc.add_special_payload = AsyncMock()
    file_svc.find_file_path = AsyncMock(return_value=('/fake', '/fake/sandcat.go'))
    file_svc.compile_go = AsyncMock()
    file_svc.sanitize_ldflag_value = MagicMock(side_effect=lambda p, v: v)
    file_svc.log = MagicMock()

    data_svc = MagicMock()
    data_svc.locate = AsyncMock(return_value=[])

    contact_svc = MagicMock()
    contact_svc.contacts = []

    app_svc = MagicMock()
    app_svc.retrieve_compiled_file = AsyncMock(return_value=('/path', 'sandcat-linux'))
    app_svc.application = MagicMock()

    auth_svc = MagicMock()

    return dict(
        file_svc=file_svc,
        data_svc=data_svc,
        contact_svc=contact_svc,
        app_svc=app_svc,
        auth_svc=auth_svc,
    )


@pytest.fixture
def sand_svc(mock_services):
    """Instantiate a SandService backed by mock services."""
    from app.sand_svc import SandService
    return SandService(mock_services)


@pytest.fixture
def tmp_dir():
    """Provide a temporary directory, cleaned up afterwards."""
    d = tempfile.mkdtemp()
    yield d
    shutil.rmtree(d, ignore_errors=True)
