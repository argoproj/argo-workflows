import {TextEncoder} from 'util';

import {ANNOTATION_KEY_POD_NAME_VERSION} from './annotations';
import {NodeStatus, Workflow} from './models';
import {createFNVHash, ensurePodNamePrefixLength, getPodName, getTemplateNameFromNode, k8sNamingHashLength, maxK8sResourceNameLength, POD_NAME_V1, POD_NAME_V2} from './pod-name';

global.TextEncoder = TextEncoder;

describe('pod names', () => {
    test('createFNVHash', () => {
        expect(createFNVHash('hello')).toEqual(1335831723);
        expect(createFNVHash('world')).toEqual(933488787);
        expect(createFNVHash('You cannot alter your fate. However, you can rise to meet it.')).toEqual(827171719);
    });

    test('createFNVHash with multibyte characters', () => {
        expect(createFNVHash('こんにちは')).toEqual(486186189);
        expect(createFNVHash('ワークフロー')).toEqual(1626941668);
        expect(createFNVHash('テスト用の日本語文字列')).toEqual(1519251954);
        expect(createFNVHash('🚀✨🔥')).toEqual(2133319838); // Emoji test
    });

    // note: the below is intended to be equivalent to the server-side Go code in workflow/util/pod_name_test.go
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

    test('ensurePodNamePrefixLength with multibyte characters', () => {
        const multibyteWfName = '日本語ワークフロー名';
        const multibyteTemplateName = 'テンプレート名サンプル';

        let expected = `${multibyteWfName}-${multibyteTemplateName}`;
        expect(ensurePodNamePrefixLength(expected)).toEqual(expected);

        const longMultibyteWfName = '非常に長い日本語のワークフロー名で色々な文字を含んでいます例えば記号や絵文字なども含まれています🚀✨🔥';
        const longMultibyteTemplateName = 'こちらも非常に長いテンプレート名でマルチバイト文字をたくさん使っています全角スペースも　含まれています';

        expected = `${longMultibyteWfName}-${longMultibyteTemplateName}`;
        const actual = ensurePodNamePrefixLength(expected);
        expect(actual.length).toBeLessThanOrEqual(maxK8sResourceNameLength - k8sNamingHashLength - 1);
        if (expected.length > maxK8sResourceNameLength - k8sNamingHashLength - 1) {
            expect(actual.length).toBeLessThanOrEqual(maxK8sResourceNameLength - k8sNamingHashLength - 1);
        }
    });

    // a node's ID is <workflow name>-<fnv32a(node name)>; the pod name takes the hash from there,
    // so the expected values below must match rehashing the node name as the pre-existing implementation did
    const nodeId = (wfName: string, nodeName: string) => `${wfName}-${createFNVHash(nodeName)}`;

    test('getPodName', () => {
        const node = {
            name: 'nodename',
            id: nodeId(shortWfName, 'nodename'),
            templateName: shortTemplateName
        } as unknown as NodeStatus;
        const wf = {
            metadata: {
                name: shortWfName,
                annotations: {
                    [ANNOTATION_KEY_POD_NAME_VERSION]: POD_NAME_V1
                }
            }
        } as unknown as Workflow;

        const v1podName = node.id;
        const v2podName = `${shortWfName}-${shortTemplateName}-${createFNVHash(node.name)}`;

        expect(getPodName(wf, node)).toEqual(v1podName);
        wf.metadata.annotations[ANNOTATION_KEY_POD_NAME_VERSION] = POD_NAME_V2;
        expect(getPodName(wf, node)).toEqual(v2podName);
        wf.metadata.annotations[ANNOTATION_KEY_POD_NAME_VERSION] = '';
        expect(getPodName(wf, node)).toEqual(v2podName);
        delete wf.metadata.annotations;
        expect(getPodName(wf, node)).toEqual(v2podName);
        expect(
            getPodName(wf, {...node, name: node.name + '.mycontainername', id: nodeId(shortWfName, node.name + '.mycontainername'), type: 'Container', boundaryID: node.id})
        ).toEqual(v2podName); // containerSet node check

        // a node that lost an ID collision carries the suffix in its ID, and so in its pod name, while its name is unchanged
        expect(getPodName(wf, {...node, id: nodeId(shortWfName, 'nodename~1')})).toEqual(`${shortWfName}-${shortTemplateName}-${createFNVHash('nodename~1')}`);

        // an ID that is not <workflow name>-<hash> falls back to hashing the node name
        expect(getPodName(wf, {...node, id: 'unrelated'})).toEqual(v2podName);

        wf.metadata.name = longWfName;
        node.id = nodeId(longWfName, node.name);
        node.templateName = longTemplateName;
        const name = getPodName(wf, node);
        expect(name.length).toEqual(maxK8sResourceNameLength);
    });

    test('getPodName with multibyte characters', () => {
        const multibyteWfName = '日本語ワークフロー';
        const multibyteTemplateName = 'テンプレート名';
        const multibyteNodeName = 'ノード名サンプル';

        const node = {
            name: multibyteNodeName,
            id: nodeId(multibyteWfName, multibyteNodeName),
            templateName: multibyteTemplateName
        } as unknown as NodeStatus;

        const wf = {
            metadata: {
                name: multibyteWfName,
                annotations: {
                    [ANNOTATION_KEY_POD_NAME_VERSION]: POD_NAME_V2
                }
            }
        } as unknown as Workflow;

        const expectedPodName = `${multibyteWfName}-${multibyteTemplateName}-${createFNVHash(multibyteNodeName)}`;
        expect(getPodName(wf, node)).toEqual(expectedPodName);

        const longMultibyteWfName = '非常に長い日本語のワークフロー名で色々な文字を含んでいます例えば記号や絵文字なども含まれています🚀✨🔥';
        const longMultibyteTemplateName = 'こちらも非常に長いテンプレート名でマルチバイト文字をたくさん使っています全角スペースも　含まれています';

        wf.metadata.name = longMultibyteWfName;
        node.id = nodeId(longMultibyteWfName, multibyteNodeName);
        node.templateName = longMultibyteTemplateName;

        const name = getPodName(wf, node);
        expect(name.length).toBeLessThanOrEqual(maxK8sResourceNameLength);

        const containerSetNodeName = `${multibyteNodeName}.コンテナ名`;
        expect(getPodName(wf, {...node, name: containerSetNodeName, id: nodeId(longMultibyteWfName, containerSetNodeName), type: 'Container', boundaryID: node.id})).toEqual(
            `${ensurePodNamePrefixLength(`${longMultibyteWfName}-${longMultibyteTemplateName}`)}-${createFNVHash(multibyteNodeName)}`
        );
    });

    test('getTemplateNameFromNode', () => {
        // case: no template ref or template name
        // expect fallback to empty string
        const node = {} as unknown as NodeStatus;
        expect(getTemplateNameFromNode(node)).toEqual('');

        // case: template ref defined but no template name defined
        node.templateRef = {
            name: 'test-template-name',
            template: 'test-template-template'
        };
        expect(getTemplateNameFromNode(node)).toEqual(node.templateRef.template);

        // case: template name defined
        node.templateName = 'test-template';
        expect(getTemplateNameFromNode(node)).toEqual(node.templateName);
    });
});
