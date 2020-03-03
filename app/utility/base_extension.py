import subprocess
from abc import ABC, abstractmethod


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
