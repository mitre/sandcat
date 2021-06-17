from plugins.sandcat.app.utility.base_extension import Extension


def load():
    return FTP()


class FTP(Extension):

    def __init__(self):
        super().__init__([('ftp.go', 'contact')])
        self.dependencies = ['']
