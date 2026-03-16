"""Exhaustive tests for app/sand_svc.py — SandService."""
import base64
import json
import os
import string
from collections import defaultdict
from unittest.mock import AsyncMock, MagicMock, patch, PropertyMock

import pytest

from app.sand_svc import (
    SandService,
    default_flag_params,
    library_flag_params,
    gocat_variants,
    default_gocat_variant,
)


# ========================================================================
# Module-level constants
# ========================================================================

class TestModuleConstants:
    def test_default_flag_params_is_tuple(self):
        assert isinstance(default_flag_params, tuple)

    def test_default_flag_params_contents(self):
        expected = ('server', 'group', 'listenP2P', 'c2', 'includeProxyPeers', 'userAgent')
        assert default_flag_params == expected

    def test_library_flag_params(self):
        assert library_flag_params == ('runOnInit',)

    def test_gocat_variants_keys(self):
        assert set(gocat_variants.keys()) == {'basic', 'red'}

    def test_gocat_basic_variant_empty(self):
        assert gocat_variants['basic'] == set()

    def test_gocat_red_variant_extensions(self):
        assert gocat_variants['red'] == {'gist', 'shared', 'shells', 'shellcode'}

    def test_default_gocat_variant_value(self):
        assert default_gocat_variant == 'basic'


# ========================================================================
# SandService.__init__
# ========================================================================

class TestSandServiceInit:
    def test_service_attributes(self, sand_svc, mock_services):
        assert sand_svc.file_svc is mock_services['file_svc']
        assert sand_svc.data_svc is mock_services['data_svc']
        assert sand_svc.contact_svc is mock_services['contact_svc']
        assert sand_svc.app_svc is mock_services['app_svc']

    def test_sandcat_dir_is_relative(self, sand_svc):
        assert sand_svc.sandcat_dir == os.path.join('plugins', 'sandcat')

    def test_sandcat_extensions_starts_empty(self, sand_svc):
        assert sand_svc.sandcat_extensions == {}

    def test_logger_created(self, sand_svc):
        assert sand_svc.log is not None


# ========================================================================
# _generate_key
# ========================================================================

class TestGenerateKey:
    def test_default_length(self):
        key = SandService._generate_key()
        assert len(key) == 30

    def test_custom_length(self):
        for size in (0, 1, 10, 50, 100):
            assert len(SandService._generate_key(size=size)) == size

    def test_characters_in_allowed_set(self):
        allowed = set(string.ascii_uppercase + string.digits)
        for _ in range(20):
            key = SandService._generate_key()
            assert set(key).issubset(allowed)

    def test_randomness(self):
        keys = {SandService._generate_key() for _ in range(50)}
        assert len(keys) > 1  # extremely unlikely to collide 50 times


# ========================================================================
# _get_c2_config
# ========================================================================

class TestGetC2Config:
    @pytest.mark.asyncio
    async def test_returns_config_when_match(self, sand_svc):
        c2 = MagicMock()
        c2.name = 'HTTP'
        c2.retrieve_config.return_value = 'http-config-value'
        sand_svc.contact_svc.contacts = [c2]
        key, val = await sand_svc._get_c2_config('HTTP')
        assert key == 'c2Key'
        assert val == 'http-config-value'

    @pytest.mark.asyncio
    async def test_returns_empty_when_no_match(self, sand_svc):
        c2 = MagicMock()
        c2.name = 'HTTP'
        sand_svc.contact_svc.contacts = [c2]
        key, val = await sand_svc._get_c2_config('DNS')
        assert key == ''
        assert val == ''

    @pytest.mark.asyncio
    async def test_returns_empty_when_no_contacts(self, sand_svc):
        sand_svc.contact_svc.contacts = []
        key, val = await sand_svc._get_c2_config('HTTP')
        assert key == ''
        assert val == ''

    @pytest.mark.asyncio
    async def test_returns_first_matching_contact(self, sand_svc):
        c1 = MagicMock(); c1.name = 'HTTP'; c1.retrieve_config.return_value = 'first'
        c2 = MagicMock(); c2.name = 'HTTP'; c2.retrieve_config.return_value = 'second'
        sand_svc.contact_svc.contacts = [c1, c2]
        _, val = await sand_svc._get_c2_config('HTTP')
        assert val == 'first'


# ========================================================================
# _obtain_extensions_from_headers
# ========================================================================

