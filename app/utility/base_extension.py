import os

from abc import ABC, abstractmethod


class Extension(ABC):

    @abstractmethod
    def __init__(self, file, package):
        self.file = file
        self.package = package
        self.go_src_path = self._check_go_src_path()
        self.dependencies = []

    def check_go_dependencies(self):
        if self.go_src_path:
            for d in self.dependencies:
                if not os.path.exists(os.path.join(self.go_src_path, d)):
                    return False
            return True
        return False

    """ PRIVATE """

    @staticmethod
    def _check_go_src_path():
        try:
            return os.path.join(os.environ['GOPATH'], 'src')
        except Exception:
            return None
