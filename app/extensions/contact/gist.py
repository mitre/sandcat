from plugins.sandcat.app.utility.base_extension import Extension


def load():
    return GIST()


class GIST(Extension):

    def __init__(self):
        super().__init__([('gist.go', 'contact')])
        self.dependencies = ['github.com/google/go-github/github', 'golang.org/x/oauth2']
