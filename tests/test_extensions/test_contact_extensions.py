"""Tests for contact extensions: dns_tunneling, ftp, gist, slack."""
import re
from unittest.mock import patch, MagicMock

import pytest

from plugins.sandcat.app.utility.base_extension import Extension, ConfigFileException


# ========================================================================
# DNS Tunneling
# ========================================================================

class TestDnsTunneling:
    @pytest.fixture
    def dns_ext(self):
        from app.extensions.contact.dns_tunneling import DnsTunneling
        return DnsTunneling()

    @pytest.fixture
    def dns_load(self):
        from app.extensions.contact.dns_tunneling import load
        return load

    def test_load_returns_instance(self, dns_load):
        ext = dns_load()
        from app.extensions.contact.dns_tunneling import DnsTunneling
        assert isinstance(ext, DnsTunneling)

    def test_is_extension(self, dns_ext):
        assert isinstance(dns_ext, Extension)

    def test_files(self, dns_ext):
        assert dns_ext.files == [('dns_tunneling.go', 'contact')]

    def test_dependencies(self, dns_ext):
        assert dns_ext.dependencies == ['github.com/miekg/dns']

    def test_file_hooks_registered(self, dns_ext):
        assert 'dns_tunneling.go' in dns_ext.file_hooks

    @pytest.mark.asyncio
    async def test_hook_replaces_domain(self, dns_ext, fake_base_world):
        fake_base_world.set_config('app.contact.dns.domain', 'evil.com')
        result = await dns_ext.hook_set_custom_domain('prefix {DNS_TUNNELING_C2_DOMAIN} suffix')
        assert result == 'prefix evil.com suffix'

    @pytest.mark.asyncio
    async def test_hook_no_domain_raises(self, dns_ext, fake_base_world):
        # No config set
        with pytest.raises(Exception, match='No DNS tunneling domain'):
            await dns_ext.hook_set_custom_domain('data')

    @pytest.mark.asyncio
    async def test_hook_replaces_only_first(self, dns_ext, fake_base_world):
        fake_base_world.set_config('app.contact.dns.domain', 'x.com')
        data = '{DNS_TUNNELING_C2_DOMAIN} {DNS_TUNNELING_C2_DOMAIN}'
        result = await dns_ext.hook_set_custom_domain(data)
        assert result.count('x.com') == 1


# ========================================================================
# FTP
# ========================================================================

class TestFTP:
    @pytest.fixture
    def ftp_ext(self):
        from app.extensions.contact.ftp import FTP
        return FTP()

    @pytest.fixture
    def ftp_load(self):
        from app.extensions.contact.ftp import load
        return load

    def test_load_returns_instance(self, ftp_load):
        ext = ftp_load()
        from app.extensions.contact.ftp import FTP
        assert isinstance(ext, FTP)

    def test_is_extension(self, ftp_ext):
        assert isinstance(ftp_ext, Extension)

    def test_files(self, ftp_ext):
        assert ftp_ext.files == [('ftp.go', 'contact')]

    def test_dependencies(self, ftp_ext):
        assert ftp_ext.dependencies == ['github.com/jlaffaye/ftp']

    def test_file_hooks_registered(self, ftp_ext):
        assert 'ftp.go' in ftp_ext.file_hooks

    @pytest.mark.asyncio
    async def test_hook_replaces_all_vars(self, ftp_ext, fake_base_world):
        fake_base_world.set_config('app.contact.ftp.user', 'myuser')
        fake_base_world.set_config('app.contact.ftp.pword', 'mypass')
        fake_base_world.set_config('app.contact.ftp.server.dir', '/uploads')
        data = '{FTP_C2_USER} {FTP_C2_PASSWORD} {FTP_C2_DIRECTORY}'
        result = await ftp_ext.hook_set_custom_values(data)
        assert 'myuser' in result
        assert 'mypass' in result
        assert '/uploads' in result

    @pytest.mark.asyncio
    async def test_hook_missing_var_raises(self, ftp_ext, fake_base_world):
        # all missing - first var triggers the exception
        with pytest.raises(ConfigFileException, match='app.contact.ftp.user'):
            await ftp_ext.hook_set_custom_values('{FTP_C2_USER} {FTP_C2_PASSWORD} {FTP_C2_DIRECTORY}')

    @pytest.mark.asyncio
    async def test_hook_missing_second_var_raises(self, ftp_ext, fake_base_world):
        fake_base_world.set_config('app.contact.ftp.user', 'myuser')
        # missing pword
        with pytest.raises(ConfigFileException, match='app.contact.ftp.pword'):
            await ftp_ext.hook_set_custom_values('{FTP_C2_USER} {FTP_C2_PASSWORD} {FTP_C2_DIRECTORY}')


# ========================================================================
# Gist
# ========================================================================

class TestGist:
    @pytest.fixture
    def gist_ext(self):
        from app.extensions.contact.gist import Gist
        return Gist()

    @pytest.fixture
    def gist_load(self):
        from app.extensions.contact.gist import load
        return load

    def test_load_returns_instance(self, gist_load):
        ext = gist_load()
        from app.extensions.contact.gist import Gist
        assert isinstance(ext, Gist)

    def test_is_extension(self, gist_ext):
        assert isinstance(gist_ext, Extension)

    def test_files(self, gist_ext):
        assert gist_ext.files == [('gist.go', 'contact'), ('util.go', 'contact')]

    def test_dependencies(self, gist_ext):
        assert 'github.com/google/go-github/github' in gist_ext.dependencies
        assert 'golang.org/x/oauth2' in gist_ext.dependencies

    def test_no_file_hooks(self, gist_ext):
        assert gist_ext.file_hooks == {}


# ========================================================================
# Slack
# ========================================================================

class TestSlack:
    @pytest.fixture
    def slack_ext(self):
        from app.extensions.contact.slack import Slack
        return Slack()

    @pytest.fixture
    def slack_load(self):
        from app.extensions.contact.slack import load
        return load

    def test_load_returns_instance(self, slack_load):
        ext = slack_load()
        from app.extensions.contact.slack import Slack
        assert isinstance(ext, Slack)

    def test_is_extension(self, slack_ext):
        assert isinstance(slack_ext, Extension)

    def test_files(self, slack_ext):
        assert slack_ext.files == [('slack.go', 'contact'), ('util.go', 'contact')]

    def test_no_dependencies(self, slack_ext):
        assert slack_ext.dependencies == []

    def test_file_hooks_registered(self, slack_ext):
        assert 'slack.go' in slack_ext.file_hooks

    @pytest.mark.asyncio
    async def test_hook_replaces_channel(self, slack_ext, fake_base_world):
        fake_base_world.set_config('app.contact.slack.channel_id', 'C12345')
        result = await slack_ext.hook_set_custom_channel('channel={SLACK_C2_CHANNEL_ID}')
        assert result == 'channel=C12345'

    @pytest.mark.asyncio
    async def test_hook_no_channel_raises(self, slack_ext, fake_base_world):
        with pytest.raises(Exception, match='No Slack channel ID'):
            await slack_ext.hook_set_custom_channel('data')

    @pytest.mark.asyncio
    async def test_hook_replaces_only_first(self, slack_ext, fake_base_world):
        fake_base_world.set_config('app.contact.slack.channel_id', 'C99')
        data = '{SLACK_C2_CHANNEL_ID} {SLACK_C2_CHANNEL_ID}'
        result = await slack_ext.hook_set_custom_channel(data)
        assert result.count('C99') == 1
