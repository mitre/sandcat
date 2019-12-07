import os

from plugins.sandcat.app.utility.base_extension import Extension


def load():
    return GIST()


class GIST(Extension):

    def __init__(self):
        super().__init__(file='gist.go', package='contact')

    def check_go_dependencies(self):
        return os.path.exists(os.path.join(self.go_path, 'github.com/google/go-github/github')) and \
            os.path.exists(os.path.join(self.go_path, 'golang.org/x/oauth2'))
