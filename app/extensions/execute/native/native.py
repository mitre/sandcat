from plugins.sandcat.app.utility.base_extension import Extension


def load():
    return Native()


class Native(Extension):

    def __init__(self):
        super().__init__([
            ('native.go', 'execute/native'),
            ('ip_addr.go', 'execute/native')
        ])
        self.dependencies = []