class TestObtainExtensionsFromHeaders:
    @pytest.mark.asyncio
    async def test_no_extensions_no_variant(self, sand_svc):
        headers = {}
        result = await sand_svc._obtain_extensions_from_headers(headers)
        assert result == set()  # basic variant has empty set

    @pytest.mark.asyncio
    async def test_explicit_basic_variant(self, sand_svc):
        headers = {'gocat-variant': 'basic'}
        result = await sand_svc._obtain_extensions_from_headers(headers)
        assert result == set()

    @pytest.mark.asyncio
    async def test_red_variant(self, sand_svc):
        headers = {'gocat-variant': 'red'}
        result = await sand_svc._obtain_extensions_from_headers(headers)
        assert result == {'gist', 'shared', 'shells', 'shellcode'}

    @pytest.mark.asyncio
    async def test_extensions_from_header_string(self, sand_svc):
        headers = {'gocat-extensions': 'dns_tunneling,ftp'}
        result = await sand_svc._obtain_extensions_from_headers(headers)
        assert result == {'dns_tunneling', 'ftp'}

    @pytest.mark.asyncio
    async def test_extensions_merged_with_variant(self, sand_svc):
        headers = {'gocat-variant': 'red', 'gocat-extensions': 'dns_tunneling'}
        result = await sand_svc._obtain_extensions_from_headers(headers)
        assert 'dns_tunneling' in result
        assert 'gist' in result  # from red variant

    @pytest.mark.asyncio
    async def test_empty_extension_string(self, sand_svc):
        headers = {'gocat-extensions': ''}
        result = await sand_svc._obtain_extensions_from_headers(headers)
        assert result == set()

    @pytest.mark.asyncio
    async def test_unknown_variant_returns_empty(self, sand_svc):
        headers = {'gocat-variant': 'nonexistent'}
        result = await sand_svc._obtain_extensions_from_headers(headers)
        assert result == set()

    @pytest.mark.asyncio
    async def test_single_extension(self, sand_svc):
        headers = {'gocat-extensions': 'shared'}
        result = await sand_svc._obtain_extensions_from_headers(headers)
        assert result == {'shared'}


# ========================================================================
# dynamically_compile_executable
# ========================================================================

class TestDynamicallyCompileExecutable:
    @pytest.mark.asyncio
    @patch('app.sand_svc.which', return_value='/usr/bin/go')
    async def test_calls_compile_with_go_present(self, mock_which, sand_svc):
        sand_svc._compile_new_agent = AsyncMock()
        headers = {'file': 'sandcat.go', 'platform': 'linux'}
        await sand_svc.dynamically_compile_executable(headers)
        sand_svc._compile_new_agent.assert_awaited_once()
        call_kwargs = sand_svc._compile_new_agent.call_args
        assert call_kwargs.kwargs['platform'] == 'linux'
        assert call_kwargs.kwargs['compile_target_name'] == 'sandcat.go'
        assert call_kwargs.kwargs['cflags'] == 'CGO_ENABLED=0'
        assert call_kwargs.kwargs['compile_target_dir'] == 'gocat'

    @pytest.mark.asyncio
    @patch('app.sand_svc.which', return_value=None)
    async def test_skips_compile_without_go(self, mock_which, sand_svc):
        sand_svc._compile_new_agent = AsyncMock()
        headers = {'file': 'sandcat.go', 'platform': 'linux'}
        await sand_svc.dynamically_compile_executable(headers)
        sand_svc._compile_new_agent.assert_not_awaited()

    @pytest.mark.asyncio
    @patch('app.sand_svc.which', return_value='/usr/bin/go')
    async def test_returns_compiled_file(self, mock_which, sand_svc):
        sand_svc._compile_new_agent = AsyncMock()
        sand_svc.app_svc.retrieve_compiled_file = AsyncMock(return_value='compiled-binary')
        headers = {'file': 'sandcat.go', 'platform': 'windows'}
        result = await sand_svc.dynamically_compile_executable(headers)
        assert result == 'compiled-binary'
        sand_svc.app_svc.retrieve_compiled_file.assert_awaited_with('sandcat.go', 'windows', location='payloads')

    @pytest.mark.asyncio
    @patch('app.sand_svc.which', return_value='/usr/bin/go')
    async def test_extensions_passed_through(self, mock_which, sand_svc):
        sand_svc._compile_new_agent = AsyncMock()
        headers = {'file': 'sandcat.go', 'platform': 'linux', 'gocat-extensions': 'gist,shared'}
        await sand_svc.dynamically_compile_executable(headers)
        ext_names = sand_svc._compile_new_agent.call_args.kwargs['extension_names']
        assert 'gist' in ext_names
        assert 'shared' in ext_names


