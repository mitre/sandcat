"""Tests for proxy extensions: proxy_http, proxy_smb_pipe."""
import pytest

from plugins.sandcat.app.utility.base_extension import Extension


# ========================================================================
# Proxy HTTP
# ========================================================================

class TestProxyHttp:
    @pytest.fixture
    def http_ext(self):
        from app.extensions.proxy.proxy_http import ProxyHttp
        return ProxyHttp()

    @pytest.fixture
    def http_load(self):
        from app.extensions.proxy.proxy_http import load
        return load

    def test_load_returns_instance(self, http_load):
        ext = http_load()
        from app.extensions.proxy.proxy_http import ProxyHttp
        assert isinstance(ext, ProxyHttp)

    def test_is_extension(self, http_ext):
        assert isinstance(http_ext, Extension)

    def test_files(self, http_ext):
        assert http_ext.files == [('proxy_receiver_http.go', 'proxy')]

    def test_no_dependencies(self, http_ext):
        assert http_ext.dependencies == []

    def test_no_file_hooks(self, http_ext):
        assert http_ext.file_hooks == {}


# ========================================================================
# Proxy SMB Pipe
# ========================================================================

class TestProxySmbPipe:
    @pytest.fixture
    def smb_ext(self):
        from app.extensions.proxy.proxy_smb_pipe import ProxySmbPipe
        return ProxySmbPipe()

    @pytest.fixture
    def smb_load(self):
        from app.extensions.proxy.proxy_smb_pipe import load
        return load

    def test_load_returns_instance(self, smb_load):
        ext = smb_load()
        from app.extensions.proxy.proxy_smb_pipe import ProxySmbPipe
        assert isinstance(ext, ProxySmbPipe)

    def test_is_extension(self, smb_ext):
        assert isinstance(smb_ext, Extension)

    def test_files(self, smb_ext):
        expected = [
            ('proxy_smb_pipe.go', 'proxy'),
            ('proxy_smb_pipe_util.go', 'proxy'),
        ]
        assert smb_ext.files == expected

    def test_dependencies(self, smb_ext):
        assert smb_ext.dependencies == ['gopkg.in/natefinch/npipe.v2']

    def test_no_file_hooks(self, smb_ext):
        assert smb_ext.file_hooks == {}
