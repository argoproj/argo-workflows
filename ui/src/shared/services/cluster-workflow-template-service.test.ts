import {exampleClusterWorkflowTemplate} from '../examples';
import {ClusterWorkflowTemplateService} from './cluster-workflow-template-service';
import requests from './requests';

jest.mock('./requests');

describe('cluster workflow template service', () => {
    describe('list', () => {
        it('passes pagination options and returns the complete list response', async () => {
            const list = {
                metadata: {continue: 'next-token'},
                items: [exampleClusterWorkflowTemplate()]
            };
            jest.spyOn(requests, 'get').mockResolvedValue({body: list} as any);

            const result = await (ClusterWorkflowTemplateService.list as any)({offset: 'current-token', limit: 20});

            expect(result).toStrictEqual(list);
            expect(requests.get).toHaveBeenCalledWith('api/v1/cluster-workflow-templates?listOptions.continue=current-token&listOptions.limit=20');
        });
    });
});