# ========================================================================
# dynamically_compile_library
# ========================================================================

class TestDynamicallyCompileLibrary:
    @pytest.mark.asyncio
    @patch('app.sand_svc.which', return_value='/usr/bin/go')
    async def test_always_includes_shared(self, mock_which, sand_svc):
        sand_svc._compile_new_agent = AsyncMock()
        headers = {'file': 'shared.go', 'platform': 'linux'}
        await sand_svc.dynamically_compile_library(headers)
        ext_names = sand_svc._compile_new_agent.call_args.kwargs['extension_names']
        assert 'shared' in ext_names

    @pytest.mark.asyncio
    @patch('app.sand_svc.which', return_value='/usr/bin/go')
    async def test_linux_compile_options(self, mock_which, sand_svc):
        sand_svc._compile_new_agent = AsyncMock()
        headers = {'file': 'shared.go', 'platform': 'linux'}
        await sand_svc.dynamically_compile_library(headers)
        call_kwargs = sand_svc._compile_new_agent.call_args.kwargs
        assert call_kwargs['cflags'] == 'CGO_ENABLED=1'
        assert call_kwargs['compile_target_dir'] == 'gocat/shared'
        assert '--buildmode=c-shared' in call_kwargs.get('buildmode', '')

    @pytest.mark.asyncio
    @patch('app.sand_svc.which', side_effect=lambda x: '/usr/bin/go' if x == 'go' else '/usr/bin/x86_64-w64-mingw32-gcc')
    async def test_windows_compile_with_mingw(self, mock_which, sand_svc):
        sand_svc._compile_new_agent = AsyncMock()
        headers = {'file': 'shared.go', 'platform': 'windows'}
        await sand_svc.dynamically_compile_library(headers)
        call_kwargs = sand_svc._compile_new_agent.call_args.kwargs
        assert 'CC=x86_64-w64-mingw32-gcc' in call_kwargs['cflags']

    @pytest.mark.asyncio
    @patch('app.sand_svc.which', side_effect=lambda x: '/usr/bin/go' if x == 'go' else None)
    async def test_windows_compile_without_mingw_raises(self, mock_which, sand_svc):
        sand_svc._compile_new_agent = AsyncMock()
        headers = {'file': 'shared.go', 'platform': 'windows'}
        with pytest.raises(Exception, match='Missing dependency for cross compilation'):
            await sand_svc.dynamically_compile_library(headers)

    @pytest.mark.asyncio
    @patch('app.sand_svc.which', return_value=None)
    async def test_skips_compile_without_go(self, mock_which, sand_svc):
        sand_svc._compile_new_agent = AsyncMock()
        headers = {'file': 'shared.go', 'platform': 'linux'}
        await sand_svc.dynamically_compile_library(headers)
        sand_svc._compile_new_agent.assert_not_awaited()

    @pytest.mark.asyncio
    @patch('app.sand_svc.which', return_value='/usr/bin/go')
    async def test_unsupported_platform_skips_compile(self, mock_which, sand_svc):
        sand_svc._compile_new_agent = AsyncMock()
        headers = {'file': 'shared.go', 'platform': 'darwin'}
        await sand_svc.dynamically_compile_library(headers)
        sand_svc._compile_new_agent.assert_not_awaited()

    @pytest.mark.asyncio
    @patch('app.sand_svc.which', return_value='/usr/bin/go')
    async def test_library_flag_params_included(self, mock_which, sand_svc):
        sand_svc._compile_new_agent = AsyncMock()
        headers = {'file': 'shared.go', 'platform': 'linux'}
        await sand_svc.dynamically_compile_library(headers)
        call_kwargs = sand_svc._compile_new_agent.call_args.kwargs
        assert call_kwargs['flag_params'] == default_flag_params + library_flag_params


# ========================================================================
# _compile_new_agent
# ========================================================================

