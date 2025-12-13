import {exampleCronWorkflow} from '../examples';
import {CronWorkflowService} from './cron-workflow-service';
import requests from './requests';

jest.mock('./requests');

describe('cron workflow service', () => {
    describe('create', () => {
        test('with valid CronWorkflow', async () => {
            const cronWf = exampleCronWorkflow('ns');
            const request = {send: jest.fn().mockResolvedValue({body: cronWf})};
            jest.spyOn(requests, 'post').mockReturnValue(request as any);

            const result = await CronWorkflowService.create(cronWf, cronWf.metadata.namespace);

            expect(result).toStrictEqual(cronWf);
            expect(requests.post).toHaveBeenCalledWith('api/v1/cron-workflows/ns');
        });
    });

    describe('list', () => {
        test('with no results', async () => {
            jest.spyOn(requests, 'get').mockResolvedValue({body: {}} as any);

            const result = await CronWorkflowService.list('ns');

            expect(result).toStrictEqual([]);
            expect(requests.get).toHaveBeenCalledWith('api/v1/cron-workflows/ns?');
        });

        test('with multiple results', async () => {
            const items = [exampleCronWorkflow('ns'), exampleCronWorkflow('ns')];
            jest.spyOn(requests, 'get').mockResolvedValue({body: {items}} as any);

            const result = await CronWorkflowService.list('ns', ['foo', 'bar']);

            expect(result).toStrictEqual(items);
            expect(requests.get).toHaveBeenCalledWith('api/v1/cron-workflows/ns?listOptions.labelSelector=foo,bar');
        });
    });

    describe('get', () => {
        test('with valid CronWorkflow', async () => {
            const cronWf = exampleCronWorkflow('ns');
            jest.spyOn(requests, 'get').mockResolvedValue({body: cronWf} as any);

            const result = await CronWorkflowService.get(cronWf.metadata.name, 'ns');

            expect(result).toStrictEqual(cronWf);
            expect(requests.get).toHaveBeenCalledWith(`api/v1/cron-workflows/ns/${cronWf.metadata.name}`);
        });

        test('with invalid CronWorkflow missing "schedules"', async () => {
            const cronWf = exampleCronWorkflow('otherns');
            delete cronWf.spec.schedules;
            jest.spyOn(requests, 'get').mockResolvedValue({body: cronWf} as any);

            const result = await CronWorkflowService.get(cronWf.metadata.name, 'otherns');

            expect(result.spec.schedules).toEqual([]);
            expect(requests.get).toHaveBeenCalledWith(`api/v1/cron-workflows/otherns/${cronWf.metadata.name}`);
        });
    });

    describe('update', () => {
        test('with valid CronWorkflow', async () => {
            const cronWf = exampleCronWorkflow('ns');
            const request = {send: jest.fn().mockResolvedValue({body: cronWf})};
            jest.spyOn(requests, 'put').mockReturnValue(request as any);

            const result = await CronWorkflowService.update(cronWf, cronWf.metadata.name, cronWf.metadata.namespace);

            expect(result).toStrictEqual(cronWf);
            expect(requests.put).toHaveBeenCalledWith(`api/v1/cron-workflows/ns/${cronWf.metadata.name}`);
        });
    });

    describe('suspend', () => {
        test('with valid CronWorkflow', async () => {
            const cronWf = exampleCronWorkflow('ns');
            jest.spyOn(requests, 'put').mockResolvedValue({body: cronWf} as any);

            const result = await CronWorkflowService.suspend(cronWf.metadata.name, 'ns');

            expect(result).toStrictEqual(cronWf);
            expect(requests.put).toHaveBeenCalledWith(`api/v1/cron-workflows/ns/${cronWf.metadata.name}/suspend`);
        });
    });

    describe('resume', () => {
        test('with valid CronWorkflow', async () => {
            const cronWf = exampleCronWorkflow('ns');
            jest.spyOn(requests, 'put').mockResolvedValue({body: cronWf} as any);

            const result = await CronWorkflowService.resume(cronWf.metadata.name, 'ns');

            expect(result).toStrictEqual(cronWf);
            expect(requests.put).toHaveBeenCalledWith(`api/v1/cron-workflows/ns/${cronWf.metadata.name}/resume`);
        });
    });
});
