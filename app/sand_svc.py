import hashlib
import os
import random
import string
from importlib import import_module
from shutil import copyfile, which

from app.utility.base_service import BaseService


class SandService(BaseService):

    def __init__(self, services, popular_process_names):
        self.file_svc = services.get('file_svc')
        self.data_svc = services.get('data_svc')
        self.contact_svc = services.get('contact_svc')
        self.log = self.create_logger('sand_svc')
        self.sandcat_dir = os.path.relpath(os.path.join('plugins', 'sandcat'))
        self.popular_process_names = popular_process_names

    async def dynamically_compile_executable(self, headers):
        name, platform = headers.get('file'), headers.get('platform')
        if which('go') is not None:
            await self._compile_new_agent(platform=platform,
                                          headers=headers,
                                          compile_target_name=name,
                                          output_name=name)
        _, path = await self.file_svc.find_file_path('sandcat.go-%s' % platform)
        signature = hashlib.md5(open(path, 'rb').read()).hexdigest()
        display_name = random.choice(self.popular_process_names.get(platform))
        self.log.debug('sandcat downloaded with hash=%s and name=%s' % (signature, display_name))
        return '%s-%s' % (name, platform), display_name

    async def dynamically_compile_library(self, headers):
        name, platform = headers.get('file'), headers.get('platform')
        compile_options = dict(
            windows=dict(
                CC='x86_64-w64-mingw32-gcc',
                cflags='CGO_ENABLED=1',
                extldflags='-extldflags "-Wl,--nxcompat -Wl,--dynamicbase -Wl,--high-entropy-va"',
            ),
            linux=dict(
                cflags='CGO_ENABLED=1'
            )
        )
        if which('go') is not None:
            if platform in compile_options.keys():
                if 'CC' in compile_options[platform].keys() and which(compile_options[platform]['CC']) is not None:
                    compile_options[platform]['cflags'] += ' CC=%s' % compile_options[platform]['CC']
                    # key is deleted from compile_options to use dict as kwargs for called function.
                    del compile_options[platform]['CC']
                await self._compile_new_agent(platform=platform,
                                              headers=headers,
                                              compile_target_name='shared.go',
                                              output_name=name,
                                              buildmode='--buildmode=c-shared',
                                              **compile_options[platform],
                                              flag_params=('defaultServer', 'defaultGroup', 'defaultSleep', 'c2')),
        return '%s-%s' % (name, platform), self.generate_name()

    async def install_gocat_extensions(self):
        if which('go') is not None:
            for module in self._find_available_extension_modules():
                if module.check_go_dependencies():
                    self._copy_file_to_sandcat(file=module.file, pkg=module.package)

    """ PRIVATE """

    @staticmethod
    def _generate_key(size=30):
        return ''.join(random.choice(string.ascii_uppercase + string.digits) for _ in range(size))

    async def _get_c2_config(self, c2_type):
        for c2 in self.contact_svc.contacts:
            if c2_type == c2.name:
                return c2.get_config()
        return '', ''

    async def _compile_new_agent(self, platform, headers, compile_target_name, output_name, buildmode='',
                                 extldflags='', cflags='', flag_params=('server', 'c2', 'apiBeacon', 'apiResult')):
        plugin, file_path = await self.file_svc.find_file_path(compile_target_name)
        ldflags = ['-s', '-w', '-X main.key=%s' % (self._generate_key(),)]
        for param in flag_params:
            if param in headers:
                if param == 'c2':
                    ldflags.append('-X main.%s=%s' % (await self._get_c2_config(headers[param])))
                else:
                    ldflags.append('-X main.%s=%s' % (param, headers[param]))
        output = 'plugins/%s/payloads/%s-%s' % (plugin, output_name, platform)
        ldflags.append(extldflags)
        self.file_svc.log.debug('Dynamically compiling %s' % compile_target_name)
        await self.file_svc.compile_go(platform, output, file_path, buildmode=buildmode,
                                       ldflags=' '.join(ldflags), cflags=cflags)

    def _copy_file_to_sandcat(self, file, pkg):
        try:
            copyfile(src=os.path.join(self.sandcat_dir, 'gocat-extensions', pkg, file),
                     dst=os.path.join(self.sandcat_dir, 'gocat', pkg, file))
        except Exception as e:
            self.log.error('Error copying file %s, %s' % (file, e))

    def _find_available_extension_modules(self):
        extensions = []
        for root, dirs, files in os.walk(os.path.join(self.sandcat_dir, 'app', 'extensions')):
            files = [f for f in files if not f[0] == '.' and not f[0] == "_"]
            dirs[:] = [d for d in dirs if not d[0] == '.' and not d[0] == "_"]
            for file in files:
                extensions.append(self._load_extension_module(root, file))
        return extensions

    def _load_extension_module(self, root, file):
        module = os.path.join(root, file.split('.')[0]).replace(os.path.sep, '.')
        try:
            return getattr(import_module(module), 'load')()
        except Exception as e:
            self.log.error('Error loading extension=%s, %s' % (module, e))
