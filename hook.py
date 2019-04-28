from plugins.sandcat.app.sand_api import SandApi

name = 'Sandcat'
description = 'An in-memory agent/server combination'
address = '/plugin/sandcat/gui'
store = None


async def initialize(app, services):
    cat_api = SandApi(services=services)
    app.router.add_static('/sandcat', 'plugins/sandcat/static/', append_version=True)
    services.get('auth_svc').set_unauthorized_route('GET', '/plugin/sandcat/gui', cat_api.splash)
    services.get('auth_svc').set_unauthorized_route('POST', '/sand/results', cat_api.results)
    services.get('auth_svc').set_unauthorized_route('POST', '/sand/register', cat_api.registration)
    services.get('auth_svc').set_unauthorized_route('POST', '/sand/instructions', cat_api.instructions)
    services.get('auth_svc').set_unauthorized_route('POST', '/file/render', cat_api.render)
    services.get('auth_svc').set_unauthorized_route('POST', '/file/download', cat_api.download)