class TestCompileNewAgent:
    @pytest.mark.asyncio
    @patch('app.sand_svc.SandService._generate_key', return_value='TESTKEY123')
    async def test_basic_compilation_flow(self, mock_key, sand_svc):
        sand_svc._install_gocat_extensions = AsyncMock(return_value=[])
        sand_svc._uninstall_gocat_extensions = AsyncMock()
        headers = {'file': 'sandcat.go', 'platform': 'linux'}
        await sand_svc._compile_new_agent(
            platform='linux', headers=headers,
            compile_target_name='sandcat.go', output_name='sandcat.go',
            cflags='CGO_ENABLED=0', compile_target_dir='gocat'
        )
        sand_svc.file_svc.compile_go.assert_awaited_once()
        call_kwargs = sand_svc.file_svc.compile_go.call_args
        assert call_kwargs.args[0] == 'linux'  # platform
        assert 'TESTKEY123' in call_kwargs.kwargs.get('ldflags', '') or 'TESTKEY123' in str(call_kwargs)

    @pytest.mark.asyncio
    async def test_architecture_defaults_to_amd64(self, sand_svc):
        sand_svc._install_gocat_extensions = AsyncMock(return_value=[])
        sand_svc._uninstall_gocat_extensions = AsyncMock()
        headers = {'file': 'sandcat.go', 'platform': 'linux'}
        await sand_svc._compile_new_agent(
            platform='linux', headers=headers,
            compile_target_name='sandcat.go', output_name='sandcat.go',
            compile_target_dir='gocat'
        )
        call_kwargs = sand_svc.file_svc.compile_go.call_args.kwargs
        assert call_kwargs['arch'] == 'amd64'

    @pytest.mark.asyncio
    async def test_custom_architecture(self, sand_svc):
        sand_svc._install_gocat_extensions = AsyncMock(return_value=[])
        sand_svc._uninstall_gocat_extensions = AsyncMock()
        headers = {'file': 'sandcat.go', 'platform': 'linux', 'architecture': 'arm64'}
        await sand_svc._compile_new_agent(
            platform='linux', headers=headers,
            compile_target_name='sandcat.go', output_name='sandcat.go',
            compile_target_dir='gocat'
        )
        call_kwargs = sand_svc.file_svc.compile_go.call_args.kwargs
        assert call_kwargs['arch'] == 'arm64'

    @pytest.mark.asyncio
    async def test_server_ldflag(self, sand_svc):
        sand_svc._install_gocat_extensions = AsyncMock(return_value=[])
        sand_svc._uninstall_gocat_extensions = AsyncMock()
        headers = {'file': 'sandcat.go', 'platform': 'linux', 'server': 'http://localhost:8888'}
        await sand_svc._compile_new_agent(
            platform='linux', headers=headers,
            compile_target_name='sandcat.go', output_name='sandcat.go',
            compile_target_dir='gocat'
        )
        ldflags = sand_svc.file_svc.compile_go.call_args.kwargs['ldflags']
        assert 'main.server=http://localhost:8888' in ldflags

    @pytest.mark.asyncio
    async def test_c2_ldflag_uses_c2_config(self, sand_svc):
        c2 = MagicMock(); c2.name = 'HTTP'; c2.retrieve_config.return_value = 'cfg_val'
        sand_svc.contact_svc.contacts = [c2]
        sand_svc._install_gocat_extensions = AsyncMock(return_value=[])
        sand_svc._uninstall_gocat_extensions = AsyncMock()
        headers = {'file': 'sandcat.go', 'platform': 'linux', 'c2': 'HTTP'}
        await sand_svc._compile_new_agent(
            platform='linux', headers=headers,
            compile_target_name='sandcat.go', output_name='sandcat.go',
            compile_target_dir='gocat'
        )
        ldflags = sand_svc.file_svc.compile_go.call_args.kwargs['ldflags']
        assert 'main.c2Key=cfg_val' in ldflags

    @pytest.mark.asyncio
    async def test_proxy_peers_ldflag(self, sand_svc):
        agent = MagicMock()
        agent.proxy_receivers = {'HTTP': ['addr1']}
        sand_svc.data_svc.locate = AsyncMock(return_value=[agent])
        sand_svc._install_gocat_extensions = AsyncMock(return_value=[])
        sand_svc._uninstall_gocat_extensions = AsyncMock()
        headers = {'file': 'sandcat.go', 'platform': 'linux', 'includeProxyPeers': 'all'}
        await sand_svc._compile_new_agent(
            platform='linux', headers=headers,
            compile_target_name='sandcat.go', output_name='sandcat.go',
            compile_target_dir='gocat'
        )
        ldflags = sand_svc.file_svc.compile_go.call_args.kwargs['ldflags']
        assert 'encodedReceivers' in ldflags
        assert 'receiverKey' in ldflags

    @pytest.mark.asyncio
    async def test_extensions_installed_and_uninstalled(self, sand_svc):
        sand_svc._install_gocat_extensions = AsyncMock(return_value=['ext1'])
        sand_svc._uninstall_gocat_extensions = AsyncMock()
        headers = {'file': 'sandcat.go', 'platform': 'linux'}
        await sand_svc._compile_new_agent(
            platform='linux', headers=headers,
            compile_target_name='sandcat.go', output_name='sandcat.go',
            extension_names={'ext1'}, compile_target_dir='gocat'
        )
        sand_svc._uninstall_gocat_extensions.assert_awaited_with(['ext1'])

    @pytest.mark.asyncio
    async def test_output_path_contains_platform(self, sand_svc):
        sand_svc._install_gocat_extensions = AsyncMock(return_value=[])
        sand_svc._uninstall_gocat_extensions = AsyncMock()
        headers = {'file': 'sandcat.go', 'platform': 'darwin'}
        await sand_svc._compile_new_agent(
            platform='darwin', headers=headers,
            compile_target_name='sandcat.go', output_name='agent',
            compile_target_dir='gocat'
        )
        output = sand_svc.file_svc.compile_go.call_args.args[1]
        assert 'agent-darwin' in output

    @pytest.mark.asyncio
    async def test_extldflags_appended_to_ldflags(self, sand_svc):
        sand_svc._install_gocat_extensions = AsyncMock(return_value=[])
        sand_svc._uninstall_gocat_extensions = AsyncMock()
        headers = {'file': 'sandcat.go', 'platform': 'linux'}
        await sand_svc._compile_new_agent(
            platform='linux', headers=headers,
            compile_target_name='sandcat.go', output_name='sandcat.go',
            extldflags='-extldflags "-Wl,--nxcompat"',
            compile_target_dir='gocat'
        )
        ldflags = sand_svc.file_svc.compile_go.call_args.kwargs['ldflags']
        assert '-extldflags' in ldflags

    @pytest.mark.asyncio
    async def test_buildmode_passed_through(self, sand_svc):
        sand_svc._install_gocat_extensions = AsyncMock(return_value=[])
        sand_svc._uninstall_gocat_extensions = AsyncMock()
        headers = {'file': 'shared.go', 'platform': 'linux'}
        await sand_svc._compile_new_agent(
            platform='linux', headers=headers,
            compile_target_name='shared.go', output_name='shared.go',
            buildmode='--buildmode=c-shared',
            compile_target_dir='gocat/shared'
        )
        call_kwargs = sand_svc.file_svc.compile_go.call_args.kwargs
        assert call_kwargs['buildmode'] == '--buildmode=c-shared'


