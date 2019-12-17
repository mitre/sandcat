import os

from abc import ABC, abstractmethod


class Extension(ABC):

    @abstractmethod
    def __init__(self, file, package):
        self.file = file
        self.package = package
        self.dependencies = []

    def check_go_dependencies(self):
        for d in self.dependencies:
            if os.system('go list "{}"'.format(d)) != 0:
                return False
        return True
