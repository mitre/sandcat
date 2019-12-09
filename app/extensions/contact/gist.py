from plugins.sandcat.app.utility.base_extension import Extension


def load():
    return GIST()


class GIST(Extension):

    def __init__(self):
        self.dependencies = ['github.com/google/go-github/github', 'golang.org/x/oauth2']
        super().__init__(file='gist.go', package='contact')