# ========================================================================
# _get_available_proxy_peer_info
# ========================================================================

class TestGetAvailableProxyPeerInfo:
    @pytest.mark.asyncio
    async def test_no_agents(self, sand_svc):
        sand_svc.data_svc.locate = AsyncMock(return_value=[])
        result = await sand_svc._get_available_proxy_peer_info(set())
        assert json.loads(result) == {}

    @pytest.mark.asyncio
    async def test_single_agent_single_protocol(self, sand_svc):
        agent = MagicMock()
        agent.proxy_receivers = {'HTTP': ['addr1', 'addr2']}
        sand_svc.data_svc.locate = AsyncMock(return_value=[agent])
        result = json.loads(await sand_svc._get_available_proxy_peer_info(set()))
        assert sorted(result['HTTP']) == ['addr1', 'addr2']

    @pytest.mark.asyncio
    async def test_deduplication(self, sand_svc):
        a1 = MagicMock(); a1.proxy_receivers = {'HTTP': ['addr1', 'addr2']}
        a2 = MagicMock(); a2.proxy_receivers = {'HTTP': ['addr2', 'addr3']}
        sand_svc.data_svc.locate = AsyncMock(return_value=[a1, a2])
        result = json.loads(await sand_svc._get_available_proxy_peer_info(set()))
        assert sorted(result['HTTP']) == ['addr1', 'addr2', 'addr3']

    @pytest.mark.asyncio
    async def test_filter_include(self, sand_svc):
        agent = MagicMock()
        agent.proxy_receivers = {'HTTP': ['a1'], 'SMB': ['a2']}
        sand_svc.data_svc.locate = AsyncMock(return_value=[agent])
        result = json.loads(await sand_svc._get_available_proxy_peer_info({'HTTP'}))
        assert 'HTTP' in result
        assert 'SMB' not in result

    @pytest.mark.asyncio
    async def test_filter_exclude(self, sand_svc):
        agent = MagicMock()
        agent.proxy_receivers = {'HTTP': ['a1'], 'SMB': ['a2']}
        sand_svc.data_svc.locate = AsyncMock(return_value=[agent])
        result = json.loads(await sand_svc._get_available_proxy_peer_info({'HTTP'}, exclude=True))
        assert 'HTTP' not in result
        assert 'SMB' in result

    @pytest.mark.asyncio
    async def test_empty_specified_includes_all(self, sand_svc):
        agent = MagicMock()
        agent.proxy_receivers = {'HTTP': ['a1'], 'SMB': ['a2']}
        sand_svc.data_svc.locate = AsyncMock(return_value=[agent])
        result = json.loads(await sand_svc._get_available_proxy_peer_info(set()))
        assert 'HTTP' in result
        assert 'SMB' in result


