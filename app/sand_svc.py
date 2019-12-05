import os
import random
import string

from shutil import copyfile, which
from app.utility.base_service import BaseService


class SandService(BaseService):

    def __init__(self, services):
        self.file_svc = services.get('file_svc')
        self.data_svc = services.get('data_svc')

    async def dynamically_compile_executable(self, headers):
        name, platform = headers.get('file'), headers.get('platform')
        if which('go') is not None:
            await self._compile_new_agent(platform=platform,
                                          headers=headers,
                                          compile_target_name=name,
                                          output_name=name)
        return '%s-%s' % (name, platform)

    async def dynamically_compile_library(self, headers):
        name, platform = headers.get('file'), headers.get('platform')
        if which('go') is not None and which('X86_64-w64-mingw32-gcc') is not None and platform == 'windows':
            await self._compile_new_agent(platform=platform,
                                          headers=headers,
                                          compile_target_name=name.split('.')[0] + '_' + platform + '.go',
                                          output_name=name,
                                          buildmode='--buildmode=c-shared',
                                          extldflags='-extldflags "-Wl,--nxcompat -Wl,--dynamicbase -Wl,'
                                                     '--high-entropy-va"',
                                          cflags='GOARCH=amd64 CGO_ENABLED=1 CC=X86_64-w64-mingw32-gcc',
                                          flag_params=('defaultServer', 'defaultGroup', 'defaultSleep',
                                                       'defaultExeName')),
        return '%s-%s' % (name, platform)

    async def install_gocat_extensions(self):
        if which('go') is not None:
            if self._check_gist_go_dependencies():
                self._copy_file_to_sandcat(file='gist.go', pkg='contact')

    """ PRIVATE """

    @staticmethod
    def _generate_key(size=30):
        return ''.join(random.choice(string.ascii_uppercase + string.digits) for _ in range(size))

    async def _get_c2_config(self, c2_type):
        c2 = await self.data_svc.locate('c2', dict(name=c2_type))
        if len(c2):
            return c2[0].get_config()
        return '', ''

    async def _compile_new_agent(self, platform, headers, compile_target_name, output_name, buildmode='',
                                 extldflags='', cflags='',
                                 flag_params=('defaultServer', 'defaultGroup', 'defaultSleep')):
        plugin, file_path = await self.file_svc.find_file_path(compile_target_name)
        ldflags = ['-s', '-w', '-X main.key=%s' % (self._generate_key(),)]
        for param in flag_params:
            if param in headers:
                ldflags.append('-X main.%s=%s' % (param, headers[param]))
        output = 'plugins/%s/payloads/%s-%s' % (plugin, output_name, platform)
        ldflags.append(extldflags)
        self.file_svc.log.debug('Dynamically compiling %s' % compile_target_name)
        await self.file_svc.compile_go(platform, output, file_path, buildmode=buildmode,
                                       ldflags=' '.join(ldflags), cflags=cflags)

    @staticmethod
    def _check_gist_go_dependencies():
        go_path = os.path.join(os.environ['GOPATH'], 'src')
        return os.path.exists(os.path.join(go_path, 'github.com/google/go-github/github')) and \
            os.path.exists(os.path.join(go_path, 'golang.org/x/oauth2'))

    @staticmethod
    def _copy_file_to_sandcat(file, pkg):
        base = os.path.abspath(os.path.join('plugins', 'sandcat'))
        try:
            copyfile(os.path.join(base, 'gocat-extensions', pkg, file), os.path.join(base, 'gocat', pkg, file))
        except Exception:
            pass
