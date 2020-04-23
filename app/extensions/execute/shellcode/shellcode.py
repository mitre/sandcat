from plugins.sandcat.app.utility.base_extension import Extension


def load():
    return Shellcode()


class Shellcode(Extension):

    def __init__(self):
        super().__init__([
            ('shellcode.go', 'execute/shellcode'),
            ('shellcode_linux.go', 'execute/shellcode'),
            ('shellcode_windows.go', 'execute/shellcode'),
        ])
        self.dependencies = []