# ========================================================================
# _get_encoded_proxy_peer_info
# ========================================================================

class TestGetEncodedProxyPeerInfo:
    @pytest.mark.asyncio
    async def test_all_filter(self, sand_svc):
        agent = MagicMock(); agent.proxy_receivers = {'HTTP': ['a1']}
        sand_svc.data_svc.locate = AsyncMock(return_value=[agent])
        encoded, key = await sand_svc._get_encoded_proxy_peer_info('all')
        assert encoded  # non-empty
        assert key  # non-empty
        # Verify XOR decode round-trip
        decoded_bytes = base64.b64decode(encoded)
        decoded = ''.join(chr(b ^ ord(key[i % len(key)])) for i, b in enumerate(decoded_bytes))
        data = json.loads(decoded)
        assert 'HTTP' in data

    @pytest.mark.asyncio
    async def test_include_filter(self, sand_svc):
        agent = MagicMock(); agent.proxy_receivers = {'HTTP': ['a1'], 'SMB': ['a2']}
        sand_svc.data_svc.locate = AsyncMock(return_value=[agent])
        encoded, key = await sand_svc._get_encoded_proxy_peer_info('HTTP')
        decoded_bytes = base64.b64decode(encoded)
        decoded = ''.join(chr(b ^ ord(key[i % len(key)])) for i, b in enumerate(decoded_bytes))
        data = json.loads(decoded)
        assert 'HTTP' in data
        assert 'SMB' not in data

    @pytest.mark.asyncio
    async def test_exclude_filter(self, sand_svc):
        agent = MagicMock(); agent.proxy_receivers = {'HTTP': ['a1'], 'SMB': ['a2']}
        sand_svc.data_svc.locate = AsyncMock(return_value=[agent])
        encoded, key = await sand_svc._get_encoded_proxy_peer_info('!HTTP')
        decoded_bytes = base64.b64decode(encoded)
        decoded = ''.join(chr(b ^ ord(key[i % len(key)])) for i, b in enumerate(decoded_bytes))
        data = json.loads(decoded)
        assert 'HTTP' not in data
        assert 'SMB' in data

    @pytest.mark.asyncio
    async def test_empty_receiver_info_returns_empty(self, sand_svc):
        sand_svc.data_svc.locate = AsyncMock(return_value=[])
        # _get_available_proxy_peer_info returns '{}' which is truthy
        # so we still get encoded output, but let's verify it decodes
        encoded, key = await sand_svc._get_encoded_proxy_peer_info('all')
        assert encoded
        decoded_bytes = base64.b64decode(encoded)
        decoded = ''.join(chr(b ^ ord(key[i % len(key)])) for i, b in enumerate(decoded_bytes))
        assert json.loads(decoded) == {}

    @pytest.mark.asyncio
    async def test_comma_separated_include(self, sand_svc):
        agent = MagicMock(); agent.proxy_receivers = {'HTTP': ['a1'], 'SMB': ['a2'], 'DNS': ['a3']}
        sand_svc.data_svc.locate = AsyncMock(return_value=[agent])
        encoded, key = await sand_svc._get_encoded_proxy_peer_info('HTTP,SMB')
        decoded_bytes = base64.b64decode(encoded)
        decoded = ''.join(chr(b ^ ord(key[i % len(key)])) for i, b in enumerate(decoded_bytes))
        data = json.loads(decoded)
        assert 'HTTP' in data
        assert 'SMB' in data
        assert 'DNS' not in data


