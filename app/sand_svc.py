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
                if platform == 'windows' and which('X86_64-w64-mingw32-gcc'):
                    await self._compile_new_agent(platform=platform,
                                                  headers=headers,
                                                  compile_target_name=name.split('.')[0] + '_' + platform + '.go',
                                                  output_name=name,
                                                  extension='-lib',
                                                  buildmode='--buildmode=c-shared',
                                                  extldflags='-extldflags "-Wl,--nxcompat -Wl,--dynamicbase -Wl,--high-entropy-va"',
                                                  cflags='GOARCH=amd64 CGO_ENABLED=1 CC=X86_64-w64-mingw32-gcc')
            else:
                await self._compile_new_agent(platform=platform,
                                              headers=headers,
                                              compile_target_name=name,
                                              output_name=name)
            if name == 'shared.go':
                return '%s-%s-lib' % (name, platform)
        return '%s-%s' % (name, platform)

    """ PRIVATE """

    @staticmethod
    def _generate_key(size=30):
        return ''.join(random.choice(string.ascii_uppercase + string.digits) for _ in range(size))

    async def _compile_new_agent(self, platform, headers, compile_target_name, output_name, buildmode='', extension='',
                                 extldflags='', cflags=''):
        plugin, file_path = await self.file_svc.find_file_path(compile_target_name)
        ldflags = ['-s', '-w', '-X main.key=%s' % (self._generate_key(),)]
        for param in ('defaultServer', 'defaultGroup', 'defaultSleep'):
            if param in headers:
                ldflags.append('-X main.%s=%s' % (param, headers[param]))
        output = 'plugins/%s/payloads/%s-%s%s' % (plugin, output_name, platform, extension)
        ldflags.append(extldflags)
        self.file_svc.log.debug('Dynamically compiling %s' % compile_target_name)
        await self.file_svc.compile_go(platform, output, file_path, buildmode=buildmode,
                                       ldflags=' '.join(ldflags), cflags=cflags)