import {Inputs, MemoizationStatus, NodePhase, NodeStatus, NodeType, Outputs, RetryStrategy} from '../../models';
import {createFNVHash, ensurePodNamePrefixLength, getPodName, getTemplateNameFromNode, k8sNamingHashLength, maxK8sResourceNameLength, POD_NAME_V1, POD_NAME_V2} from './pod-name';

describe('pod names', () => {
    test('createFNVHash', () => {
        expect(createFNVHash('hello')).toEqual(1335831723);
        expect(createFNVHash('world')).toEqual(933488787);
        expect(createFNVHash('You cannot alter your fate. However, you can rise to meet it.')).toEqual(827171719);
    });

    // note: the below is intended to be equivalent to the server-side Go code in workflow/util/pod_name_test.go
    const nodeName = 'nodename';
    const nodeID = '1';

    const shortWfName = 'wfname';
    const shortTemplateName = 'templatename';

    const longWfName = 'alongworkflownamethatincludeslotsofdetailsandisessentiallyalargerunonsentencewithpoorstyleandnopunctuationtobehadwhatsoever';
    const longTemplateName =
        'alongtemplatenamethatincludessliightlymoredetailsandiscertainlyalargerunonstnencewithevenworsestylisticconcernsandpreposterouslyeliminatespunctuation';

    test('ensurePodNamePrefixLength', () => {
        let expected = `${shortWfName}-${shortTemplateName}`;
        expect(ensurePodNamePrefixLength(expected)).toEqual(expected);

        expected = `${longWfName}-${longTemplateName}`;
        const actual = ensurePodNamePrefixLength(expected);
        expect(actual.length).toEqual(maxK8sResourceNameLength - k8sNamingHashLength - 1);
    });

    test('getPodName', () => {
        const v1podName = nodeID;
        const v2podName = `${shortWfName}-${shortTemplateName}-${createFNVHash(nodeName)}`;
        expect(getPodName(shortWfName, nodeName, shortTemplateName, nodeID, POD_NAME_V2)).toEqual(v2podName);
        expect(getPodName(shortWfName, nodeName, shortTemplateName, nodeID, POD_NAME_V1)).toEqual(v1podName);
        expect(getPodName(shortWfName, nodeName, shortTemplateName, nodeID, '')).toEqual(v2podName);
        expect(getPodName(shortWfName, nodeName, shortTemplateName, nodeID, undefined)).toEqual(v2podName);

        const name = getPodName(longWfName, nodeName, longTemplateName, nodeID, POD_NAME_V2);
        expect(name.length).toEqual(maxK8sResourceNameLength);
    });

    test('getTemplateNameFromNode', () => {
        // case: no template ref or template name
        // expect fallback to empty string
        const nodeType: NodeType = 'Pod';
        const nodePhase: NodePhase = 'Succeeded';
        const retryStrategy: RetryStrategy = {};
        const outputs: Outputs = {};
        const inputs: Inputs = {};
        const memoizationStatus: MemoizationStatus = {
            hit: false,
            key: 'key',
            cacheName: 'cache'
        };

        const node: NodeStatus = {
            id: 'patch-processing-pipeline-ksp78-1623891970',
            name: 'patch-processing-pipeline-ksp78.retriable-map-authoring-initializer',
            displayName: 'retriable-map-authoring-initializer',
            type: nodeType,
            templateScope: 'local/',
            phase: nodePhase,
            boundaryID: '',
            message: '',
            startedAt: '',
            finishedAt: '',
            podIP: '',
            daemoned: false,
            retryStrategy,
            outputs,
            children: [],
            outboundNodes: [],
            templateName: '',
            inputs,
            hostNodeName: '',
            memoizationStatus
        };

        expect(getTemplateNameFromNode(node)).toEqual('');

        // case: template ref defined but no template name defined
        // expect to return templateRef.template
        node.templateRef = {
            name: 'test-template-name',
            template: 'test-template-template'
        };
        expect(getTemplateNameFromNode(node)).toEqual(node.templateRef.template);

        // case: template name defined
        // expect to return templateName
        node.templateName = 'test-template';
        expect(getTemplateNameFromNode(node)).toEqual(node.templateName);
    });
});