# ========================================================================
# _install_gocat_extensions / _uninstall_gocat_extensions
# ========================================================================

class TestInstallUninstallExtensions:
    @pytest.mark.asyncio
    @patch('app.sand_svc.which', return_value='/usr/bin/go')
    async def test_install_calls_attempt_copy(self, mock_which, sand_svc):
        sand_svc._attempt_module_copy = AsyncMock(return_value=True)
        result = await sand_svc._install_gocat_extensions(['ext1', 'ext2'], headers={})
        assert result == ['ext1', 'ext2']

    @pytest.mark.asyncio
    @patch('app.sand_svc.which', return_value=None)
    async def test_install_without_go_returns_empty(self, mock_which, sand_svc):
        result = await sand_svc._install_gocat_extensions(['ext1'], headers={})
        assert result == []

    @pytest.mark.asyncio
    @patch('app.sand_svc.which', return_value='/usr/bin/go')
    async def test_install_empty_list_returns_empty(self, mock_which, sand_svc):
        result = await sand_svc._install_gocat_extensions([], headers={})
        assert result == []

    @pytest.mark.asyncio
    @patch('app.sand_svc.which', return_value='/usr/bin/go')
    async def test_install_none_returns_empty(self, mock_which, sand_svc):
        result = await sand_svc._install_gocat_extensions(None, headers={})
        assert result == []

    @pytest.mark.asyncio
    @patch('app.sand_svc.which', return_value='/usr/bin/go')
    async def test_install_failed_copy_excluded(self, mock_which, sand_svc):
        sand_svc._attempt_module_copy = AsyncMock(side_effect=[True, False])
        result = await sand_svc._install_gocat_extensions(['ext1', 'ext2'], headers={})
        assert result == ['ext1']

    @pytest.mark.asyncio
    @patch('app.sand_svc.which', return_value='/usr/bin/go')
    async def test_uninstall_calls_remove(self, mock_which, sand_svc):
        mod = MagicMock()
        sand_svc.sandcat_extensions = {'ext1': mod}
        await sand_svc._uninstall_gocat_extensions(['ext1'])
        mod.remove_module_files.assert_called_once()

    @pytest.mark.asyncio
    @patch('app.sand_svc.which', return_value=None)
    async def test_uninstall_without_go_does_nothing(self, mock_which, sand_svc):
        mod = MagicMock()
        sand_svc.sandcat_extensions = {'ext1': mod}
        await sand_svc._uninstall_gocat_extensions(['ext1'])
        mod.remove_module_files.assert_not_called()

    @pytest.mark.asyncio
    @patch('app.sand_svc.which', return_value='/usr/bin/go')
    async def test_uninstall_empty_does_nothing(self, mock_which, sand_svc):
        await sand_svc._uninstall_gocat_extensions([])
        # no error raised


# ========================================================================
# _attempt_module_copy
# ========================================================================

class TestAttemptModuleCopy:
    @pytest.mark.asyncio
    async def test_successful_copy(self, sand_svc):
        mod = MagicMock()
        mod.copy_module_files = AsyncMock(return_value=True)
        sand_svc.sandcat_extensions = {'ext1': mod}
        result = await sand_svc._attempt_module_copy('ext1', headers={})
        assert result is True

    @pytest.mark.asyncio
    async def test_module_not_found(self, sand_svc):
        sand_svc.sandcat_extensions = {}
        result = await sand_svc._attempt_module_copy('nonexistent', headers={})
        assert result is False

    @pytest.mark.asyncio
    async def test_copy_raises_exception(self, sand_svc):
        mod = MagicMock()
        mod.copy_module_files = AsyncMock(side_effect=Exception('copy error'))
        sand_svc.sandcat_extensions = {'ext1': mod}
        result = await sand_svc._attempt_module_copy('ext1', headers={})
        assert result is False


