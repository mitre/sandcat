import random
import string

from shutil import which
from app.utility.base_service import BaseService


class SandService(BaseService):

    def __init__(self, services):
        self.file_svc = services.get('file_svc')
        self.data_svc = services.get('data_svc')
        self.c2_svc = services.get('c2_svc')

    async def dynamically_compile(self, headers):
        name, platform, c2 = headers.get('file'), headers.get('platform'), headers.get('c2', 'http')
        if c2 is not 'http':
            c2_list = await self.data_svc.locate('c2', dict(name=c2))
            c2_module = await self.load_module(module_type=c2_list[0].name, module_info=dict(module=c2_list[0].module,
                                                config=c2_list[0].config, c2_type=c2_list[0].c2_type))
            self.c2_svc.start_channel(c2_module)
            # TODO Encode C2 config data into the sandcat code before compilation

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
