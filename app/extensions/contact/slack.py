from plugins.sandcat.app.utility.base_extension import Extension


def load():
    return SLACK()


class SLACK(Extension):

    def __init__(self):
        super().__init__([('slack.go', 'contact')])
        self.dependencies = ['github.com/google/go-github/github', 'golang.org/x/oauth2']
