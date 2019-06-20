import json
from datetime import datetime


class SandService:

    def __init__(self, services):
        self.data_svc = services.get('data_svc')
        self.utility_svc = services.get('utility_svc')
        self.log = self.utility_svc.create_logger('sandcat')

    async def beacon(self, paw, platform, server, host, group, files):
        agent = await self.data_svc.dao.get('core_agent', dict(paw=paw))
        if agent:
            self.log.debug('Beacon (%s)' % paw)
            last_seen = datetime.now().strftime('%Y-%m-%d %H:%M:%S')
            updated = dict(last_seen=last_seen, checks=agent[0]['checks'] + 1, platform=platform, server=server)
            await self.data_svc.dao.update('core_agent', 'paw', paw, data=updated)
            await self.data_svc.create_group(name=group, paws=[paw])
            return agent[0]['id']
        else:
            self.log.debug('New beacon (%s)' % paw)
            queued = dict(hostname=host, last_seen=datetime.now(), paw=paw, checks=1, platform=platform,
                          server=server, files=files)
            agent_id = await self.data_svc.dao.create('core_agent', queued)
            await self.data_svc.create_group(name=group, paws=[paw])
            return agent_id

    async def instructions(self, agent_id):
        sql = 'SELECT * FROM core_chain where host_id = %s and collect is null' % agent_id
        instructions = []
        for link in await self.data_svc.dao.raw_select(sql):
            await self.data_svc.dao.update('core_chain', key='id', value=link['id'], data=dict(collect=datetime.now()))
            payload = await self.data_svc.dao.get('core_payload', dict(ability=link['ability']))
            instructions.append(json.dumps(dict(id=link['id'], sleep=link['jitter'],
                                                command=link['command'], cleanup=link['cleanup'],
                                                payload=payload[0]['payload'] if payload else '')))
        return json.dumps(instructions)

    async def post_results(self, paw, link_id, output, status):
        self.log.debug('[AGENT] posting results (%s)' % paw)
        finished = datetime.now().strftime('%Y-%m-%d %H:%M:%S')
        await self.data_svc.dao.create('core_result', dict(link_id=link_id, output=output))
        await self.data_svc.dao.update('core_chain', key='id', value=link_id, data=dict(status=int(status), finish=finished))
        return json.dumps(dict(status=True))
