import re

from app.utility.base_world import BaseWorld
from plugins.sandcat.app.utility.base_extension import Extension

GOCAT_PLUGIN = 'gocat'
PACKAGE_NAME = 'contact'
FILE_NAME = 'dns_tunneling.go'
DOMAIN_CONFIG = 'app.contact.dns.domain'
TEXT_TO_REPLACE = r'BASE_DOMAIN = "mycaldera.caldera"'
REPLACEMENT_TEMPLATE = 'BASE_DOMAIN = "{}"'


def load():
    return DnsTunneling()


class DnsTunneling(Extension):
    def __init__(self):
        super().__init__([(FILE_NAME, PACKAGE_NAME)],
                         dependencies=['github.com/miekg/dns'],
                         file_hooks={FILE_NAME: self.hook_set_custom_domain})

    async def hook_set_custom_domain(self, original_data):
        """Will replace the C2 domain variable with the domain in the C2 configuration."""
        domain_name = BaseWorld.get_config(prop=DOMAIN_CONFIG)
        if domain_name:
            replacement_text = REPLACEMENT_TEMPLATE.format(domain_name)
            new_data = re.sub(TEXT_TO_REPLACE, replacement_text, original_data, count=1)
            return new_data
        else:
            raise Exception('No DNS tunneling domain specified in C2 configuration file under app.contact.dns.domain')
