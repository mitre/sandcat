import os
import random
import string
from importlib import import_module
from shutil import copyfile, which

from app.utility.base_service import BaseService


class SandService(BaseService):

    def __init__(self, services):
        self.file_svc = services.get('file_svc')
        self.data_svc = services.get('data_svc')
        self.contact_svc = services.get('contact_svc')
        self.app_svc = services.get('app_svc')
        self.log = self.create_logger('sand_svc')
        self.sandcat_dir = os.path.relpath(os.path.join('plugins', 'sandcat'))

    async def dynamically_compile_executable(self, headers):
        # HTTP headers will specify the file name, platform, and comma-separated list of extension modules to include.
        name, platform = headers.get('file'), headers.get('platform')
        extension_names = [ext_name for ext_name in headers.get('gocat-extensions', "").split(',') if ext_name]
        if which('go') is not None:
            await self._compile_new_agent(platform=platform,
                                          headers=headers,
                                          compile_target_name=name,
                                          output_name=name,
                                          extension_names=extension_names)
        return await self.app_svc.retrieve_compiled_file(name, platform)

    async def dynamically_compile_library(self, headers):
        # HTTP headers will specify the file name, platform, and comma-separated list of extension modules to include.
        name, platform = headers.get('file'), headers.get('platform')
        extension_names = [ext_name for ext_name in headers.get('gocat-extensions', "").split(',') if ext_name]
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
                                              flag_params=('server', 'c2', 'fetchPeers'),
                                              extension_names=extension_names)
        return '%s-%s' % (name, platform), self.generate_name()

    """ PRIVATE """

    @staticmethod
    def _generate_key(size=30):
        return ''.join(random.choice(string.ascii_uppercase + string.digits) for _ in range(size))

    async def _get_c2_config(self, c2_type):
        for c2 in self.contact_svc.contacts:
            if c2_type == c2.name:
                return c2.get_config()
        return '', ''

    async def _install_gocat_extensions(self, extension_names):
        """Given a list of extension names, will copy the required files for each extension from the gocat-extensions
        subdirectory into the gocat subdirectory."""

        installed_extensions = []
        if which('go') is not None and extension_names:
            self.log.debug('Installing gocat extension modules: %s' % ', '.join(extension_names))
            for extension_name in extension_names:
                module = self._fetch_extension_module(extension_name)
                if module:
                    if module.check_go_dependencies():
                        self.log.debug('Fetched extension module %s' % extension_name)
                        self._copy_module_files_to_sandcat(module)
                        installed_extensions.append(extension_name)
                    else:
                        self.log.error('Dependencies not satisfied for extension %s' % extension_name)
                else:
                    self.log.error("Failed to fetch extension %s" % extension_name)
        return installed_extensions

    async def _uninstall_gocat_extensions(self, extension_names):
        """Given a list of extension names, will remove the required files for each extension from the gocat
        subdirectory."""

        if which('go') is not None and extension_names:
            for extension_name in extension_names:
                module = self._fetch_extension_module(extension_name)
                if module:
                    self._remove_module_files_from_sandcat(module)

    async def _compile_new_agent(self, platform, headers, compile_target_name, output_name, buildmode='',
                                 extldflags='', cflags='', flag_params=('server', 'c2', 'fetchPeers'),
                                 extension_names=None):
        """Compile sandcat agent using specified parameters. Will also include any requested extension modules."""

        plugin, file_path = await self.file_svc.find_file_path(compile_target_name)
        ldflags = ['-s', '-w', '-X main.key=%s' % (self._generate_key(),)]
        for param in flag_params:
            if param in headers:
                if param == 'c2':
                    ldflags.append('-X main.%s=%s' % (await self._get_c2_config(headers[param])))
                elif param == 'fetchPeers' and headers.get(param, "").lower() == "true":
                    ldflags.append('-X main.%s=%s' % ('onlineHosts', ','.join([agent.host for agent in await self.data_svc.locate('agents')])))
                else:
                    ldflags.append('-X main.%s=%s' % (param, headers[param]))
        output = 'plugins/%s/payloads/%s-%s' % (plugin, output_name, platform)
        ldflags.append(extldflags)

        # Load extensions and compile.
        installed_extensions = await self._install_gocat_extensions(extension_names)
        self.file_svc.log.debug('Dynamically compiling %s' % compile_target_name)
        await self.file_svc.compile_go(platform, output, file_path, buildmode=buildmode, ldflags=' '.join(ldflags), cflags=cflags)

        # Remove extension files.
        if installed_extensions:
            self.log.debug('Cleaning up files for gocat extension modules %s' % ', '.join(installed_extensions))
            await self._uninstall_gocat_extensions(extension_names)

    def _copy_module_files_to_sandcat(self, module):
        """Given an extension module object, will copy the module-required files from the gocat-extension subdirectory
        into the gocat subdirectory in order to compile the extension module into sandcat."""

        if module:
            for file, pkg in module.files:
                try:
                    # Make sure the package folders are there or are created.
                    package_path = os.path.join(self.sandcat_dir, 'gocat', pkg)
                    if not os.path.exists(package_path):
                        os.makedirs(package_path)

                    copyfile(src=os.path.join(self.sandcat_dir, 'gocat-extensions', pkg, file),
                             dst=os.path.join(self.sandcat_dir, 'gocat', pkg, file))
                except Exception as e:
                    self.log.error('Error copying file %s, %s' % (file, e))

    def _remove_module_files_from_sandcat(self, module):
        """Given an extension module object, will delete the module-required files from the gocat subdirectory
        in order to provide a clean file structure for the next sandcat compilation."""

        if module:
            for file, pkg in module.files:
                try:
                    file_path = os.path.join(self.sandcat_dir, 'gocat', pkg, file)
                    if os.path.exists(file_path):
                        os.remove(file_path)
                except Exception as e:
                    self.log.error('Error copying file %s, %s' % (file, e))

    def _fetch_extension_module(self, extension_name):
        """Given an extension name, returns the extension module object for the extension, or None if not found."""

        extension = None
        for root, dirs, files in os.walk(os.path.join(self.sandcat_dir, 'app', 'extensions')):
            files = [f for f in files if not f[0] == '.' and not f[0] == "_"]
            dirs[:] = [d for d in dirs if not d[0] == '.' and not d[0] == "_"]
            for file in files:
                if file.lower() == extension_name.lower() + ".py":
                    extension = self._load_extension_module(root, file)
                    break
        return extension

    def _load_extension_module(self, root, file):
        """Give the file path and file name for the extension module file, will return the extension
        module object. Helper method for _fetch_extension_module."""

        module = os.path.join(root, file.split('.')[0]).replace(os.path.sep, '.')
        try:
            # Module's "load" method will return the extension module object.
            return getattr(import_module(module), 'load')()
        except Exception as e:
            self.log.error('Error loading extension=%s, %s' % (module, e))
