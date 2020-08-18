import {WorkflowDag} from "./workflow-dag";
import * as React from "react";
import * as renderer from "react-test-renderer";

it('renders correctly', () => {
    const tree = renderer
        .create(<WorkflowDag workflowName={"dag-diamond-jvlhq"} nodes={{
            "dag-diamond-jvlhq": {
                "id": "dag-diamond-jvlhq",
                "name": "dag-diamond-jvlhq",
                "displayName": "dag-diamond-jvlhq",
                "type": "DAG",
                "templateName": "diamond",
                "templateScope": "local/dag-diamond-jvlhq",
                "phase": "Succeeded",
                "startedAt": "2020-08-13T15:34:38Z",
                "finishedAt": "2020-08-13T15:34:54Z",
                "children": [
                    "dag-diamond-jvlhq-3813812925"
                ],
                "outboundNodes": [
                    "dag-diamond-jvlhq-3729924830"
                ]
            },
            "dag-diamond-jvlhq-3396837735": {
                "id": "dag-diamond-jvlhq-3396837735",
                "name": "dag-diamond-jvlhq.onExit",
                "displayName": "dag-diamond-jvlhq.onExit",
                "type": "Pod",
                "templateName": "exit",
                "templateScope": "local/dag-diamond-jvlhq",
                "phase": "Succeeded",
                "startedAt": "2020-08-13T15:34:54Z",
                "finishedAt": "2020-08-13T15:34:56Z",
                "resourcesDuration": {
                    "cpu": 1,
                    "memory": 0
                },
                "outputs": {
                    "artifacts": [
                        {
                            "name": "main-logs",
                            "archiveLogs": true,
                            "s3": {
                                "endpoint": "minio:9000",
                                "bucket": "my-bucket",
                                "insecure": true,
                                "accessKeySecret": {
                                    "name": "my-minio-cred",
                                    "key": "accesskey"
                                },
                                "secretKeySecret": {
                                    "name": "my-minio-cred",
                                    "key": "secretkey"
                                },
                                "key": "dag-diamond-jvlhq/dag-diamond-jvlhq-3396837735/main.log"
                            }
                        }
                    ],
                    "exitCode": "0"
                },
                "hostNodeName": "minikube"
            },
            "dag-diamond-jvlhq-3729924830": {
                "id": "dag-diamond-jvlhq-3729924830",
                "name": "dag-diamond-jvlhq.D",
                "displayName": "D",
                "type": "Pod",
                "templateName": "echo",
                "templateScope": "local/dag-diamond-jvlhq",
                "phase": "Succeeded",
                "boundaryID": "dag-diamond-jvlhq",
                "startedAt": "2020-08-13T15:34:49Z",
                "finishedAt": "2020-08-13T15:34:52Z",
                "resourcesDuration": {
                    "cpu": 2,
                    "memory": 1
                },
                "inputs": {
                    "parameters": [
                        {
                            "name": "message",
                            "value": "D"
                        }
                    ]
                },
                "outputs": {
                    "artifacts": [
                        {
                            "name": "main-logs",
                            "archiveLogs": true,
                            "s3": {
                                "endpoint": "minio:9000",
                                "bucket": "my-bucket",
                                "insecure": true,
                                "accessKeySecret": {
                                    "name": "my-minio-cred",
                                    "key": "accesskey"
                                },
                                "secretKeySecret": {
                                    "name": "my-minio-cred",
                                    "key": "secretkey"
                                },
                                "key": "dag-diamond-jvlhq/dag-diamond-jvlhq-3729924830/main.log"
                            }
                        }
                    ],
                    "exitCode": "0"
                },
                "hostNodeName": "minikube"
            },
            "dag-diamond-jvlhq-3763480068": {
                "id": "dag-diamond-jvlhq-3763480068",
                "name": "dag-diamond-jvlhq.B",
                "displayName": "B",
                "type": "Pod",
                "templateName": "echo",
                "templateScope": "local/dag-diamond-jvlhq",
                "phase": "Succeeded",
                "boundaryID": "dag-diamond-jvlhq",
                "startedAt": "2020-08-13T15:34:41Z",
                "finishedAt": "2020-08-13T15:34:47Z",
                "resourcesDuration": {
                    "cpu": 1,
                    "memory": 0
                },
                "inputs": {
                    "parameters": [
                        {
                            "name": "message",
                            "value": "B"
                        }
                    ]
                },
                "outputs": {
                    "artifacts": [
                        {
                            "name": "main-logs",
                            "archiveLogs": true,
                            "s3": {
                                "endpoint": "minio:9000",
                                "bucket": "my-bucket",
                                "insecure": true,
                                "accessKeySecret": {
                                    "name": "my-minio-cred",
                                    "key": "accesskey"
                                },
                                "secretKeySecret": {
                                    "name": "my-minio-cred",
                                    "key": "secretkey"
                                },
                                "key": "dag-diamond-jvlhq/dag-diamond-jvlhq-3763480068/main.log"
                            }
                        }
                    ],
                    "exitCode": "0"
                },
                "children": [
                    "dag-diamond-jvlhq-3729924830"
                ],
                "hostNodeName": "minikube"
            },
            "dag-diamond-jvlhq-3780257687": {
                "id": "dag-diamond-jvlhq-3780257687",
                "name": "dag-diamond-jvlhq.C",
                "displayName": "C",
                "type": "Pod",
                "templateName": "echo",
                "templateScope": "local/dag-diamond-jvlhq",
                "phase": "Succeeded",
                "boundaryID": "dag-diamond-jvlhq",
                "startedAt": "2020-08-13T15:34:42Z",
                "finishedAt": "2020-08-13T15:34:47Z",
                "resourcesDuration": {
                    "cpu": 2,
                    "memory": 1
                },
                "inputs": {
                    "parameters": [
                        {
                            "name": "message",
                            "value": "C"
                        }
                    ]
                },
                "outputs": {
                    "artifacts": [
                        {
                            "name": "main-logs",
                            "archiveLogs": true,
                            "s3": {
                                "endpoint": "minio:9000",
                                "bucket": "my-bucket",
                                "insecure": true,
                                "accessKeySecret": {
                                    "name": "my-minio-cred",
                                    "key": "accesskey"
                                },
                                "secretKeySecret": {
                                    "name": "my-minio-cred",
                                    "key": "secretkey"
                                },
                                "key": "dag-diamond-jvlhq/dag-diamond-jvlhq-3780257687/main.log"
                            }
                        }
                    ],
                    "exitCode": "0"
                },
                "children": [
                    "dag-diamond-jvlhq-3729924830"
                ],
                "hostNodeName": "minikube"
            },
            "dag-diamond-jvlhq-3813812925": {
                "id": "dag-diamond-jvlhq-3813812925",
                "name": "dag-diamond-jvlhq.A",
                "displayName": "A",
                "type": "Pod",
                "templateName": "echo",
                "templateScope": "local/dag-diamond-jvlhq",
                "phase": "Succeeded",
                "boundaryID": "dag-diamond-jvlhq",
                "startedAt": "2020-08-13T15:34:38Z",
                "finishedAt": "2020-08-13T15:34:40Z",
                "resourcesDuration": {
                    "cpu": 1,
                    "memory": 0
                },
                "inputs": {
                    "parameters": [
                        {
                            "name": "message",
                            "value": "A"
                        }
                    ]
                },
                "outputs": {
                    "artifacts": [
                        {
                            "name": "main-logs",
                            "archiveLogs": true,
                            "s3": {
                                "endpoint": "minio:9000",
                                "bucket": "my-bucket",
                                "insecure": true,
                                "accessKeySecret": {
                                    "name": "my-minio-cred",
                                    "key": "accesskey"
                                },
                                "secretKeySecret": {
                                    "name": "my-minio-cred",
                                    "key": "secretkey"
                                },
                                "key": "dag-diamond-jvlhq/dag-diamond-jvlhq-3813812925/main.log"
                            }
                        }
                    ]
                },
                "children": [
                    "dag-diamond-jvlhq-3763480068",
                    "dag-diamond-jvlhq-3780257687"
                ],
                "hostNodeName": "minikube"
            }
        }} />)
        .toJSON();
    expect(tree).toMatchSnapshot();
});
