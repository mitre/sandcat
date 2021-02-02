from plugins.sandcat.app.utility.base_extension import Extension


def load():
    return DnsTunneling()


class DnsTunneling(Extension):

    def __init__(self):
        super().__init__([('dns_tunneling.go', 'contact')])
        self.dependencies = ['github.com/miekg/dns']
