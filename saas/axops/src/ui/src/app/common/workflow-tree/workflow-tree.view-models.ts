import * as _ from 'lodash';
import { Task, TaskStatus, Template, StaticFixtureInfo } from '../../model';

export type UsedFixtureInfo = Task & { staticFixtureInfo: StaticFixtureInfo };
export type UsedFixtureInfoWithSteps = UsedFixtureInfo & { steps: Task[] };

export class JobTreeNode {
    public fixturesUsed: UsedFixtureInfo [] = [];
    public fixturesStatus: TaskStatus = 0;

    public static createFromTask(task: Task): JobTreeNode {
        return JobTreeNode.create(task.template, task);
    }

    public static createFromTemplate(template: Template): JobTreeNode {
        return JobTreeNode.create(template, null);
    }

    public static getChildTaskForStep(parent: Task, stepInfoId: string): Task {
        let task = (parent.children || []).find(item => item.id === stepInfoId);
        if (!task) {
            // provide empty task in Init state if task cannot be found
            task = this.createEmptyStepTask(stepInfoId);
            console.warn(`Task for ${stepInfoId} step is missing in parent ${parent.id} task children list`);
        }
        return task;
    }

    private static create(template: Template, task?: Task): JobTreeNode {
        let steps: any[] = null;
        if (template && template.type !== 'workflow') {
            if (task) {
                let step = {};
                step[task.name] = task;
                steps = [step];
                task.children = [task];
            } else {
                let step = {};
                step[template.name] = { template };
                steps = [step];
            }
        } else {
            steps = template['steps'];
        }
        let children = steps ? JobTreeNode.mapStepsToTreeNodes(steps, task) : [];
        return new JobTreeNode(task, '', children, task);
    }

    private static mapStepsToTreeNodes(steps: any[], task: Task): JobTreeNode[][] {
        return steps.map((step: any) => {
            let stepNodes: JobTreeNode[] = [];
            for (let stepName in step) {
                if (step[stepName]) {
                    let stepInfo = step[stepName];
                    let stepTemplate = stepInfo.template;
                    let children = stepTemplate.hasOwnProperty('steps') ? JobTreeNode.mapStepsToTreeNodes(stepTemplate.steps, task) : [];
                    let stepTask = task ? this.getChildTaskForStep(task, stepInfo.id) : this.createEmptyStepTask(stepInfo.id);

                    stepNodes.push(new JobTreeNode(stepTask, stepName, children, task));
                }
            }
            return stepNodes;
        });
    }

    private static createEmptyStepTask(stepInfoId: string): Task {
        return new Task({id: stepInfoId, status: TaskStatus.Init});
    }

    constructor(public value: Task, public name: string, public children: JobTreeNode[][], rootTask: Task) {
        if (value && rootTask) {
            this.fillFixtureData(value, rootTask, children);
        }
    }

    getFlattenNodes(): JobTreeNode[] {
        let result: JobTreeNode[];
        if (this.children.length === 0) {
            result = [this];
        } else {
            result = [this];
            this.children.map(subChildren => subChildren.forEach(child => {
                result = result.concat(child.getFlattenNodes());
            }));
        }
        return result;
    }

    getAllUsedFixtures(): UsedFixtureInfoWithSteps[] {
        let idToUsedFixtureInfo = new Map<string, UsedFixtureInfoWithSteps>();
        let nodes: JobTreeNode[] = [this];
        while (nodes.length > 0) {
            let next = nodes.pop();
            next.fixturesUsed.forEach(usedFixture => {
                let infoWithSteps = idToUsedFixtureInfo.get(usedFixture.id) || Object.assign({}, usedFixture, { steps: <Task[]>[] });
                infoWithSteps.steps.push(next.value);
                idToUsedFixtureInfo.set(infoWithSteps.id, infoWithSteps);
            });
            next.children.forEach(childrenGroup => nodes = nodes.concat(childrenGroup));
        }
        return Array.from(idToUsedFixtureInfo.values());
    }

    getMostRecentStartedNode() {
        let nodes = this.getFlattenNodes().filter(node => node.value.launch_time !== 0).filter(node => node !== this);
        let failedNodes = nodes.filter(node => node.value.status === TaskStatus.Failed);
        if (failedNodes.length > 0) {
            nodes = failedNodes;
        }
        nodes = nodes.sort((first: JobTreeNode, second: JobTreeNode) => second.value.launch_time - first.value.launch_time);
        if (nodes.length > 0) {
            return nodes[0];
        }
        return null;
    }

