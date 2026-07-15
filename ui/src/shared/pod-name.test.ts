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

    test('getPodName', () => {
        const node = {
            name: 'nodename',
            id: '1',
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
        expect(getPodName(wf, {...node, name: node.name + '.mycontainername', type: 'Container'})).toEqual(v2podName); // containerSet node check

        wf.metadata.name = longWfName;
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
            id: '1',
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
        node.templateName = longMultibyteTemplateName;

        const name = getPodName(wf, node);
        expect(name.length).toBeLessThanOrEqual(maxK8sResourceNameLength);

        const containerSetNodeName = `${multibyteNodeName}.コンテナ名`;
        expect(getPodName(wf, {...node, name: containerSetNodeName, type: 'Container'})).toEqual(
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
