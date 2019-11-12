import random
import string

from app.utility.base_service import BaseService
from shutil import which


class SandService(BaseService):

    def __init__(self, services):
        self.file_svc = services.get('file_svc')
        self.data_svc = services.get('data_svc')
        self.c2_svc = services.get('c2_svc')

    async def dynamically_compile(self, headers):
        name, platform = headers.get('file'), headers.get('platform')
        if which('go') is not None:
            plugin, file_path = await self.file_svc.find_file_path(name)
            ldflags = ['-s', '-w', '-X main.key=%s' % (self._generate_key(),)]
            for param in ('defaultServer', 'defaultGroup', 'defaultSleep', 'defaultC2', 'c2'):
                if param in headers:
                    if param == 'c2':
                        ldflags.append('-X main.%s=%s' % (await self._get_c2_requirement(headers[param])))
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

    async def _get_c2_requirement(self, type):
        var = 'githubToken'
        if type.lower() == 'gist':
            c2 = await self.data_svc.locate('c2', dict(enabled=True, c2_type='active'))
            if len(c2):
                c2_module = await self.load_module(module_type=c2[0].name, module_info=dict(module=c2[0].module,
                                                                                            config=c2[0].config,
                                                                                            c2_type=c2[0].c2_type))
                return var, c2_module.encode_config_info()
        return var, ''


