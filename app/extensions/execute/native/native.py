from plugins.sandcat.app.utility.base_extension import Extension


def load():
    return NativeExecutor()


class NativeExecutor(Extension):

    def __init__(self):
        super().__init__([
            ('native.go', 'execute/native'),
            ('ls.go', 'execute/native/discovery'),
        ])
        self.dependencies = []
