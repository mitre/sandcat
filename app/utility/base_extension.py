import os

from abc import ABC, abstractmethod


class Extension(ABC):

    @abstractmethod
    def __init__(self, file, package):
        self.file = file
        self.package = package
        self.go_src_path = os.path.join(os.environ['GOPATH'], 'src')

    @abstractmethod
    def check_go_dependencies(self):
        pass
