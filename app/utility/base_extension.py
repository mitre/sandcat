import os
import subprocess

from abc import ABC, abstractmethod
from shutil import copyfile


class Extension(ABC):

    @abstractmethod
    def __init__(self, files):
        """files is list of 2-tuples of the form (filename, package)"""
        self.files = files
        self.dependencies = []

    def check_go_dependencies(self):
        for d in self.dependencies:
            dep_result = subprocess.run('go list "{}"'.format(d), shell=True, stdout=subprocess.PIPE)
            if (dep_result.stdout.decode()).strip() != d:
                return False
        return True

    def copy_module_files(self, base_dir):
        for file, pkg in self.files:
            # Make sure the package folders are there or are created.
            package_path = os.path.join(base_dir, 'gocat', pkg)
            if not os.path.exists(package_path):
                os.makedirs(package_path)

            copyfile(src=os.path.join(base_dir, 'gocat-extensions', pkg, file),
                     dst=os.path.join(base_dir, 'gocat', pkg, file))

    def remove_module_files(self, base_dir):
        for file, pkg in self.files:
            file_path = os.path.join(base_dir, 'gocat', pkg, file)
            if os.path.exists(file_path):
                os.remove(file_path)

    def install_dependencies(self):
        # TODO: attempt to install all dependencies if they aren't fulfilled
        pass
