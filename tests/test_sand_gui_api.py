"""Tests for app/sand_gui_api.py — SandGuiApi."""
from unittest.mock import MagicMock, AsyncMock

import pytest

from app.sand_gui_api import SandGuiApi


class TestSandGuiApiInit:
    def test_stores_auth_svc(self, mock_services):
        api = SandGuiApi(services=mock_services)
        assert api.auth_svc is mock_services['auth_svc']


class TestSplash:
    @pytest.mark.asyncio
    async def test_splash_returns_empty_dict(self, mock_services):
        api = SandGuiApi(services=mock_services)
        request = MagicMock()
        result = await api.splash(request)
        assert result == {}

    @pytest.mark.asyncio
    async def test_splash_is_callable(self, mock_services):
        api = SandGuiApi(services=mock_services)
        assert callable(api.splash)
