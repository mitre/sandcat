from plugins.sandcat.app.sand_api import SandApi

name = 'Sandcat'
description = 'A custom multi-platform RAT'
address = None


async def initialize(app, services):
    cat_api = SandApi(services=services)
    services.get('auth_svc').set_unauthorized_route('POST', '/sand/beacon', cat_api.beacon)
    services.get('auth_svc').set_unauthorized_route('POST', '/sand/results', cat_api.results)

