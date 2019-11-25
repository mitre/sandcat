import random
import string

from shutil import which
from app.utility.base_service import BaseService


class SandService(BaseService):

    def __init__(self, services):
        self.file_svc = services.get('file_svc')
        self.data_svc = services.get('data_svc')

    async def dynamically_compile(self, headers):
        name, platform = headers.get('file'), headers.get('platform')
        if which('go') is not None:
            plugin, file_path = await self.file_svc.find_file_path(name)

            ldflags = ['-s', '-w', '-X main.key=%s' % (self._generate_key(),)]
            for param in ('defaultServer', 'defaultGroup', 'defaultSleep', 'defaultC2', 'c2'):
                if param in headers:
                    if param == 'c2':
                        ldflags.append('-X main.%s=%s' % (await self._get_c2_config(headers[param])))
                    else:
                        ldflags.append('-X main.%s=%s' % (param, headers[param]))

            output = 'plugins/%s/payloads/%s-%s' % (plugin, name, platform)
            self.file_svc.log.debug('Dynamically compiling %s' % name)
            await self.file_svc.compile_go(platform, output, file_path, ldflags=' '.join(ldflags))
        return '%s-%s' % (name, platform)

    """ PRIVATE """

    @staticmethod
    def _generate_key(size=30):
        return ''.join(random.choice(string.ascii_uppercase + string.digits) for _ in range(size))

    async def _get_c2_config(self, c2_type):
        c2 = await self.data_svc.locate('c2', dict(name=c2_type))
        if len(c2):
            return c2[0].get_config()
        return '', ''

    def _generate_name(self):
        """TODO: make random or get from config I guess. Right now output filename still includes platform. """
        config = self.file_svc.get_service('app_svc').config
        if 'sandcat_compile_name' in config:
            return config['sandcat_compile_name']
        return "notsandcat"