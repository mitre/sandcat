"""Shared fixtures and mock services for sandcat plugin tests."""
import os
import sys
import shutil
import tempfile
import asyncio
from unittest.mock import AsyncMock, MagicMock, PropertyMock, patch

import pytest


# ---------------------------------------------------------------------------
# Fake caldera service stubs so that imports from app.utility.base_service,
# app.utility.base_world, and app.service.auth_svc never touch real code.
# We inject thin fakes into sys.modules *before* any sandcat code is imported.
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


# Build fake module trees ------------------------------------------------
_base_service_mod = type(sys)('app.utility.base_service')
_base_service_mod.BaseService = _FakeBaseService

_base_world_mod = type(sys)('app.utility.base_world')
_base_world_mod.BaseWorld = _FakeBaseWorld

_auth_svc_mod = type(sys)('app.service.auth_svc')
_auth_svc_mod.check_authorization = _noop_decorator
_auth_svc_mod.for_all_public_methods = _for_all_public_methods

# aiohttp_jinja2 stub
_jinja2_mod = type(sys)('aiohttp_jinja2')
_jinja2_mod.template = lambda name: (lambda fn: fn)

# _REPO_ROOT needed early for path setup
_REPO_ROOT = os.path.dirname(os.path.abspath(__file__))

# Ensure parent packages exist as proper namespace stubs with real paths
_pkg_paths = {
    'app': [os.path.join(_REPO_ROOT, 'app')],
    'app.utility': [os.path.join(_REPO_ROOT, 'app', 'utility')],
    'app.service': [],
    'app.extensions': [os.path.join(_REPO_ROOT, 'app', 'extensions')],
    'app.extensions.contact': [os.path.join(_REPO_ROOT, 'app', 'extensions', 'contact')],
    'app.extensions.donut': [os.path.join(_REPO_ROOT, 'app', 'extensions', 'donut')],
    'app.extensions.execute': [os.path.join(_REPO_ROOT, 'app', 'extensions', 'execute')],
    'app.extensions.execute.native': [os.path.join(_REPO_ROOT, 'app', 'extensions', 'execute', 'native')],
    'app.extensions.execute.shellcode': [os.path.join(_REPO_ROOT, 'app', 'extensions', 'execute', 'shellcode')],
    'app.extensions.execute.shells': [os.path.join(_REPO_ROOT, 'app', 'extensions', 'execute', 'shells')],
    'app.extensions.proxy': [os.path.join(_REPO_ROOT, 'app', 'extensions', 'proxy')],
    'app.extensions.shared': [os.path.join(_REPO_ROOT, 'app', 'extensions', 'shared')],
}
for pkg, paths in _pkg_paths.items():
    if pkg not in sys.modules:
        mod = type(sys)(pkg)
        mod.__path__ = paths
        sys.modules[pkg] = mod

sys.modules['app.utility.base_service'] = _base_service_mod
sys.modules['app.utility.base_world'] = _base_world_mod
sys.modules['app.service.auth_svc'] = _auth_svc_mod
sys.modules['aiohttp_jinja2'] = _jinja2_mod

# Now make the *plugin* package importable.
# Create a `plugins.sandcat` namespace that maps to the repo root.
if 'plugins' not in sys.modules:
    _plugins_mod = type(sys)('plugins')
    _plugins_mod.__path__ = [os.path.dirname(_REPO_ROOT)]  # parent of sandcat-pytest
    sys.modules['plugins'] = _plugins_mod

if 'plugins.sandcat' not in sys.modules:
    _sandcat_mod = type(sys)('plugins.sandcat')
    _sandcat_mod.__path__ = [_REPO_ROOT]
    sys.modules['plugins.sandcat'] = _sandcat_mod

if 'plugins.sandcat.app' not in sys.modules:
    _ps_app = type(sys)('plugins.sandcat.app')
    _ps_app.__path__ = [os.path.join(_REPO_ROOT, 'app')]
    sys.modules['plugins.sandcat.app'] = _ps_app

if 'plugins.sandcat.app.utility' not in sys.modules:
    _ps_util = type(sys)('plugins.sandcat.app.utility')
    _ps_util.__path__ = [os.path.join(_REPO_ROOT, 'app', 'utility')]
    sys.modules['plugins.sandcat.app.utility'] = _ps_util

# Also add repo root to sys.path so plain `from app.…` works for extension files
if _REPO_ROOT not in sys.path:
    sys.path.insert(0, _REPO_ROOT)


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
