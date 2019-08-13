import json

from datetime import datetime


class SandService:

    def __init__(self, services):
        self.data_svc = services.get('data_svc')
        self.utility_svc = services.get('utility_svc')
        self.log = self.utility_svc.create_logger('sandcat')

    async def beacon(self, paw, platform, server, group, files):
        agent = await self.data_svc.explode_agents(criteria=dict(paw=paw))
        if agent:
            self.log.debug('Beacon (%s)' % paw)
            last_seen = datetime.now().strftime('%Y-%m-%d %H:%M:%S')
            updated = dict(last_seen=last_seen, checks=agent[0]['checks'] + 1, platform=platform, server=server)
            await self.data_svc.update('core_agent', 'paw', paw, data=updated)
            return agent[0]['id']
        else:
            self.log.debug('New beacon (%s)' % paw)
            queued = dict(last_seen=datetime.now(), paw=paw, checks=1, platform=platform, server=server, files=files, host_group=group)
            agent_id = await self.data_svc.create_agent(agent=queued)
            return agent_id

    async def instructions(self, agent_id):
        commands = await self.data_svc.explode_chain(criteria=dict(host_id=agent_id))
        instructions = []
        for link in [c for c in commands if not c['collect']]:
            await self.data_svc.update('core_chain', key='id', value=link['id'], data=dict(collect=datetime.now()))
            payload = await self.data_svc.explode_payloads(criteria=dict(ability=link['ability']))
            instructions.append(json.dumps(dict(id=link['id'], sleep=link['jitter'],
                                                command=link['command'], cleanup=link['cleanup'],
                                                payload=payload[0]['payload'] if payload else '')))
        return json.dumps(instructions)

    async def post_results(self, paw, link_id, output, status):
        self.log.debug('[AGENT] posting results (%s)' % paw)
        finished = datetime.now().strftime('%Y-%m-%d %H:%M:%S')
        await self.data_svc.create_result(result=dict(link_id=link_id, output=output))
        await self.data_svc.update('core_chain', key='id', value=link_id, data=dict(status=int(status), finish=finished))
        return json.dumps(dict(status=True))
