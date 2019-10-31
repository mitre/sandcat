import os
import random
import string

from shutil import which


class SandService:

    def __init__(self, file_svc):
        self.file_svc = file_svc

    async def dynamically_compile(self, headers):
        name, platform = headers.get('file'), headers.get('platform')
        if which('go') is not None:
            if name == 'shared.go':
                await self._compile_new_agent(platform, headers,
                                              compile_target_name=name.split('.')[0] + '_' + platform + '.go',
                                              output_name=name,
                                              extension='-lib',
                                              extflags='-extldflags "-Wl,--nxcompat"',
                                              custom_params='GOARCH=amd64 CGO_ENABLED=1 CC=X86_64-w64-mingw32-gcc')
            else:
                await self._compile_new_agent(platform, headers,
                                              compile_target_name=name,
                                              output_name=name)
        if name == 'shared.go':
            return '%s-%s-lib' % (name, platform)
        return '%s-%s' % (name, platform)

    """ PRIVATE """

    @staticmethod
    def _generate_key(size=30):
        return ''.join(random.choice(string.ascii_uppercase + string.digits) for _ in range(size))

    async def _compile_new_agent(self, platform, headers, compile_target_name, output_name, extension='', extflags='', custom_params=''):
        plugin, file_path = await self.file_svc.find_file_path(compile_target_name)
        path = os.path.dirname(os.path.abspath(file_path))
        ldflags = ['-s', '-w', '-X _%s/core.Key=%s' % (path, self._generate_key(),)]
        for param in ('DefaultServer', 'DefaultGroup', 'DefaultSleep'):
            if param in headers:
                ldflags.append('-X _%s/core.%s=%s' % (path, param, headers[param]))
        output = 'plugins/%s/payloads/%s-%s%s' % (plugin, output_name, platform, extension)
        ldflags.append(extflags)
        self.file_svc.log.debug('Dynamically compiling %s' % compile_target_name)
        await self.file_svc.compile_go(platform, output, file_path,
                                       ldflags=' '.join(ldflags), custom_params=custom_params)