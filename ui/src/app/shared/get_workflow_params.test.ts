import {createBrowserHistory} from 'history';
import {getWorkflowParametersFromQuery} from './get_workflow_params';

describe('get_workflow_params', () => {
    it('should return an empty object when there are no query parameters', () => {
        const history = createBrowserHistory();
        const result = getWorkflowParametersFromQuery(history);
        expect(result).toEqual({});
    });

    it('should return the parameters provided in the URL', () => {
        const history = createBrowserHistory();
        history.location.search = '?parameters[key1]=value1&parameters[key2]=value2';
        const result = getWorkflowParametersFromQuery(history);
        expect(result).toEqual({
            key1: 'value1',
            key2: 'value2'
        });
    });

    it('should not return any key value pairs which are not in parameters query ', () => {
        const history = createBrowserHistory();
        history.location.search = '?retryparameters[key1]=value1&retryparameters[key2]=value2';
        const result = getWorkflowParametersFromQuery(history);
        expect(result).toEqual({});
    });

    it('should only return the parameters provided in the URL', () => {
        const history = createBrowserHistory();
        history.location.search = '?parameters[key1]=value1&parameters[key2]=value2&test=123';
        const result = getWorkflowParametersFromQuery(history);
        expect(result).toEqual({
            key1: 'value1',
            key2: 'value2'
        });
    });
});
