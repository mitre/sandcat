from plugins.sandcat.app.sand_api import SandApi

name = 'Sandcat'
description = 'A custom multi-platform RAT'
address = None


async def initialize(app, services):
    cat_api = SandApi(services=services)
    app.router.add_route('*', '/sand/download', cat_api.download)
    app.router.add_route('POST', '/sand/beacon', cat_api.beacon)
    app.router.add_route('POST', '/sand/results', cat_api.results)

