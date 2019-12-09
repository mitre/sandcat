import os

from abc import ABC, abstractmethod


class Extension(ABC):

    @abstractmethod
    def __init__(self, file, package):
        self.file = file
        self.package = package
        try:
            self.go_src_path = os.path.join(os.environ['GOPATH'], 'src')
        except Exception:
            pass

    @abstractmethod
    def check_go_dependencies(self):
        pass