    /**
     * We will fill in the fixture related information on the child nodes
     */
    private fillFixtureData(task: Task, rootTask: Task, children: JobTreeNode[][]): void {
        let nameToFixture = new Map<string, any>();
        // Create fixture name to fixture template map
        if (task && task.template && task.template.fixtures) {
            for (let i = 0; i < task.template.fixtures.length; i++) {
                let fixtureMap = task.template.fixtures[i];
                for (let key in fixtureMap) {
                    if (fixtureMap.hasOwnProperty(key)) {
                        nameToFixture.set(key, fixtureMap[key]);
                    }
                }
            }
        }

        let fixtureToRelatives = this.getFixtureWithRelations(task, nameToFixture);

        if (fixtureToRelatives.size > 0) {
            // Associate fixtures linked to template with children
            children.forEach((list: JobTreeNode[]) => {
                list.forEach((childNode: JobTreeNode) => {
                    // iterate the parameters data on children to infer fixture usage
                    if (childNode && childNode.value['arguments']) {
                        let paramMap = childNode.value['arguments'];
                        let [fixtureTasks, fixturesStatus] = this.getUsedFixtures(paramMap, fixtureToRelatives, nameToFixture);
                        if (fixtureTasks.length > 0) {
                            let dynamicFixtures = fixtureTasks.filter(item => item.template).map(item => Object.assign({}, item, { staticFixtureInfo: <StaticFixtureInfo> null}));
                            let staticFixtureTasks = fixtureTasks.filter(item => !item.template);
                            let idToStaticFixtureInfo = new Map<string, StaticFixtureInfo>();

                            Object.keys(rootTask.fixtures || {}).map(name => rootTask.fixtures[name]).forEach(info => {
                                info.service_ids.forEach(id => {
                                    if (id.service_id && id.reference_name) {
                                        idToStaticFixtureInfo.set(id.reference_name, info);
                                    }
                                });
                            });
                            let staticFixtures = staticFixtureTasks.map(item => {
                                let info = idToStaticFixtureInfo.get(item.name) || {};
                                return Object.assign({}, item, {
                                    staticFixtureInfo: info
                                });
                            });

                            childNode.fixturesUsed = dynamicFixtures.concat(staticFixtures);
                            childNode.fixturesStatus = fixturesStatus;
                        }
                    }
                });
            });
        }
    }

    private getFixtureWithRelations(task: Task, nameToFixture: Map<string, Task>): Map<string, string[]> {
        // Create list of sets where each set contains fixture name and names of all fixtures which are dependency of that fixture
        let clusters = Array.from(nameToFixture.keys()).map(name => {
            let fixture = nameToFixture.get(name);
            return this.findFixtureNames(fixture.arguments).add(name);
        });

        // Merge all intersected sets
        let i = 1;
        while (clusters.length - i > 0) {
            let current = clusters[clusters.length - i];
            for (let j = clusters.length - 1; j >= 0; j--) {
                let next = clusters[j];
                if (next !== current) {
                    if (Array.from(next).find(item => current.has(item))) {
                        next.forEach(item => current.add(item));
                        clusters.splice(j, 1);
                    }
                }
            }
            i++;
        }

        // Create map where key fixture name and value is array of related fixtures
        let fixtureToRelatedFixtures = new Map<string, string[]>();
        for (i = 0; i < clusters.length; i++) {
            let cluster = Array.from(clusters[i]);
            cluster.forEach(fixture => fixtureToRelatedFixtures.set(fixture, cluster));
        }
        return fixtureToRelatedFixtures;
    }

    // Return set of fixture names which are referenced in given template parameters
    private findFixtureNames(params): Set<string> {
        let fixtureNames = new Set<string>();
        _.forEach(params, (val: string, key) => {
            (val.match(/%%fixtures[.](.*?)%%/g) || []).forEach(group => {
                let nameMatch = group.match(/[.](.*?)([.]|%%)/) || [];
                if (nameMatch.length >= 1) {
                    fixtureNames.add(nameMatch[1]);
                }
            });
        });
        return fixtureNames;
    }

    private getUsedFixtures(paramMap, fixtureToRelatives: Map<string, string[]>, nameToFixture: Map<string, Task>): [Task[], TaskStatus] {
        let fixturesUsed = new Set<string>();
        this.findFixtureNames(paramMap || {}).forEach(fixtureName => {
            if (fixtureToRelatives.get(fixtureName)) {
                fixtureToRelatives.get(fixtureName).forEach(item => {
                    fixturesUsed.add(item);
                });
            }
        });
        let fixtures: Task[] = Array.from(fixturesUsed).map(name => Object.assign({}, nameToFixture.get(name), {name: name}));
        let statuses = fixtures.filter(fixture => fixture.template).map(fixture => fixture.status);
        let status = TaskStatus.Success;
        if (statuses.indexOf(TaskStatus.Failed) > -1) {
            status = TaskStatus.Failed;
        } else if (statuses.indexOf(TaskStatus.Running) > -1) {
            status = TaskStatus.Running;
        }
        return [ fixtures, status ];
    }
}

export interface NodeInfo {
    name: string;
    workflow: JobTreeNode;
}
