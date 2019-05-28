import json
from datetime import datetime, timedelta


class SandService:

    def __init__(self, services):
        self.data_svc = services.get('data_svc')
        self.utility_svc = services.get('utility_svc')
        self.log = self.utility_svc.create_logger('54ndc47')
        self.plugins = services.get('plugins')

    async def registration(self, paw, executor, server, host, group):
        agent = await self.data_svc.dao.get('core_agent', dict(paw=paw))
        if agent:
            last_seen = datetime.strptime(agent[0]['last_seen'], '%Y-%m-%d %H:%M:%S.%f')
            if last_seen + timedelta(seconds=int(agent[0]['sleep'])) > datetime.now():
                self.log.debug('Agent already active - disregard (%s)' % paw)
                status = True
            else:
                self.log.console('Stale agent, re-connecting (%s)' % paw)
                status = True
        else:
            self.log.console('New agent connection (%s)' % paw)
            aa = dict(hostname=host, last_seen=datetime.now(), paw=paw, checks=1, executor=executor, sleep=60, server=server)
            await self.data_svc.dao.create('core_agent', aa)
            status = True
        if None if group == 'None' else group:
            await self.data_svc.create_group(name=group, paws=[paw])
        return json.dumps(dict(status=status))

    async def check_in(self, paw, executor):
        self.log.debug('[AGENT] check in (%s)' % paw)
        agent = await self.data_svc.dao.get('core_agent', dict(paw=paw))
        if not agent:
            self.log.debug('[AGENT] paw not recognized')
            return
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
