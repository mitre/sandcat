import subprocess
from abc import ABC, abstractmethod


class Extension(ABC):

    @abstractmethod
    def __init__(self, file, package):
        self.file = file
        self.package = package
        self.dependencies = []

    def check_go_dependencies(self):
        for d in self.dependencies:
            dep_result = subprocess.run('go list "{}"'.format(d), shell=True, stdout=subprocess.PIPE)
            if (dep_result.stdout.decode()).strip() != d:
                return False
        return True
