import os
import random
import string

from shutil import which, copyfile
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
                        for k, v in (await self._get_c2_config(headers[param])).items():
                            ldflags.append('-X main.c2%s=%s' % (k, v))
                    else:
                        ldflags.append('-X main.%s=%s' % (param, headers[param]))

            output = 'plugins/%s/payloads/%s-%s' % (plugin, name, platform)
            self.file_svc.log.debug('Dynamically compiling %s' % name)
            await self.file_svc.compile_go(platform, output, file_path, ldflags=' '.join(ldflags))
        return '%s-%s' % (name, platform)

    async def install_gocat_extensions(self):
        if which('go') is not None:
            if self._check_gist_go_dependencies():
                self._copy_file_to_sandcat(file='gist.go', pkg='contact')

    """ PRIVATE """

    @staticmethod
    def _check_gist_go_dependencies():
        go_path = os.path.join(os.environ['GOPATH'], 'src')
        return os.path.exists(os.path.join(go_path, 'github.com/google/go-github/github')) and \
            os.path.exists(os.path.join(go_path, 'golang.org/x/oauth2'))

    @staticmethod
    def _copy_file_to_sandcat(file, pkg):
        base = os.path.abspath(os.path.join('plugins', 'sandcat'))
        copyfile(os.path.join(base, 'gocat-extensions', pkg, file), os.path.join(base, 'gocat', pkg, file))

    @staticmethod
    def _generate_key(size=30):
        return ''.join(random.choice(string.ascii_uppercase + string.digits) for _ in range(size))

    async def _get_c2_config(self, c2_type):
        c2 = await self.data_svc.locate('c2', dict(name=c2_type))
        if len(c2):
            return c2[0].get_config()
        return dict(c2Name='', c2Key='')
