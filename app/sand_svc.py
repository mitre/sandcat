import random
import string

from shutil import which


class SandService:

    def __init__(self, file_svc):
        self.file_svc = file_svc

    async def dynamically_compile(self, headers):
        name, platform = headers.get('file'), headers.get('platform')
        if which('go') is not None:
            plugin, file_path = await self.file_svc.find_file_path(name)

            ldflags = ['-s', '-w', '-X main.key=%s' % (self._generate_key(),)]
            for param in ('defaultServer', 'defaultGroup', 'defaultSleep'):
                if param in headers:
                    ldflags.append('-X main.%s=%s' % (param, headers[param]))

            output = 'plugins/%s/payloads/%s-%s' % (plugin, name, platform)
            self.file_svc.log.debug('Dynamically compiling %s' % name)
            await self.file_svc.compile_go(platform, output, file_path, ldflags=' '.join(ldflags))
        return '%s-%s' % (name, platform)

    """ PRIVATE """

    @staticmethod
    def _generate_key(size=30):
        return ''.join(random.choice(string.ascii_uppercase + string.digits) for _ in range(size))
