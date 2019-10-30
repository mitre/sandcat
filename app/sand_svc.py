import os
import random
import string

from shutil import which


class SandService:

    def __init__(self, file_svc):
        self.file_svc = file_svc

    async def dynamically_compile(self, headers):
        name, platform, shared = headers.get('file'), headers.get('platform'), headers.get('shared')

        if which('go') is not None:
            plugin, file_path = await self.file_svc.find_file_path(name)
            path = os.path.dirname(os.path.abspath(file_path))
            ldflags = ['-s', '-w', '-X _%s/core.Key=%s' % (path, self._generate_key(),)]
            for param in ('DefaultServer', 'DefaultGroup', 'DefaultSleep'):
                if param in headers:
                    ldflags.append('-X _%s/core.%s=%s' % (path, param, headers[param]))

            output = 'plugins/%s/payloads/%s-%s' % (plugin, name, platform)
            self.file_svc.log.debug('Dynamically compiling %s' % name)
            await self.file_svc.compile_go(platform, output, file_path, ldflags=' '.join(ldflags))
        return '%s-%s' % (name, platform)

    """ PRIVATE """

    @staticmethod
    def _generate_key(size=30):
        return ''.join(random.choice(string.ascii_uppercase + string.digits) for _ in range(size))