# ========================================================================
# load_sandcat_extension_modules
# ========================================================================

class TestLoadSandcatExtensionModules:
    @pytest.mark.asyncio
    async def test_loads_modules_from_directory(self, sand_svc):
        mock_module = MagicMock()
        mock_module.check_go_dependencies.return_value = True
        sand_svc._load_extension_module = AsyncMock(return_value=mock_module)

        with patch('os.walk') as mock_walk:
            mock_walk.return_value = [
                ('/ext', [], ['gist.py', 'slack.py'])
            ]
            await sand_svc.load_sandcat_extension_modules()

        assert 'gist' in sand_svc.sandcat_extensions
        assert 'slack' in sand_svc.sandcat_extensions

    @pytest.mark.asyncio
    async def test_skips_hidden_files(self, sand_svc):
        mock_module = MagicMock()
        mock_module.check_go_dependencies.return_value = True
        sand_svc._load_extension_module = AsyncMock(return_value=mock_module)

        with patch('os.walk') as mock_walk:
            mock_walk.return_value = [
                ('/ext', [], ['.hidden.py', '__init__.py', 'gist.py'])
            ]
            await sand_svc.load_sandcat_extension_modules()

        assert 'gist' in sand_svc.sandcat_extensions
        assert len(sand_svc.sandcat_extensions) == 1

    @pytest.mark.asyncio
    async def test_skips_hidden_dirs(self, sand_svc):
        mock_module = MagicMock()
        mock_module.check_go_dependencies.return_value = True
        sand_svc._load_extension_module = AsyncMock(return_value=mock_module)

        dirs_list = ['.git', '__pycache__', 'valid']
        with patch('os.walk') as mock_walk:
            mock_walk.return_value = [
                ('/ext', dirs_list, ['gist.py'])
            ]
            await sand_svc.load_sandcat_extension_modules()
            # dirs_list should have been pruned
            assert '.git' not in dirs_list
            assert '__pycache__' not in dirs_list
            assert 'valid' in dirs_list

    @pytest.mark.asyncio
    async def test_skips_module_with_failed_deps(self, sand_svc):
        mock_module = MagicMock()
        mock_module.check_go_dependencies.return_value = False
        mock_module.install_dependencies.return_value = False
        sand_svc._load_extension_module = AsyncMock(return_value=mock_module)

        with patch('os.walk') as mock_walk:
            mock_walk.return_value = [('/ext', [], ['gist.py'])]
            await sand_svc.load_sandcat_extension_modules()

        assert len(sand_svc.sandcat_extensions) == 0

    @pytest.mark.asyncio
    async def test_installs_deps_if_check_fails(self, sand_svc):
        mock_module = MagicMock()
        mock_module.check_go_dependencies.return_value = False
        mock_module.install_dependencies.return_value = True
        sand_svc._load_extension_module = AsyncMock(return_value=mock_module)

        with patch('os.walk') as mock_walk:
            mock_walk.return_value = [('/ext', [], ['gist.py'])]
            await sand_svc.load_sandcat_extension_modules()

        assert 'gist' in sand_svc.sandcat_extensions

    @pytest.mark.asyncio
    async def test_none_module_skipped(self, sand_svc):
        sand_svc._load_extension_module = AsyncMock(return_value=None)
        with patch('os.walk') as mock_walk:
            mock_walk.return_value = [('/ext', [], ['bad.py'])]
            await sand_svc.load_sandcat_extension_modules()
        assert len(sand_svc.sandcat_extensions) == 0


# ========================================================================
# _load_extension_module
# ========================================================================

class TestLoadExtensionModule:
    @pytest.mark.asyncio
    async def test_successful_load(self, sand_svc):
        mock_ext = MagicMock()
        with patch('app.sand_svc.import_module') as mock_import:
            mock_mod = MagicMock()
            mock_mod.load.return_value = mock_ext
            mock_import.return_value = mock_mod
            result = await sand_svc._load_extension_module('/some/root', 'gist.py')
        assert result is mock_ext

    @pytest.mark.asyncio
    async def test_failed_load_returns_none(self, sand_svc):
        with patch('app.sand_svc.import_module', side_effect=ImportError('nope')):
            result = await sand_svc._load_extension_module('/some/root', 'bad.py')
        assert result is None
