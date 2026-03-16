"""Tests for hook.py — plugin lifecycle and enable()."""
from unittest.mock import MagicMock, AsyncMock, patch

import pytest

import hook


class TestHookModuleAttributes:
    def test_name(self):
        assert hook.name == 'Sandcat'

    def test_description(self):
        assert hook.description == 'A custom multi-platform RAT'

    def test_address(self):
        assert hook.address == '/plugin/sandcat/gui'


class TestEnable:
    @pytest.mark.asyncio
    @patch('hook.which', return_value='/usr/bin/x86_64-w64-mingw32-gcc')
    @patch('hook.SandService')
    @patch('hook.SandGuiApi')
    async def test_enable_registers_payloads(self, mock_gui_cls, mock_svc_cls, mock_which, mock_services):
        mock_svc = MagicMock()
        mock_svc.dynamically_compile_executable = AsyncMock()
        mock_svc.dynamically_compile_library = AsyncMock()
        mock_svc.load_sandcat_extension_modules = AsyncMock()
        mock_svc_cls.return_value = mock_svc

        mock_gui = MagicMock()
        mock_gui.splash = AsyncMock()
        mock_gui_cls.return_value = mock_gui

        app = MagicMock()
        mock_services['app_svc'].application = app
        mock_services['file_svc'].add_special_payload = AsyncMock()

        await hook.enable(mock_services)

        # Should register both special payloads
        calls = mock_services['file_svc'].add_special_payload.call_args_list
        assert len(calls) == 2
        assert calls[0].args[0] == 'sandcat.go'
        assert calls[1].args[0] == 'shared.go'

    @pytest.mark.asyncio
    @patch('hook.which', return_value='/usr/bin/x86_64-w64-mingw32-gcc')
    @patch('hook.SandService')
    @patch('hook.SandGuiApi')
    async def test_enable_registers_routes(self, mock_gui_cls, mock_svc_cls, mock_which, mock_services):
        mock_svc = MagicMock()
        mock_svc.load_sandcat_extension_modules = AsyncMock()
        mock_svc_cls.return_value = mock_svc

        mock_gui = MagicMock()
        mock_gui_cls.return_value = mock_gui

        app = MagicMock()
        mock_services['app_svc'].application = app
        mock_services['file_svc'].add_special_payload = AsyncMock()

        await hook.enable(mock_services)

        app.router.add_static.assert_called_once()
        app.router.add_route.assert_called_once_with('GET', '/plugin/sandcat/gui', mock_gui.splash)

    @pytest.mark.asyncio
    @patch('hook.which', return_value='/usr/bin/x86_64-w64-mingw32-gcc')
    @patch('hook.SandService')
    @patch('hook.SandGuiApi')
    async def test_enable_loads_extensions(self, mock_gui_cls, mock_svc_cls, mock_which, mock_services):
        mock_svc = MagicMock()
        mock_svc.load_sandcat_extension_modules = AsyncMock()
        mock_svc_cls.return_value = mock_svc

        mock_gui_cls.return_value = MagicMock()
        mock_services['app_svc'].application = MagicMock()
        mock_services['file_svc'].add_special_payload = AsyncMock()

        await hook.enable(mock_services)
        mock_svc.load_sandcat_extension_modules.assert_awaited_once()

    @pytest.mark.asyncio
    @patch('hook.which', return_value=None)
    @patch('hook.SandService')
    @patch('hook.SandGuiApi')
    async def test_enable_warns_missing_mingw(self, mock_gui_cls, mock_svc_cls, mock_which, mock_services):
        mock_svc = MagicMock()
        mock_svc.load_sandcat_extension_modules = AsyncMock()
        mock_svc.log = MagicMock()
        mock_svc_cls.return_value = mock_svc

        mock_gui_cls.return_value = MagicMock()
        mock_services['app_svc'].application = MagicMock()
        mock_services['file_svc'].add_special_payload = AsyncMock()

        await hook.enable(mock_services)
        mock_svc.log.warning.assert_called_once()
        assert 'mingw' in mock_svc.log.warning.call_args.args[0].lower() or \
               'x86_64-w64-mingw32-gcc' in mock_svc.log.warning.call_args.args[0]
