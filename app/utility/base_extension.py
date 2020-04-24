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
            src_package_path = os.path.join(base_dir, 'gocat-extensions', pkg)
            dest_package_path = os.path.join(base_dir, 'gocat', pkg)
            if not os.path.exists(dest_package_path):
                os.makedirs(dest_package_path)

            # Check if entire package is to be copied
            if file == '*':
                for dir_item in os.listdir(src_package_path):
                    src_path = os.path.join(src_package_path, dir_item)
                    if os.path.isfile(src_path):
                        copyfile(src=src_path,
                                 dst=os.path.join(dest_package_path, dir_item))
            else:
                copyfile(src=os.path.join(src_package_path, file),
                         dst=os.path.join(dest_package_path, file))

    def remove_module_files(self, base_dir):
        for file, pkg in self.files:
            package_path = os.path.join(base_dir, 'gocat', pkg)
            # Check if entire package is to be deleted
            if file == '*':
                for dir_item in os.listdir(package_path):
                    full_path = os.path.join(package_path, dir_item)
                    if os.path.isfile(full_path) and dir_item != 'load.go':
                        os.remove(full_path)
            else:
                file_path = os.path.join(package_path, file)
                if os.path.exists(file_path):
                    os.remove(file_path)

    def install_dependencies(self):
        # Attempt to install all dependencies if they aren't fulfilled
        pass
