import json
import os
from datetime import datetime, timedelta


class SandService:

    def __init__(self, services):
        self.data_svc = services.get('data_svc')
        self.log = services.get('logger')
        self.utility_svc = services.get('utility_svc')
        self.plugins = services.get('plugins')

    async def registration(self, paw, executor, server, host, group):
        agent = await self.data_svc.dao.get('core_agent', dict(paw=paw))
        if agent:
            last_seen = datetime.strptime(agent[0]['last_seen'], '%Y-%m-%d %H:%M:%S.%f')
            if last_seen + timedelta(seconds=int(agent[0]['sleep'])) > datetime.now():
                self.log.debug('[AGENT] already active (%s)' % paw)
                status = True
            else:
                self.log.debug('[AGENT] stale, re-connecting (%s)' % paw)
                status = True
        else:
            self.log.debug('[AGENT] new connection (%s)' % paw)
            aa = dict(hostname=host, last_seen=datetime.now(), paw=paw, checks=1, executor=executor, sleep=60, server=server)
            await self.data_svc.dao.create('core_agent', aa)
            status = True
        if None if group == 'None' else group:
            await self.data_svc.create_group(name=group, paws=[paw])
        return json.dumps(dict(status=status))

    async def check_in(self, paw, executor):
        self.log.debug('[AGENT] check in (%s)' % paw)
        agent = await self.data_svc.dao.get('core_agent', dict(paw=paw))
        updated_host = dict(last_seen=datetime.now(), executor=executor, checks=agent[0]['checks'] + 1)
        await self.data_svc.dao.update('core_agent', 'paw', paw, data=updated_host)
        return agent[0]

    async def instructions(self, agent):
        sql = 'SELECT * FROM core_chain where host_id = %s and collect is null' % agent['id']
        for link in await self.data_svc.dao.raw_select(sql):
            await self.data_svc.dao.update('core_chain', key='id', value=link['id'], data=dict(collect=datetime.now()))
            return json.dumps(dict(sleep=link['jitter'], id=link['id'], command=link['command']))
        return json.dumps(dict(sleep=agent['sleep'], id=None, command=None))

    async def post_results(self, paw, link_id, output, status):
        self.log.debug('[AGENT] posting results (%s)' % paw)
        await self.data_svc.dao.create('core_result', dict(link_id=link_id, output=output))
        await self.data_svc.dao.update('core_chain', key='id', value=link_id, data=dict(status=status, finish=datetime.now()))
        return json.dumps(dict(status=True))

    @staticmethod
    async def render_file(name, group, environment, url_root):
        try:
            t = environment.get_template(name)
            return t.render(url_root=url_root, group=group)
        except Exception:
            return None

    async def download_file(self, name):
        stores = [p.store for p in self.plugins if p.store]
        for store in stores:
            for root, dirs, files in os.walk(store):
                if name in files:
                    headers = dict([('CONTENT-DISPOSITION', 'attachment; filename="%s"' % name)])
                    return os.path.join(root, name), headers
        return None, None
