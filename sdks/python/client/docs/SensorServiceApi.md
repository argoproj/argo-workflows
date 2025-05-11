# argo_workflows.SensorServiceApi

All URIs are relative to *http://localhost:2746*

Method | HTTP request | Description
------------- | ------------- | -------------
[**create_sensor**](SensorServiceApi.md#create_sensor) | **POST** /api/v1/sensors/{namespace} | 
[**delete_sensor**](SensorServiceApi.md#delete_sensor) | **DELETE** /api/v1/sensors/{namespace}/{name} | 
[**get_sensor**](SensorServiceApi.md#get_sensor) | **GET** /api/v1/sensors/{namespace}/{name} | 
[**list_sensors**](SensorServiceApi.md#list_sensors) | **GET** /api/v1/sensors/{namespace} | 
[**sensors_logs**](SensorServiceApi.md#sensors_logs) | **GET** /api/v1/stream/sensors/{namespace}/logs | 
[**update_sensor**](SensorServiceApi.md#update_sensor) | **PUT** /api/v1/sensors/{namespace}/{name} | 
[**watch_sensors**](SensorServiceApi.md#watch_sensors) | **GET** /api/v1/stream/sensors/{namespace} | 


# **create_sensor**
> GithubComArgoprojArgoEventsPkgApisEventsV1alpha1Sensor create_sensor(namespace, body)



### Example

* Api Key Authentication (BearerToken):

```python
import time
import argo_workflows
from argo_workflows.api import sensor_service_api
from argo_workflows.model.grpc_gateway_runtime_error import GrpcGatewayRuntimeError
from argo_workflows.model.github_com_argoproj_argo_events_pkg_apis_events_v1alpha1_sensor import GithubComArgoprojArgoEventsPkgApisEventsV1alpha1Sensor
from argo_workflows.model.sensor_create_sensor_request import SensorCreateSensorRequest
from pprint import pprint
# Defining the host is optional and defaults to http://localhost:2746
# See configuration.py for a list of all supported configuration parameters.
configuration = argo_workflows.Configuration(
    host = "http://localhost:2746"
)

# The client must configure the authentication and authorization parameters
# in accordance with the API server security policy.
# Examples for each auth method are provided below, use the example that
# satisfies your auth use case.

# Configure API key authorization: BearerToken
configuration.api_key['BearerToken'] = 'YOUR_API_KEY'

# Uncomment below to setup prefix (e.g. Bearer) for API key, if needed
# configuration.api_key_prefix['BearerToken'] = 'Bearer'

# Enter a context with an instance of the API client
with argo_workflows.ApiClient(configuration) as api_client:
    # Create an instance of the API class
    api_instance = sensor_service_api.SensorServiceApi(api_client)
    namespace = "namespace_example" # str | 
    body = SensorCreateSensorRequest(
        create_options=CreateOptions(
            dry_run=[
                "dry_run_example",
            ],
            field_manager="field_manager_example",
            field_validation="field_validation_example",
        ),
        namespace="namespace_example",
        sensor=GithubComArgoprojArgoEventsPkgApisEventsV1alpha1Sensor(
            metadata=ObjectMeta(
                annotations={
                    "key": "key_example",
                },
                creation_timestamp=dateutil_parser('1970-01-01T00:00:00.00Z'),
                deletion_grace_period_seconds=1,
                deletion_timestamp=dateutil_parser('1970-01-01T00:00:00.00Z'),
                finalizers=[
                    "finalizers_example",
                ],
                generate_name="generate_name_example",
                generation=1,
                labels={
                    "key": "key_example",
                },
                managed_fields=[
                    ManagedFieldsEntry(
                        api_version="api_version_example",
                        fields_type="fields_type_example",
                        fields_v1={},
                        manager="manager_example",
                        operation="operation_example",
                        subresource="subresource_example",
                        time=dateutil_parser('1970-01-01T00:00:00.00Z'),
                    ),
                ],
                name="name_example",
                namespace="namespace_example",
                owner_references=[
                    OwnerReference(
                        api_version="api_version_example",
                        block_owner_deletion=True,
                        controller=True,
                        kind="kind_example",
                        name="name_example",
                        uid="uid_example",
                    ),
                ],
                resource_version="resource_version_example",
                self_link="self_link_example",
                uid="uid_example",
            ),
            spec=GithubComArgoprojArgoEventsPkgApisEventsV1alpha1SensorSpec(
                dependencies=[
                    GithubComArgoprojArgoEventsPkgApisEventsV1alpha1EventDependency(
                        event_name="event_name_example",
                        event_source_name="event_source_name_example",
                        filters=GithubComArgoprojArgoEventsPkgApisEventsV1alpha1EventDependencyFilter(
                            context=GithubComArgoprojArgoEventsPkgApisEventsV1alpha1EventContext(
                                datacontenttype="datacontenttype_example",
                                id="id_example",
                                source="source_example",
                                specversion="specversion_example",
                                subject="subject_example",
                                time=dateutil_parser('1970-01-01T00:00:00.00Z'),
                                type="type_example",
                            ),
                            data=[
                                GithubComArgoprojArgoEventsPkgApisEventsV1alpha1DataFilter(
                                    comparator="comparator_example",
                                    path="path_example",
                                    template="template_example",
                                    type="type_example",
                                    value=[
                                        "value_example",
                                    ],
                                ),
                            ],
                            data_logical_operator="data_logical_operator_example",
                            expr_logical_operator="expr_logical_operator_example",
                            exprs=[
                                GithubComArgoprojArgoEventsPkgApisEventsV1alpha1ExprFilter(
                                    expr="expr_example",
                                    fields=[
                                        GithubComArgoprojArgoEventsPkgApisEventsV1alpha1PayloadField(
                                            name="name_example",
                                            path="path_example",
                                        ),
                                    ],
                                ),
                            ],
                            script="script_example",
                            time=GithubComArgoprojArgoEventsPkgApisEventsV1alpha1TimeFilter(
                                start="start_example",
                                stop="stop_example",
                            ),
                        ),
                        filters_logical_operator="filters_logical_operator_example",
                        name="name_example",
                        transform=GithubComArgoprojArgoEventsPkgApisEventsV1alpha1EventDependencyTransformer(
                            jq="jq_example",
                            script="script_example",
                        ),
                    ),
                ],
                error_on_failed_round=True,
                event_bus_name="event_bus_name_example",
                logging_fields={
                    "key": "key_example",
                },
                replicas=1,
                revision_history_limit=1,
                template=GithubComArgoprojArgoEventsPkgApisEventsV1alpha1Template(
                    affinity=Affinity(
                        node_affinity=NodeAffinity(
                            preferred_during_scheduling_ignored_during_execution=[
                                PreferredSchedulingTerm(
                                    preference=NodeSelectorTerm(
                                        match_expressions=[
                                            NodeSelectorRequirement(
                                                key="key_example",
                                                operator="operator_example",
                                                values=[
                                                    "values_example",
                                                ],
                                            ),
                                        ],
                                        match_fields=[
                                            NodeSelectorRequirement(
                                                key="key_example",
                                                operator="operator_example",
                                                values=[
                                                    "values_example",
                                                ],
                                            ),
                                        ],
                                    ),
                                    weight=1,
                                ),
                            ],
                            required_during_scheduling_ignored_during_execution=NodeSelector(
                                node_selector_terms=[
                                    NodeSelectorTerm(
                                        match_expressions=[
                                            NodeSelectorRequirement(
                                                key="key_example",
                                                operator="operator_example",
                                                values=[
                                                    "values_example",
                                                ],
                                            ),
                                        ],
                                        match_fields=[
                                            NodeSelectorRequirement(
                                                key="key_example",
                                                operator="operator_example",
                                                values=[
                                                    "values_example",
                                                ],
                                            ),
                                        ],
                                    ),
                                ],
                            ),
                        ),
                        pod_affinity=PodAffinity(
                            preferred_during_scheduling_ignored_during_execution=[
                                WeightedPodAffinityTerm(
                                    pod_affinity_term=PodAffinityTerm(
                                        label_selector=LabelSelector(
                                            match_expressions=[
                                                LabelSelectorRequirement(
                                                    key="key_example",
                                                    operator="operator_example",
                                                    values=[
                                                        "values_example",
                                                    ],
                                                ),
                                            ],
                                            match_labels={
                                                "key": "key_example",
                                            },
                                        ),
                                        match_label_keys=[
                                            "match_label_keys_example",
                                        ],
                                        mismatch_label_keys=[
                                            "mismatch_label_keys_example",
                                        ],
                                        namespace_selector=LabelSelector(
                                            match_expressions=[
                                                LabelSelectorRequirement(
                                                    key="key_example",
                                                    operator="operator_example",
                                                    values=[
                                                        "values_example",
                                                    ],
                                                ),
                                            ],
                                            match_labels={
                                                "key": "key_example",
                                            },
                                        ),
                                        namespaces=[
                                            "namespaces_example",
                                        ],
                                        topology_key="topology_key_example",
                                    ),
                                    weight=1,
                                ),
                            ],
                            required_during_scheduling_ignored_during_execution=[
                                PodAffinityTerm(
                                    label_selector=LabelSelector(
                                        match_expressions=[
                                            LabelSelectorRequirement(
                                                key="key_example",
                                                operator="operator_example",
                                                values=[
                                                    "values_example",
                                                ],
                                            ),
                                        ],
                                        match_labels={
                                            "key": "key_example",
                                        },
                                    ),
                                    match_label_keys=[
                                        "match_label_keys_example",
                                    ],
                                    mismatch_label_keys=[
                                        "mismatch_label_keys_example",
                                    ],
                                    namespace_selector=LabelSelector(
                                        match_expressions=[
                                            LabelSelectorRequirement(
                                                key="key_example",
                                                operator="operator_example",
                                                values=[
                                                    "values_example",
                                                ],
                                            ),
                                        ],
                                        match_labels={
                                            "key": "key_example",
                                        },
                                    ),
                                    namespaces=[
                                        "namespaces_example",
                                    ],
                                    topology_key="topology_key_example",
                                ),
                            ],
                        ),
                        pod_anti_affinity=PodAntiAffinity(
                            preferred_during_scheduling_ignored_during_execution=[
                                WeightedPodAffinityTerm(
                                    pod_affinity_term=PodAffinityTerm(
                                        label_selector=LabelSelector(
                                            match_expressions=[
                                                LabelSelectorRequirement(
                                                    key="key_example",
                                                    operator="operator_example",
                                                    values=[
                                                        "values_example",
                                                    ],
                                                ),
                                            ],
                                            match_labels={
                                                "key": "key_example",
                                            },
                                        ),
                                        match_label_keys=[
                                            "match_label_keys_example",
                                        ],
                                        mismatch_label_keys=[
                                            "mismatch_label_keys_example",
                                        ],
                                        namespace_selector=LabelSelector(
                                            match_expressions=[
                                                LabelSelectorRequirement(
                                                    key="key_example",
                                                    operator="operator_example",
                                                    values=[
                                                        "values_example",
                                                    ],
                                                ),
                                            ],
                                            match_labels={
                                                "key": "key_example",
                                            },
                                        ),
                                        namespaces=[
                                            "namespaces_example",
                                        ],
                                        topology_key="topology_key_example",
                                    ),
                                    weight=1,
                                ),
                            ],
                            required_during_scheduling_ignored_during_execution=[
                                PodAffinityTerm(
                                    label_selector=LabelSelector(
                                        match_expressions=[
                                            LabelSelectorRequirement(
                                                key="key_example",
                                                operator="operator_example",
                                                values=[
                                                    "values_example",
                                                ],
                                            ),
                                        ],
                                        match_labels={
                                            "key": "key_example",
                                        },
                                    ),
                                    match_label_keys=[
                                        "match_label_keys_example",
                                    ],
                                    mismatch_label_keys=[
                                        "mismatch_label_keys_example",
                                    ],
                                    namespace_selector=LabelSelector(
                                        match_expressions=[
                                            LabelSelectorRequirement(
                                                key="key_example",
                                                operator="operator_example",
                                                values=[
                                                    "values_example",
                                                ],
                                            ),
                                        ],
                                        match_labels={
                                            "key": "key_example",
                                        },
                                    ),
                                    namespaces=[
                                        "namespaces_example",
                                    ],
                                    topology_key="topology_key_example",
                                ),
                            ],
                        ),
                    ),
                    container=GithubComArgoprojArgoEventsPkgApisEventsV1alpha1Container(
                        env=[
                            EnvVar(
                                name="name_example",
                                value="value_example",
                                value_from=EnvVarSource(
                                    config_map_key_ref=ConfigMapKeySelector(
                                        key="key_example",
                                        name="name_example",
                                        optional=True,
                                    ),
                                    field_ref=ObjectFieldSelector(
                                        api_version="api_version_example",
                                        field_path="field_path_example",
                                    ),
                                    resource_field_ref=ResourceFieldSelector(
                                        container_name="container_name_example",
                                        divisor="divisor_example",
                                        resource="resource_example",
                                    ),
                                    secret_key_ref=SecretKeySelector(
                                        key="key_example",
                                        name="name_example",
                                        optional=True,
                                    ),
                                ),
                            ),
                        ],
                        env_from=[
                            EnvFromSource(
                                config_map_ref=ConfigMapEnvSource(
                                    name="name_example",
                                    optional=True,
                                ),
                                prefix="prefix_example",
                                secret_ref=SecretEnvSource(
                                    name="name_example",
                                    optional=True,
                                ),
                            ),
                        ],
                        image_pull_policy="image_pull_policy_example",
                        resources=ResourceRequirements(
                            claims=[
                                ResourceClaim(
                                    name="name_example",
                                    request="request_example",
                                ),
                            ],
                            limits={
                                "key": "key_example",
                            },
                            requests={
                                "key": "key_example",
                            },
                        ),
                        security_context=SecurityContext(
                            allow_privilege_escalation=True,
                            app_armor_profile=AppArmorProfile(
                                localhost_profile="localhost_profile_example",
                                type="type_example",
                            ),
                            capabilities=Capabilities(
                                add=[
                                    "add_example",
                                ],
                                drop=[
                                    "drop_example",
                                ],
                            ),
                            privileged=True,
                            proc_mount="proc_mount_example",
                            read_only_root_filesystem=True,
                            run_as_group=1,
                            run_as_non_root=True,
                            run_as_user=1,
                            se_linux_options=SELinuxOptions(
                                level="level_example",
                                role="role_example",
                                type="type_example",
                                user="user_example",
                            ),
                            seccomp_profile=SeccompProfile(
                                localhost_profile="localhost_profile_example",
                                type="type_example",
                            ),
                            windows_options=WindowsSecurityContextOptions(
                                gmsa_credential_spec="gmsa_credential_spec_example",
                                gmsa_credential_spec_name="gmsa_credential_spec_name_example",
                                host_process=True,
                                run_as_user_name="run_as_user_name_example",
                            ),
                        ),
                        volume_mounts=[
                            VolumeMount(
                                mount_path="mount_path_example",
                                mount_propagation="mount_propagation_example",
                                name="name_example",
                                read_only=True,
                                recursive_read_only="recursive_read_only_example",
                                sub_path="sub_path_example",
                                sub_path_expr="sub_path_expr_example",
                            ),
                        ],
                    ),
                    image_pull_secrets=[
                        LocalObjectReference(
                            name="name_example",
                        ),
                    ],
                    metadata=GithubComArgoprojArgoEventsPkgApisEventsV1alpha1Metadata(
                        annotations={
                            "key": "key_example",
                        },
                        labels={
                            "key": "key_example",
                        },
                    ),
                    node_selector={
                        "key": "key_example",
                    },
                    priority=1,
                    priority_class_name="priority_class_name_example",
                    security_context=PodSecurityContext(
                        app_armor_profile=AppArmorProfile(
                            localhost_profile="localhost_profile_example",
                            type="type_example",
                        ),
                        fs_group=1,
                        fs_group_change_policy="fs_group_change_policy_example",
                        run_as_group=1,
                        run_as_non_root=True,
                        run_as_user=1,
                        se_linux_change_policy="se_linux_change_policy_example",
                        se_linux_options=SELinuxOptions(
                            level="level_example",
                            role="role_example",
                            type="type_example",
                            user="user_example",
                        ),
                        seccomp_profile=SeccompProfile(
                            localhost_profile="localhost_profile_example",
                            type="type_example",
                        ),
                        supplemental_groups=[
                            1,
                        ],
                        supplemental_groups_policy="supplemental_groups_policy_example",
                        sysctls=[
                            Sysctl(
                                name="name_example",
                                value="value_example",
                            ),
                        ],
                        windows_options=WindowsSecurityContextOptions(
                            gmsa_credential_spec="gmsa_credential_spec_example",
                            gmsa_credential_spec_name="gmsa_credential_spec_name_example",
                            host_process=True,
                            run_as_user_name="run_as_user_name_example",
                        ),
                    ),
                    service_account_name="service_account_name_example",
                    tolerations=[
                        Toleration(
                            effect="effect_example",
                            key="key_example",
                            operator="operator_example",
                            toleration_seconds=1,
                            value="value_example",
                        ),
                    ],
                    volumes=[
                        Volume(
                            aws_elastic_block_store=AWSElasticBlockStoreVolumeSource(
                                fs_type="fs_type_example",
                                partition=1,
                                read_only=True,
                                volume_id="volume_id_example",
                            ),
                            azure_disk=AzureDiskVolumeSource(
                                caching_mode="caching_mode_example",
                                disk_name="disk_name_example",
                                disk_uri="disk_uri_example",
                                fs_type="fs_type_example",
                                kind="kind_example",
                                read_only=True,
                            ),
                            azure_file=AzureFileVolumeSource(
                                read_only=True,
                                secret_name="secret_name_example",
                                share_name="share_name_example",
                            ),
                            cephfs=CephFSVolumeSource(
                                monitors=[
                                    "monitors_example",
                                ],
                                path="path_example",
                                read_only=True,
                                secret_file="secret_file_example",
                                secret_ref=LocalObjectReference(
                                    name="name_example",
                                ),
                                user="user_example",
                            ),
                            cinder=CinderVolumeSource(
                                fs_type="fs_type_example",
                                read_only=True,
                                secret_ref=LocalObjectReference(
                                    name="name_example",
                                ),
                                volume_id="volume_id_example",
                            ),
                            config_map=ConfigMapVolumeSource(
                                default_mode=1,
                                items=[
                                    KeyToPath(
                                        key="key_example",
                                        mode=1,
                                        path="path_example",
                                    ),
                                ],
                                name="name_example",
                                optional=True,
                            ),
                            csi=CSIVolumeSource(
                                driver="driver_example",
                                fs_type="fs_type_example",
                                node_publish_secret_ref=LocalObjectReference(
                                    name="name_example",
                                ),
                                read_only=True,
                                volume_attributes={
                                    "key": "key_example",
                                },
                            ),
                            downward_api=DownwardAPIVolumeSource(
                                default_mode=1,
                                items=[
                                    DownwardAPIVolumeFile(
                                        field_ref=ObjectFieldSelector(
                                            api_version="api_version_example",
                                            field_path="field_path_example",
                                        ),
                                        mode=1,
                                        path="path_example",
                                        resource_field_ref=ResourceFieldSelector(
                                            container_name="container_name_example",
                                            divisor="divisor_example",
                                            resource="resource_example",
                                        ),
                                    ),
                                ],
                            ),
                            empty_dir=EmptyDirVolumeSource(
                                medium="medium_example",
                                size_limit="size_limit_example",
                            ),
                            ephemeral=EphemeralVolumeSource(
                                volume_claim_template=PersistentVolumeClaimTemplate(
                                    metadata=ObjectMeta(
                                        annotations={
                                            "key": "key_example",
                                        },
                                        creation_timestamp=dateutil_parser('1970-01-01T00:00:00.00Z'),
                                        deletion_grace_period_seconds=1,
                                        deletion_timestamp=dateutil_parser('1970-01-01T00:00:00.00Z'),
                                        finalizers=[
                                            "finalizers_example",
                                        ],
                                        generate_name="generate_name_example",
                                        generation=1,
                                        labels={
                                            "key": "key_example",
                                        },
                                        managed_fields=[
                                            ManagedFieldsEntry(
                                                api_version="api_version_example",
                                                fields_type="fields_type_example",
                                                fields_v1={},
                                                manager="manager_example",
                                                operation="operation_example",
                                                subresource="subresource_example",
                                                time=dateutil_parser('1970-01-01T00:00:00.00Z'),
                                            ),
                                        ],
                                        name="name_example",
                                        namespace="namespace_example",
                                        owner_references=[
                                            OwnerReference(
                                                api_version="api_version_example",
                                                block_owner_deletion=True,
                                                controller=True,
                                                kind="kind_example",
                                                name="name_example",
                                                uid="uid_example",
                                            ),
                                        ],
                                        resource_version="resource_version_example",
                                        self_link="self_link_example",
                                        uid="uid_example",
                                    ),
                                    spec=PersistentVolumeClaimSpec(
                                        access_modes=[
                                            "access_modes_example",
                                        ],
                                        data_source=TypedLocalObjectReference(
                                            api_group="api_group_example",
                                            kind="kind_example",
                                            name="name_example",
                                        ),
                                        data_source_ref=TypedObjectReference(
                                            api_group="api_group_example",
                                            kind="kind_example",
                                            name="name_example",
                                            namespace="namespace_example",
                                        ),
                                        resources=VolumeResourceRequirements(
                                            limits={
                                                "key": "key_example",
                                            },
                                            requests={
                                                "key": "key_example",
                                            },
                                        ),
                                        selector=LabelSelector(
                                            match_expressions=[
                                                LabelSelectorRequirement(
                                                    key="key_example",
                                                    operator="operator_example",
                                                    values=[
                                                        "values_example",
                                                    ],
                                                ),
                                            ],
                                            match_labels={
                                                "key": "key_example",
                                            },
                                        ),
                                        storage_class_name="storage_class_name_example",
                                        volume_attributes_class_name="volume_attributes_class_name_example",
                                        volume_mode="volume_mode_example",
                                        volume_name="volume_name_example",
                                    ),
                                ),
                            ),
                            fc=FCVolumeSource(
                                fs_type="fs_type_example",
                                lun=1,
                                read_only=True,
                                target_wwns=[
                                    "target_wwns_example",
                                ],
                                wwids=[
                                    "wwids_example",
                                ],
                            ),
                            flex_volume=FlexVolumeSource(
                                driver="driver_example",
                                fs_type="fs_type_example",
                                options={
                                    "key": "key_example",
                                },
                                read_only=True,
                                secret_ref=LocalObjectReference(
                                    name="name_example",
                                ),
                            ),
                            flocker=FlockerVolumeSource(
                                dataset_name="dataset_name_example",
                                dataset_uuid="dataset_uuid_example",
                            ),
                            gce_persistent_disk=GCEPersistentDiskVolumeSource(
                                fs_type="fs_type_example",
                                partition=1,
                                pd_name="pd_name_example",
                                read_only=True,
                            ),
                            git_repo=GitRepoVolumeSource(
                                directory="directory_example",
                                repository="repository_example",
                                revision="revision_example",
                            ),
                            glusterfs=GlusterfsVolumeSource(
                                endpoints="endpoints_example",
                                path="path_example",
                                read_only=True,
                            ),
                            host_path=HostPathVolumeSource(
                                path="path_example",
                                type="type_example",
                            ),
                            image=ImageVolumeSource(
                                pull_policy="pull_policy_example",
                                reference="reference_example",
                            ),
                            iscsi=ISCSIVolumeSource(
                                chap_auth_discovery=True,
                                chap_auth_session=True,
                                fs_type="fs_type_example",
                                initiator_name="initiator_name_example",
                                iqn="iqn_example",
                                iscsi_interface="iscsi_interface_example",
                                lun=1,
                                portals=[
                                    "portals_example",
                                ],
                                read_only=True,
                                secret_ref=LocalObjectReference(
                                    name="name_example",
                                ),
                                target_portal="target_portal_example",
                            ),
                            name="name_example",
                            nfs=NFSVolumeSource(
                                path="path_example",
                                read_only=True,
                                server="server_example",
                            ),
                            persistent_volume_claim=PersistentVolumeClaimVolumeSource(
                                claim_name="claim_name_example",
                                read_only=True,
                            ),
                            photon_persistent_disk=PhotonPersistentDiskVolumeSource(
                                fs_type="fs_type_example",
                                pd_id="pd_id_example",
                            ),
                            portworx_volume=PortworxVolumeSource(
                                fs_type="fs_type_example",
                                read_only=True,
                                volume_id="volume_id_example",
                            ),
                            projected=ProjectedVolumeSource(
                                default_mode=1,
                                sources=[
                                    VolumeProjection(
                                        cluster_trust_bundle=ClusterTrustBundleProjection(
                                            label_selector=LabelSelector(
                                                match_expressions=[
                                                    LabelSelectorRequirement(
                                                        key="key_example",
                                                        operator="operator_example",
                                                        values=[
                                                            "values_example",
                                                        ],
                                                    ),
                                                ],
                                                match_labels={
                                                    "key": "key_example",
                                                },
                                            ),
                                            name="name_example",
                                            optional=True,
                                            path="path_example",
                                            signer_name="signer_name_example",
                                        ),
                                        config_map=ConfigMapProjection(
                                            items=[
                                                KeyToPath(
                                                    key="key_example",
                                                    mode=1,
                                                    path="path_example",
                                                ),
                                            ],
                                            name="name_example",
                                            optional=True,
                                        ),
                                        downward_api=DownwardAPIProjection(
                                            items=[
                                                DownwardAPIVolumeFile(
                                                    field_ref=ObjectFieldSelector(
                                                        api_version="api_version_example",
                                                        field_path="field_path_example",
                                                    ),
                                                    mode=1,
                                                    path="path_example",
                                                    resource_field_ref=ResourceFieldSelector(
                                                        container_name="container_name_example",
                                                        divisor="divisor_example",
                                                        resource="resource_example",
                                                    ),
                                                ),
                                            ],
                                        ),
                                        secret=SecretProjection(
                                            items=[
                                                KeyToPath(
                                                    key="key_example",
                                                    mode=1,
                                                    path="path_example",
                                                ),
                                            ],
                                            name="name_example",
                                            optional=True,
                                        ),
                                        service_account_token=ServiceAccountTokenProjection(
                                            audience="audience_example",
                                            expiration_seconds=1,
                                            path="path_example",
                                        ),
                                    ),
                                ],
                            ),
                            quobyte=QuobyteVolumeSource(
                                group="group_example",
                                read_only=True,
                                registry="registry_example",
                                tenant="tenant_example",
                                user="user_example",
                                volume="volume_example",
                            ),
                            rbd=RBDVolumeSource(
                                fs_type="fs_type_example",
                                image="image_example",
                                keyring="keyring_example",
                                monitors=[
                                    "monitors_example",
                                ],
                                pool="pool_example",
                                read_only=True,
                                secret_ref=LocalObjectReference(
                                    name="name_example",
                                ),
                                user="user_example",
                            ),
                            scale_io=ScaleIOVolumeSource(
                                fs_type="fs_type_example",
                                gateway="gateway_example",
                                protection_domain="protection_domain_example",
                                read_only=True,
                                secret_ref=LocalObjectReference(
                                    name="name_example",
                                ),
                                ssl_enabled=True,
                                storage_mode="storage_mode_example",
                                storage_pool="storage_pool_example",
                                system="system_example",
                                volume_name="volume_name_example",
                            ),
                            secret=SecretVolumeSource(
                                default_mode=1,
                                items=[
                                    KeyToPath(
                                        key="key_example",
                                        mode=1,
                                        path="path_example",
                                    ),
                                ],
                                optional=True,
                                secret_name="secret_name_example",
                            ),
                            storageos=StorageOSVolumeSource(
                                fs_type="fs_type_example",
                                read_only=True,
                                secret_ref=LocalObjectReference(
                                    name="name_example",
                                ),
                                volume_name="volume_name_example",
                                volume_namespace="volume_namespace_example",
                            ),
                            vsphere_volume=VsphereVirtualDiskVolumeSource(
                                fs_type="fs_type_example",
                                storage_policy_id="storage_policy_id_example",
                                storage_policy_name="storage_policy_name_example",
                                volume_path="volume_path_example",
                            ),
                        ),
                    ],
                ),
                triggers=[
                    GithubComArgoprojArgoEventsPkgApisEventsV1alpha1Trigger(
                        at_least_once=True,
                        dlq_trigger=GithubComArgoprojArgoEventsPkgApisEventsV1alpha1Trigger(),
                        parameters=[
                            GithubComArgoprojArgoEventsPkgApisEventsV1alpha1TriggerParameter(
                                dest="dest_example",
                                operation="operation_example",
                                src=GithubComArgoprojArgoEventsPkgApisEventsV1alpha1TriggerParameterSource(
                                    context_key="context_key_example",
                                    context_template="context_template_example",
                                    data_key="data_key_example",
                                    data_template="data_template_example",
                                    dependency_name="dependency_name_example",
                                    use_raw_data=True,
                                    value="value_example",
                                ),
                            ),
                        ],
                        policy=GithubComArgoprojArgoEventsPkgApisEventsV1alpha1TriggerPolicy(
                            k8s=GithubComArgoprojArgoEventsPkgApisEventsV1alpha1K8SResourcePolicy(
                                backoff=GithubComArgoprojArgoEventsPkgApisEventsV1alpha1Backoff(
                                    duration=GithubComArgoprojArgoEventsPkgApisEventsV1alpha1Int64OrString(
                                        int64_val="int64_val_example",
                                        str_val="str_val_example",
                                        type="type_example",
                                    ),
                                    factor=GithubComArgoprojArgoEventsPkgApisEventsV1alpha1Amount(
                                        value='YQ==',
                                    ),
                                    jitter=GithubComArgoprojArgoEventsPkgApisEventsV1alpha1Amount(
                                        value='YQ==',
                                    ),
                                    steps=1,
                                ),
                                error_on_backoff_timeout=True,
                                labels={
                                    "key": "key_example",
                                },
                            ),
                            status=GithubComArgoprojArgoEventsPkgApisEventsV1alpha1StatusPolicy(
                                allow=[
                                    1,
                                ],
                            ),
                        ),
                        rate_limit=GithubComArgoprojArgoEventsPkgApisEventsV1alpha1RateLimit(
                            requests_per_unit=1,
                            unit="unit_example",
                        ),
                        retry_strategy=GithubComArgoprojArgoEventsPkgApisEventsV1alpha1Backoff(
                            duration=GithubComArgoprojArgoEventsPkgApisEventsV1alpha1Int64OrString(
                                int64_val="int64_val_example",
                                str_val="str_val_example",
                                type="type_example",
                            ),
                            factor=GithubComArgoprojArgoEventsPkgApisEventsV1alpha1Amount(
                                value='YQ==',
                            ),
                            jitter=GithubComArgoprojArgoEventsPkgApisEventsV1alpha1Amount(
                                value='YQ==',
                            ),
                            steps=1,
                        ),
                        template=GithubComArgoprojArgoEventsPkgApisEventsV1alpha1TriggerTemplate(
                            argo_workflow=GithubComArgoprojArgoEventsPkgApisEventsV1alpha1ArgoWorkflowTrigger(
                                args=[
                                    "args_example",
                                ],
                                operation="operation_example",
                                parameters=[
                                    GithubComArgoprojArgoEventsPkgApisEventsV1alpha1TriggerParameter(
                                        dest="dest_example",
                                        operation="operation_example",
                                        src=GithubComArgoprojArgoEventsPkgApisEventsV1alpha1TriggerParameterSource(
                                            context_key="context_key_example",
                                            context_template="context_template_example",
                                            data_key="data_key_example",
                                            data_template="data_template_example",
                                            dependency_name="dependency_name_example",
                                            use_raw_data=True,
                                            value="value_example",
                                        ),
                                    ),
                                ],
                                source=GithubComArgoprojArgoEventsPkgApisEventsV1alpha1ArtifactLocation(
                                    configmap=ConfigMapKeySelector(
                                        key="key_example",
                                        name="name_example",
                                        optional=True,
                                    ),
                                    file=GithubComArgoprojArgoEventsPkgApisEventsV1alpha1FileArtifact(
                                        path="path_example",
                                    ),
                                    git=GithubComArgoprojArgoEventsPkgApisEventsV1alpha1GitArtifact(
                                        branch="branch_example",
                                        clone_directory="clone_directory_example",
                                        creds=GithubComArgoprojArgoEventsPkgApisEventsV1alpha1GitCreds(
                                            password=SecretKeySelector(
                                                key="key_example",
                                                name="name_example",
                                                optional=True,
                                            ),
                                            username=SecretKeySelector(
                                                key="key_example",
                                                name="name_example",
                                                optional=True,
                                            ),
                                        ),
                                        file_path="file_path_example",
                                        insecure_ignore_host_key=True,
                                        ref="ref_example",
                                        remote=GithubComArgoprojArgoEventsPkgApisEventsV1alpha1GitRemoteConfig(
                                            name="name_example",
                                            urls=[
                                                "urls_example",
                                            ],
                                        ),
                                        ssh_key_secret=SecretKeySelector(
                                            key="key_example",
                                            name="name_example",
                                            optional=True,
                                        ),
                                        tag="tag_example",
                                        url="url_example",
                                    ),
                                    inline="inline_example",
                                    resource=GithubComArgoprojArgoEventsPkgApisEventsV1alpha1K8SResource(
                                        value='YQ==',
                                    ),
                                    s3=GithubComArgoprojArgoEventsPkgApisEventsV1alpha1S3Artifact(
                                        access_key=SecretKeySelector(
                                            key="key_example",
                                            name="name_example",
                                            optional=True,
                                        ),
                                        bucket=GithubComArgoprojArgoEventsPkgApisEventsV1alpha1S3Bucket(
                                            key="key_example",
                                            name="name_example",
                                        ),
                                        ca_certificate=SecretKeySelector(
                                            key="key_example",
                                            name="name_example",
                                            optional=True,
                                        ),
                                        endpoint="endpoint_example",
                                        events=[
                                            "events_example",
                                        ],
                                        filter=GithubComArgoprojArgoEventsPkgApisEventsV1alpha1S3Filter(
                                            prefix="prefix_example",
                                            suffix="suffix_example",
                                        ),
                                        insecure=True,
                                        metadata={
                                            "key": "key_example",
                                        },
                                        region="region_example",
                                        secret_key=SecretKeySelector(
                                            key="key_example",
                                            name="name_example",
                                            optional=True,
                                        ),
                                    ),
                                    url=GithubComArgoprojArgoEventsPkgApisEventsV1alpha1URLArtifact(
                                        path="path_example",
                                        verify_cert=True,
                                    ),
                                ),
                            ),
                            aws_lambda=GithubComArgoprojArgoEventsPkgApisEventsV1alpha1AWSLambdaTrigger(
                                access_key=SecretKeySelector(
                                    key="key_example",
                                    name="name_example",
                                    optional=True,
                                ),
                                function_name="function_name_example",
                                invocation_type="invocation_type_example",
                                parameters=[
                                    GithubComArgoprojArgoEventsPkgApisEventsV1alpha1TriggerParameter(
                                        dest="dest_example",
                                        operation="operation_example",
                                        src=GithubComArgoprojArgoEventsPkgApisEventsV1alpha1TriggerParameterSource(
                                            context_key="context_key_example",
                                            context_template="context_template_example",
                                            data_key="data_key_example",
                                            data_template="data_template_example",
                                            dependency_name="dependency_name_example",
                                            use_raw_data=True,
                                            value="value_example",
                                        ),
                                    ),
                                ],
                                payload=[
                                    GithubComArgoprojArgoEventsPkgApisEventsV1alpha1TriggerParameter(
                                        dest="dest_example",
                                        operation="operation_example",
                                        src=GithubComArgoprojArgoEventsPkgApisEventsV1alpha1TriggerParameterSource(
                                            context_key="context_key_example",
                                            context_template="context_template_example",
                                            data_key="data_key_example",
                                            data_template="data_template_example",
                                            dependency_name="dependency_name_example",
                                            use_raw_data=True,
                                            value="value_example",
                                        ),
                                    ),
                                ],
                                region="region_example",
                                role_arn="role_arn_example",
                                secret_key=SecretKeySelector(
                                    key="key_example",
                                    name="name_example",
                                    optional=True,
                                ),
                            ),
                            azure_event_hubs=GithubComArgoprojArgoEventsPkgApisEventsV1alpha1AzureEventHubsTrigger(
                                fqdn="fqdn_example",
                                hub_name="hub_name_example",
                                parameters=[
                                    GithubComArgoprojArgoEventsPkgApisEventsV1alpha1TriggerParameter(
                                        dest="dest_example",
                                        operation="operation_example",
                                        src=GithubComArgoprojArgoEventsPkgApisEventsV1alpha1TriggerParameterSource(
                                            context_key="context_key_example",
                                            context_template="context_template_example",
                                            data_key="data_key_example",
                                            data_template="data_template_example",
                                            dependency_name="dependency_name_example",
                                            use_raw_data=True,
                                            value="value_example",
                                        ),
                                    ),
                                ],
                                payload=[
                                    GithubComArgoprojArgoEventsPkgApisEventsV1alpha1TriggerParameter(
                                        dest="dest_example",
                                        operation="operation_example",
                                        src=GithubComArgoprojArgoEventsPkgApisEventsV1alpha1TriggerParameterSource(
                                            context_key="context_key_example",
                                            context_template="context_template_example",
                                            data_key="data_key_example",
                                            data_template="data_template_example",
                                            dependency_name="dependency_name_example",
                                            use_raw_data=True,
                                            value="value_example",
                                        ),
                                    ),
                                ],
                                shared_access_key=SecretKeySelector(
                                    key="key_example",
                                    name="name_example",
                                    optional=True,
                                ),
                                shared_access_key_name=SecretKeySelector(
                                    key="key_example",
                                    name="name_example",
                                    optional=True,
                                ),
                            ),
                            azure_service_bus=GithubComArgoprojArgoEventsPkgApisEventsV1alpha1AzureServiceBusTrigger(
                                connection_string=SecretKeySelector(
                                    key="key_example",
                                    name="name_example",
                                    optional=True,
                                ),
                                parameters=[
                                    GithubComArgoprojArgoEventsPkgApisEventsV1alpha1TriggerParameter(
                                        dest="dest_example",
                                        operation="operation_example",
                                        src=GithubComArgoprojArgoEventsPkgApisEventsV1alpha1TriggerParameterSource(
                                            context_key="context_key_example",
                                            context_template="context_template_example",
                                            data_key="data_key_example",
                                            data_template="data_template_example",
                                            dependency_name="dependency_name_example",
                                            use_raw_data=True,
                                            value="value_example",
                                        ),
                                    ),
                                ],
                                payload=[
                                    GithubComArgoprojArgoEventsPkgApisEventsV1alpha1TriggerParameter(
                                        dest="dest_example",
                                        operation="operation_example",
                                        src=GithubComArgoprojArgoEventsPkgApisEventsV1alpha1TriggerParameterSource(
                                            context_key="context_key_example",
                                            context_template="context_template_example",
                                            data_key="data_key_example",
                                            data_template="data_template_example",
                                            dependency_name="dependency_name_example",
                                            use_raw_data=True,
                                            value="value_example",
                                        ),
                                    ),
                                ],
                                queue_name="queue_name_example",
                                subscription_name="subscription_name_example",
                                tls=GithubComArgoprojArgoEventsPkgApisEventsV1alpha1TLSConfig(
                                    ca_cert_secret=SecretKeySelector(
                                        key="key_example",
                                        name="name_example",
                                        optional=True,
                                    ),
                                    client_cert_secret=SecretKeySelector(
                                        key="key_example",
                                        name="name_example",
                                        optional=True,
                                    ),
                                    client_key_secret=SecretKeySelector(
                                        key="key_example",
                                        name="name_example",
                                        optional=True,
                                    ),
                                    insecure_skip_verify=True,
                                ),
                                topic_name="topic_name_example",
                            ),
                            conditions="conditions_example",
                            conditions_reset=[
                                GithubComArgoprojArgoEventsPkgApisEventsV1alpha1ConditionsResetCriteria(
                                    by_time=GithubComArgoprojArgoEventsPkgApisEventsV1alpha1ConditionsResetByTime(
                                        cron="cron_example",
                                        timezone="timezone_example",
                                    ),
                                ),
                            ],
                            custom=GithubComArgoprojArgoEventsPkgApisEventsV1alpha1CustomTrigger(
                                cert_secret=SecretKeySelector(
                                    key="key_example",
                                    name="name_example",
                                    optional=True,
                                ),
                                parameters=[
                                    GithubComArgoprojArgoEventsPkgApisEventsV1alpha1TriggerParameter(
                                        dest="dest_example",
                                        operation="operation_example",
                                        src=GithubComArgoprojArgoEventsPkgApisEventsV1alpha1TriggerParameterSource(
                                            context_key="context_key_example",
                                            context_template="context_template_example",
                                            data_key="data_key_example",
                                            data_template="data_template_example",
                                            dependency_name="dependency_name_example",
                                            use_raw_data=True,
                                            value="value_example",
                                        ),
                                    ),
                                ],
                                payload=[
                                    GithubComArgoprojArgoEventsPkgApisEventsV1alpha1TriggerParameter(
                                        dest="dest_example",
                                        operation="operation_example",
                                        src=GithubComArgoprojArgoEventsPkgApisEventsV1alpha1TriggerParameterSource(
                                            context_key="context_key_example",
                                            context_template="context_template_example",
                                            data_key="data_key_example",
                                            data_template="data_template_example",
                                            dependency_name="dependency_name_example",
                                            use_raw_data=True,
                                            value="value_example",
                                        ),
                                    ),
                                ],
                                secure=True,
                                server_name_override="server_name_override_example",
                                server_url="server_url_example",
                                spec={
                                    "key": "key_example",
                                },
                            ),
                            email=GithubComArgoprojArgoEventsPkgApisEventsV1alpha1EmailTrigger(
                                body="body_example",
                                _from="_from_example",
                                host="host_example",
                                parameters=[
                                    GithubComArgoprojArgoEventsPkgApisEventsV1alpha1TriggerParameter(
                                        dest="dest_example",
                                        operation="operation_example",
                                        src=GithubComArgoprojArgoEventsPkgApisEventsV1alpha1TriggerParameterSource(
                                            context_key="context_key_example",
                                            context_template="context_template_example",
                                            data_key="data_key_example",
                                            data_template="data_template_example",
                                            dependency_name="dependency_name_example",
                                            use_raw_data=True,
                                            value="value_example",
                                        ),
                                    ),
                                ],
                                port=1,
                                smtp_password=SecretKeySelector(
                                    key="key_example",
                                    name="name_example",
                                    optional=True,
                                ),
                                subject="subject_example",
                                to=[
                                    "to_example",
                                ],
                                username="username_example",
                            ),
                            http=GithubComArgoprojArgoEventsPkgApisEventsV1alpha1HTTPTrigger(
                                basic_auth=GithubComArgoprojArgoEventsPkgApisEventsV1alpha1BasicAuth(
                                    password=SecretKeySelector(
                                        key="key_example",
                                        name="name_example",
                                        optional=True,
                                    ),
                                    username=SecretKeySelector(
                                        key="key_example",
                                        name="name_example",
                                        optional=True,
                                    ),
                                ),
                                headers={
                                    "key": "key_example",
                                },
                                method="method_example",
                                parameters=[
                                    GithubComArgoprojArgoEventsPkgApisEventsV1alpha1TriggerParameter(
                                        dest="dest_example",
                                        operation="operation_example",
                                        src=GithubComArgoprojArgoEventsPkgApisEventsV1alpha1TriggerParameterSource(
                                            context_key="context_key_example",
                                            context_template="context_template_example",
                                            data_key="data_key_example",
                                            data_template="data_template_example",
                                            dependency_name="dependency_name_example",
                                            use_raw_data=True,
                                            value="value_example",
                                        ),
                                    ),
                                ],
                                payload=[
                                    GithubComArgoprojArgoEventsPkgApisEventsV1alpha1TriggerParameter(
                                        dest="dest_example",
                                        operation="operation_example",
                                        src=GithubComArgoprojArgoEventsPkgApisEventsV1alpha1TriggerParameterSource(
                                            context_key="context_key_example",
                                            context_template="context_template_example",
                                            data_key="data_key_example",
                                            data_template="data_template_example",
                                            dependency_name="dependency_name_example",
                                            use_raw_data=True,
                                            value="value_example",
                                        ),
                                    ),
                                ],
                                secure_headers=[
                                    GithubComArgoprojArgoEventsPkgApisEventsV1alpha1SecureHeader(
                                        name="name_example",
                                        value_from=GithubComArgoprojArgoEventsPkgApisEventsV1alpha1ValueFromSource(
                                            config_map_key_ref=ConfigMapKeySelector(
                                                key="key_example",
                                                name="name_example",
                                                optional=True,
                                            ),
                                            secret_key_ref=SecretKeySelector(
                                                key="key_example",
                                                name="name_example",
                                                optional=True,
                                            ),
                                        ),
                                    ),
                                ],
                                timeout="timeout_example",
                                tls=GithubComArgoprojArgoEventsPkgApisEventsV1alpha1TLSConfig(
                                    ca_cert_secret=SecretKeySelector(
                                        key="key_example",
                                        name="name_example",
                                        optional=True,
                                    ),
                                    client_cert_secret=SecretKeySelector(
                                        key="key_example",
                                        name="name_example",
                                        optional=True,
                                    ),
                                    client_key_secret=SecretKeySelector(
                                        key="key_example",
                                        name="name_example",
                                        optional=True,
                                    ),
                                    insecure_skip_verify=True,
                                ),
                                url="url_example",
                            ),
                            k8s=GithubComArgoprojArgoEventsPkgApisEventsV1alpha1StandardK8STrigger(
                                live_object=True,
                                operation="operation_example",
                                parameters=[
                                    GithubComArgoprojArgoEventsPkgApisEventsV1alpha1TriggerParameter(
                                        dest="dest_example",
                                        operation="operation_example",
                                        src=GithubComArgoprojArgoEventsPkgApisEventsV1alpha1TriggerParameterSource(
                                            context_key="context_key_example",
                                            context_template="context_template_example",
                                            data_key="data_key_example",
                                            data_template="data_template_example",
                                            dependency_name="dependency_name_example",
                                            use_raw_data=True,
                                            value="value_example",
                                        ),
                                    ),
                                ],
                                patch_strategy="patch_strategy_example",
                                source=GithubComArgoprojArgoEventsPkgApisEventsV1alpha1ArtifactLocation(
                                    configmap=ConfigMapKeySelector(
                                        key="key_example",
                                        name="name_example",
                                        optional=True,
                                    ),
                                    file=GithubComArgoprojArgoEventsPkgApisEventsV1alpha1FileArtifact(
                                        path="path_example",
                                    ),
                                    git=GithubComArgoprojArgoEventsPkgApisEventsV1alpha1GitArtifact(
                                        branch="branch_example",
                                        clone_directory="clone_directory_example",
                                        creds=GithubComArgoprojArgoEventsPkgApisEventsV1alpha1GitCreds(
                                            password=SecretKeySelector(
                                                key="key_example",
                                                name="name_example",
                                                optional=True,
                                            ),
                                            username=SecretKeySelector(
                                                key="key_example",
                                                name="name_example",
                                                optional=True,
                                            ),
                                        ),
                                        file_path="file_path_example",
                                        insecure_ignore_host_key=True,
                                        ref="ref_example",
                                        remote=GithubComArgoprojArgoEventsPkgApisEventsV1alpha1GitRemoteConfig(
                                            name="name_example",
                                            urls=[
                                                "urls_example",
                                            ],
                                        ),
                                        ssh_key_secret=SecretKeySelector(
                                            key="key_example",
                                            name="name_example",
                                            optional=True,
                                        ),
                                        tag="tag_example",
                                        url="url_example",
                                    ),
                                    inline="inline_example",
                                    resource=GithubComArgoprojArgoEventsPkgApisEventsV1alpha1K8SResource(
                                        value='YQ==',
                                    ),
                                    s3=GithubComArgoprojArgoEventsPkgApisEventsV1alpha1S3Artifact(
                                        access_key=SecretKeySelector(
                                            key="key_example",
                                            name="name_example",
                                            optional=True,
                                        ),
                                        bucket=GithubComArgoprojArgoEventsPkgApisEventsV1alpha1S3Bucket(
                                            key="key_example",
                                            name="name_example",
                                        ),
                                        ca_certificate=SecretKeySelector(
                                            key="key_example",
                                            name="name_example",
                                            optional=True,
                                        ),
                                        endpoint="endpoint_example",
                                        events=[
                                            "events_example",
                                        ],
                                        filter=GithubComArgoprojArgoEventsPkgApisEventsV1alpha1S3Filter(
                                            prefix="prefix_example",
                                            suffix="suffix_example",
                                        ),
                                        insecure=True,
                                        metadata={
                                            "key": "key_example",
                                        },
                                        region="region_example",
                                        secret_key=SecretKeySelector(
                                            key="key_example",
                                            name="name_example",
                                            optional=True,
                                        ),
                                    ),
                                    url=GithubComArgoprojArgoEventsPkgApisEventsV1alpha1URLArtifact(
                                        path="path_example",
                                        verify_cert=True,
                                    ),
                                ),
                            ),
                            kafka=GithubComArgoprojArgoEventsPkgApisEventsV1alpha1KafkaTrigger(
                                compress=True,
                                flush_frequency=1,
                                headers={
                                    "key": "key_example",
                                },
                                parameters=[
                                    GithubComArgoprojArgoEventsPkgApisEventsV1alpha1TriggerParameter(
                                        dest="dest_example",
                                        operation="operation_example",
                                        src=GithubComArgoprojArgoEventsPkgApisEventsV1alpha1TriggerParameterSource(
                                            context_key="context_key_example",
                                            context_template="context_template_example",
                                            data_key="data_key_example",
                                            data_template="data_template_example",
                                            dependency_name="dependency_name_example",
                                            use_raw_data=True,
                                            value="value_example",
                                        ),
                                    ),
                                ],
                                partition=1,
                                partitioning_key="partitioning_key_example",
                                payload=[
                                    GithubComArgoprojArgoEventsPkgApisEventsV1alpha1TriggerParameter(
                                        dest="dest_example",
                                        operation="operation_example",
                                        src=GithubComArgoprojArgoEventsPkgApisEventsV1alpha1TriggerParameterSource(
                                            context_key="context_key_example",
                                            context_template="context_template_example",
                                            data_key="data_key_example",
                                            data_template="data_template_example",
                                            dependency_name="dependency_name_example",
                                            use_raw_data=True,
                                            value="value_example",
                                        ),
                                    ),
                                ],
                                required_acks=1,
                                sasl=GithubComArgoprojArgoEventsPkgApisEventsV1alpha1SASLConfig(
                                    mechanism="mechanism_example",
                                    password_secret=SecretKeySelector(
                                        key="key_example",
                                        name="name_example",
                                        optional=True,
                                    ),
                                    user_secret=SecretKeySelector(
                                        key="key_example",
                                        name="name_example",
                                        optional=True,
                                    ),
                                ),
                                schema_registry=GithubComArgoprojArgoEventsPkgApisEventsV1alpha1SchemaRegistryConfig(
                                    auth=GithubComArgoprojArgoEventsPkgApisEventsV1alpha1BasicAuth(
                                        password=SecretKeySelector(
                                            key="key_example",
                                            name="name_example",
                                            optional=True,
                                        ),
                                        username=SecretKeySelector(
                                            key="key_example",
                                            name="name_example",
                                            optional=True,
                                        ),
                                    ),
                                    schema_id=1,
                                    url="url_example",
                                ),
                                secure_headers=[
                                    GithubComArgoprojArgoEventsPkgApisEventsV1alpha1SecureHeader(
                                        name="name_example",
                                        value_from=GithubComArgoprojArgoEventsPkgApisEventsV1alpha1ValueFromSource(
                                            config_map_key_ref=ConfigMapKeySelector(
                                                key="key_example",
                                                name="name_example",
                                                optional=True,
                                            ),
                                            secret_key_ref=SecretKeySelector(
                                                key="key_example",
                                                name="name_example",
                                                optional=True,
                                            ),
                                        ),
                                    ),
                                ],
                                tls=GithubComArgoprojArgoEventsPkgApisEventsV1alpha1TLSConfig(
                                    ca_cert_secret=SecretKeySelector(
                                        key="key_example",
                                        name="name_example",
                                        optional=True,
                                    ),
                                    client_cert_secret=SecretKeySelector(
                                        key="key_example",
                                        name="name_example",
                                        optional=True,
                                    ),
                                    client_key_secret=SecretKeySelector(
                                        key="key_example",
                                        name="name_example",
                                        optional=True,
                                    ),
                                    insecure_skip_verify=True,
                                ),
                                topic="topic_example",
                                url="url_example",
                                version="version_example",
                            ),
                            log=GithubComArgoprojArgoEventsPkgApisEventsV1alpha1LogTrigger(
                                interval_seconds="interval_seconds_example",
                            ),
                            name="name_example",
                            nats=GithubComArgoprojArgoEventsPkgApisEventsV1alpha1NATSTrigger(
                                auth=GithubComArgoprojArgoEventsPkgApisEventsV1alpha1NATSAuth(
                                    basic=GithubComArgoprojArgoEventsPkgApisEventsV1alpha1BasicAuth(
                                        password=SecretKeySelector(
                                            key="key_example",
                                            name="name_example",
                                            optional=True,
                                        ),
                                        username=SecretKeySelector(
                                            key="key_example",
                                            name="name_example",
                                            optional=True,
                                        ),
                                    ),
                                    credential=SecretKeySelector(
                                        key="key_example",
                                        name="name_example",
                                        optional=True,
                                    ),
                                    nkey=SecretKeySelector(
                                        key="key_example",
                                        name="name_example",
                                        optional=True,
                                    ),
                                    token=SecretKeySelector(
                                        key="key_example",
                                        name="name_example",
                                        optional=True,
                                    ),
                                ),
                                parameters=[
                                    GithubComArgoprojArgoEventsPkgApisEventsV1alpha1TriggerParameter(
                                        dest="dest_example",
                                        operation="operation_example",
                                        src=GithubComArgoprojArgoEventsPkgApisEventsV1alpha1TriggerParameterSource(
                                            context_key="context_key_example",
                                            context_template="context_template_example",
                                            data_key="data_key_example",
                                            data_template="data_template_example",
                                            dependency_name="dependency_name_example",
                                            use_raw_data=True,
                                            value="value_example",
                                        ),
                                    ),
                                ],
                                payload=[
                                    GithubComArgoprojArgoEventsPkgApisEventsV1alpha1TriggerParameter(
                                        dest="dest_example",
                                        operation="operation_example",
                                        src=GithubComArgoprojArgoEventsPkgApisEventsV1alpha1TriggerParameterSource(
                                            context_key="context_key_example",
                                            context_template="context_template_example",
                                            data_key="data_key_example",
                                            data_template="data_template_example",
                                            dependency_name="dependency_name_example",
                                            use_raw_data=True,
                                            value="value_example",
                                        ),
                                    ),
                                ],
                                subject="subject_example",
                                tls=GithubComArgoprojArgoEventsPkgApisEventsV1alpha1TLSConfig(
                                    ca_cert_secret=SecretKeySelector(
                                        key="key_example",
                                        name="name_example",
                                        optional=True,
                                    ),
                                    client_cert_secret=SecretKeySelector(
                                        key="key_example",
                                        name="name_example",
                                        optional=True,
                                    ),
                                    client_key_secret=SecretKeySelector(
                                        key="key_example",
                                        name="name_example",
                                        optional=True,
                                    ),
                                    insecure_skip_verify=True,
                                ),
                                url="url_example",
                            ),
                            open_whisk=GithubComArgoprojArgoEventsPkgApisEventsV1alpha1OpenWhiskTrigger(
                                action_name="action_name_example",
                                auth_token=SecretKeySelector(
                                    key="key_example",
                                    name="name_example",
                                    optional=True,
                                ),
                                host="host_example",
                                namespace="namespace_example",
                                parameters=[
                                    GithubComArgoprojArgoEventsPkgApisEventsV1alpha1TriggerParameter(
                                        dest="dest_example",
                                        operation="operation_example",
                                        src=GithubComArgoprojArgoEventsPkgApisEventsV1alpha1TriggerParameterSource(
                                            context_key="context_key_example",
                                            context_template="context_template_example",
                                            data_key="data_key_example",
                                            data_template="data_template_example",
                                            dependency_name="dependency_name_example",
                                            use_raw_data=True,
                                            value="value_example",
                                        ),
                                    ),
                                ],
                                payload=[
                                    GithubComArgoprojArgoEventsPkgApisEventsV1alpha1TriggerParameter(
                                        dest="dest_example",
                                        operation="operation_example",
                                        src=GithubComArgoprojArgoEventsPkgApisEventsV1alpha1TriggerParameterSource(
                                            context_key="context_key_example",
                                            context_template="context_template_example",
                                            data_key="data_key_example",
                                            data_template="data_template_example",
                                            dependency_name="dependency_name_example",
                                            use_raw_data=True,
                                            value="value_example",
                                        ),
                                    ),
                                ],
                                version="version_example",
                            ),
                            pulsar=GithubComArgoprojArgoEventsPkgApisEventsV1alpha1PulsarTrigger(
                                auth_athenz_params={
                                    "key": "key_example",
                                },
                                auth_athenz_secret=SecretKeySelector(
                                    key="key_example",
                                    name="name_example",
                                    optional=True,
                                ),
                                auth_token_secret=SecretKeySelector(
                                    key="key_example",
                                    name="name_example",
                                    optional=True,
                                ),
                                connection_backoff=GithubComArgoprojArgoEventsPkgApisEventsV1alpha1Backoff(
                                    duration=GithubComArgoprojArgoEventsPkgApisEventsV1alpha1Int64OrString(
                                        int64_val="int64_val_example",
                                        str_val="str_val_example",
                                        type="type_example",
                                    ),
                                    factor=GithubComArgoprojArgoEventsPkgApisEventsV1alpha1Amount(
                                        value='YQ==',
                                    ),
                                    jitter=GithubComArgoprojArgoEventsPkgApisEventsV1alpha1Amount(
                                        value='YQ==',
                                    ),
                                    steps=1,
                                ),
                                parameters=[
                                    GithubComArgoprojArgoEventsPkgApisEventsV1alpha1TriggerParameter(
                                        dest="dest_example",
                                        operation="operation_example",
                                        src=GithubComArgoprojArgoEventsPkgApisEventsV1alpha1TriggerParameterSource(
                                            context_key="context_key_example",
                                            context_template="context_template_example",
                                            data_key="data_key_example",
                                            data_template="data_template_example",
                                            dependency_name="dependency_name_example",
                                            use_raw_data=True,
                                            value="value_example",
                                        ),
                                    ),
                                ],
                                payload=[
                                    GithubComArgoprojArgoEventsPkgApisEventsV1alpha1TriggerParameter(
                                        dest="dest_example",
                                        operation="operation_example",
                                        src=GithubComArgoprojArgoEventsPkgApisEventsV1alpha1TriggerParameterSource(
                                            context_key="context_key_example",
                                            context_template="context_template_example",
                                            data_key="data_key_example",
                                            data_template="data_template_example",
                                            dependency_name="dependency_name_example",
                                            use_raw_data=True,
                                            value="value_example",
                                        ),
                                    ),
                                ],
                                tls=GithubComArgoprojArgoEventsPkgApisEventsV1alpha1TLSConfig(
                                    ca_cert_secret=SecretKeySelector(
                                        key="key_example",
                                        name="name_example",
                                        optional=True,
                                    ),
                                    client_cert_secret=SecretKeySelector(
                                        key="key_example",
                                        name="name_example",
                                        optional=True,
                                    ),
                                    client_key_secret=SecretKeySelector(
                                        key="key_example",
                                        name="name_example",
                                        optional=True,
                                    ),
                                    insecure_skip_verify=True,
                                ),
                                tls_allow_insecure_connection=True,
                                tls_trust_certs_secret=SecretKeySelector(
                                    key="key_example",
                                    name="name_example",
                                    optional=True,
                                ),
                                tls_validate_hostname=True,
                                topic="topic_example",
                                url="url_example",
                            ),
                            slack=GithubComArgoprojArgoEventsPkgApisEventsV1alpha1SlackTrigger(
                                attachments="attachments_example",
                                blocks="blocks_example",
                                channel="channel_example",
                                message="message_example",
                                parameters=[
                                    GithubComArgoprojArgoEventsPkgApisEventsV1alpha1TriggerParameter(
                                        dest="dest_example",
                                        operation="operation_example",
                                        src=GithubComArgoprojArgoEventsPkgApisEventsV1alpha1TriggerParameterSource(
                                            context_key="context_key_example",
                                            context_template="context_template_example",
                                            data_key="data_key_example",
                                            data_template="data_template_example",
                                            dependency_name="dependency_name_example",
                                            use_raw_data=True,
                                            value="value_example",
                                        ),
                                    ),
                                ],
                                sender=GithubComArgoprojArgoEventsPkgApisEventsV1alpha1SlackSender(
                                    icon="icon_example",
                                    username="username_example",
                                ),
                                slack_token=SecretKeySelector(
                                    key="key_example",
                                    name="name_example",
                                    optional=True,
                                ),
                                thread=GithubComArgoprojArgoEventsPkgApisEventsV1alpha1SlackThread(
                                    broadcast_message_to_channel=True,
                                    message_aggregation_key="message_aggregation_key_example",
                                ),
                            ),
                        ),
                    ),
                ],
            ),
            status=GithubComArgoprojArgoEventsPkgApisEventsV1alpha1SensorStatus(
                status=GithubComArgoprojArgoEventsPkgApisEventsV1alpha1Status(
                    conditions=[
                        GithubComArgoprojArgoEventsPkgApisEventsV1alpha1Condition(
                            last_transition_time=dateutil_parser('1970-01-01T00:00:00.00Z'),
                            message="message_example",
                            reason="reason_example",
                            status="status_example",
                            type="type_example",
                        ),
                    ],
                ),
            ),
        ),
    ) # SensorCreateSensorRequest | 

    # example passing only required values which don't have defaults set
    try:
        api_response = api_instance.create_sensor(namespace, body)
        pprint(api_response)
    except argo_workflows.ApiException as e:
        print("Exception when calling SensorServiceApi->create_sensor: %s\n" % e)
```


### Parameters

Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **namespace** | **str**|  |
 **body** | [**SensorCreateSensorRequest**](SensorCreateSensorRequest.md)|  |

### Return type

[**GithubComArgoprojArgoEventsPkgApisEventsV1alpha1Sensor**](GithubComArgoprojArgoEventsPkgApisEventsV1alpha1Sensor.md)

### Authorization

[BearerToken](../README.md#BearerToken)

### HTTP request headers

 - **Content-Type**: application/json
 - **Accept**: application/json


### HTTP response details

| Status code | Description | Response headers |
|-------------|-------------|------------------|
**200** | A successful response. |  -  |
**0** | An unexpected error response. |  -  |

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

# **delete_sensor**
> bool, date, datetime, dict, float, int, list, str, none_type delete_sensor(namespace, name)



### Example

* Api Key Authentication (BearerToken):

```python
import time
import argo_workflows
from argo_workflows.api import sensor_service_api
from argo_workflows.model.grpc_gateway_runtime_error import GrpcGatewayRuntimeError
from pprint import pprint
# Defining the host is optional and defaults to http://localhost:2746
# See configuration.py for a list of all supported configuration parameters.
configuration = argo_workflows.Configuration(
    host = "http://localhost:2746"
)

# The client must configure the authentication and authorization parameters
# in accordance with the API server security policy.
# Examples for each auth method are provided below, use the example that
# satisfies your auth use case.

# Configure API key authorization: BearerToken
configuration.api_key['BearerToken'] = 'YOUR_API_KEY'

# Uncomment below to setup prefix (e.g. Bearer) for API key, if needed
# configuration.api_key_prefix['BearerToken'] = 'Bearer'

# Enter a context with an instance of the API client
with argo_workflows.ApiClient(configuration) as api_client:
    # Create an instance of the API class
    api_instance = sensor_service_api.SensorServiceApi(api_client)
    namespace = "namespace_example" # str | 
    name = "name_example" # str | 
    delete_options_grace_period_seconds = "deleteOptions.gracePeriodSeconds_example" # str | The duration in seconds before the object should be deleted. Value must be non-negative integer. The value zero indicates delete immediately. If this value is nil, the default grace period for the specified type will be used. Defaults to a per object value if not specified. zero means delete immediately. +optional. (optional)
    delete_options_preconditions_uid = "deleteOptions.preconditions.uid_example" # str | Specifies the target UID. +optional. (optional)
    delete_options_preconditions_resource_version = "deleteOptions.preconditions.resourceVersion_example" # str | Specifies the target ResourceVersion +optional. (optional)
    delete_options_orphan_dependents = True # bool | Deprecated: please use the PropagationPolicy, this field will be deprecated in 1.7. Should the dependent objects be orphaned. If true/false, the \"orphan\" finalizer will be added to/removed from the object's finalizers list. Either this field or PropagationPolicy may be set, but not both. +optional. (optional)
    delete_options_propagation_policy = "deleteOptions.propagationPolicy_example" # str | Whether and how garbage collection will be performed. Either this field or OrphanDependents may be set, but not both. The default policy is decided by the existing finalizer set in the metadata.finalizers and the resource-specific default policy. Acceptable values are: 'Orphan' - orphan the dependents; 'Background' - allow the garbage collector to delete the dependents in the background; 'Foreground' - a cascading policy that deletes all dependents in the foreground. +optional. (optional)
    delete_options_dry_run = [
        "deleteOptions.dryRun_example",
    ] # [str] | When present, indicates that modifications should not be persisted. An invalid or unrecognized dryRun directive will result in an error response and no further processing of the request. Valid values are: - All: all dry run stages will be processed +optional +listType=atomic. (optional)
    delete_options_ignore_store_read_error_with_cluster_breaking_potential = True # bool | if set to true, it will trigger an unsafe deletion of the resource in case the normal deletion flow fails with a corrupt object error. A resource is considered corrupt if it can not be retrieved from the underlying storage successfully because of a) its data can not be transformed e.g. decryption failure, or b) it fails to decode into an object. NOTE: unsafe deletion ignores finalizer constraints, skips precondition checks, and removes the object from the storage. WARNING: This may potentially break the cluster if the workload associated with the resource being unsafe-deleted relies on normal deletion flow. Use only if you REALLY know what you are doing. The default value is false, and the user must opt in to enable it +optional. (optional)

    # example passing only required values which don't have defaults set
    try:
        api_response = api_instance.delete_sensor(namespace, name)
        pprint(api_response)
    except argo_workflows.ApiException as e:
        print("Exception when calling SensorServiceApi->delete_sensor: %s\n" % e)

    # example passing only required values which don't have defaults set
    # and optional values
    try:
        api_response = api_instance.delete_sensor(namespace, name, delete_options_grace_period_seconds=delete_options_grace_period_seconds, delete_options_preconditions_uid=delete_options_preconditions_uid, delete_options_preconditions_resource_version=delete_options_preconditions_resource_version, delete_options_orphan_dependents=delete_options_orphan_dependents, delete_options_propagation_policy=delete_options_propagation_policy, delete_options_dry_run=delete_options_dry_run, delete_options_ignore_store_read_error_with_cluster_breaking_potential=delete_options_ignore_store_read_error_with_cluster_breaking_potential)
        pprint(api_response)
    except argo_workflows.ApiException as e:
        print("Exception when calling SensorServiceApi->delete_sensor: %s\n" % e)
```


### Parameters

Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **namespace** | **str**|  |
 **name** | **str**|  |
 **delete_options_grace_period_seconds** | **str**| The duration in seconds before the object should be deleted. Value must be non-negative integer. The value zero indicates delete immediately. If this value is nil, the default grace period for the specified type will be used. Defaults to a per object value if not specified. zero means delete immediately. +optional. | [optional]
 **delete_options_preconditions_uid** | **str**| Specifies the target UID. +optional. | [optional]
 **delete_options_preconditions_resource_version** | **str**| Specifies the target ResourceVersion +optional. | [optional]
 **delete_options_orphan_dependents** | **bool**| Deprecated: please use the PropagationPolicy, this field will be deprecated in 1.7. Should the dependent objects be orphaned. If true/false, the \&quot;orphan\&quot; finalizer will be added to/removed from the object&#39;s finalizers list. Either this field or PropagationPolicy may be set, but not both. +optional. | [optional]
 **delete_options_propagation_policy** | **str**| Whether and how garbage collection will be performed. Either this field or OrphanDependents may be set, but not both. The default policy is decided by the existing finalizer set in the metadata.finalizers and the resource-specific default policy. Acceptable values are: &#39;Orphan&#39; - orphan the dependents; &#39;Background&#39; - allow the garbage collector to delete the dependents in the background; &#39;Foreground&#39; - a cascading policy that deletes all dependents in the foreground. +optional. | [optional]
 **delete_options_dry_run** | **[str]**| When present, indicates that modifications should not be persisted. An invalid or unrecognized dryRun directive will result in an error response and no further processing of the request. Valid values are: - All: all dry run stages will be processed +optional +listType&#x3D;atomic. | [optional]
 **delete_options_ignore_store_read_error_with_cluster_breaking_potential** | **bool**| if set to true, it will trigger an unsafe deletion of the resource in case the normal deletion flow fails with a corrupt object error. A resource is considered corrupt if it can not be retrieved from the underlying storage successfully because of a) its data can not be transformed e.g. decryption failure, or b) it fails to decode into an object. NOTE: unsafe deletion ignores finalizer constraints, skips precondition checks, and removes the object from the storage. WARNING: This may potentially break the cluster if the workload associated with the resource being unsafe-deleted relies on normal deletion flow. Use only if you REALLY know what you are doing. The default value is false, and the user must opt in to enable it +optional. | [optional]

### Return type

**bool, date, datetime, dict, float, int, list, str, none_type**

### Authorization

[BearerToken](../README.md#BearerToken)

### HTTP request headers

 - **Content-Type**: Not defined
 - **Accept**: application/json


### HTTP response details

| Status code | Description | Response headers |
|-------------|-------------|------------------|
**200** | A successful response. |  -  |
**0** | An unexpected error response. |  -  |

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

# **get_sensor**
> GithubComArgoprojArgoEventsPkgApisEventsV1alpha1Sensor get_sensor(namespace, name)



### Example

* Api Key Authentication (BearerToken):

```python
import time
import argo_workflows
from argo_workflows.api import sensor_service_api
from argo_workflows.model.grpc_gateway_runtime_error import GrpcGatewayRuntimeError
from argo_workflows.model.github_com_argoproj_argo_events_pkg_apis_events_v1alpha1_sensor import GithubComArgoprojArgoEventsPkgApisEventsV1alpha1Sensor
from pprint import pprint
# Defining the host is optional and defaults to http://localhost:2746
# See configuration.py for a list of all supported configuration parameters.
configuration = argo_workflows.Configuration(
    host = "http://localhost:2746"
)

# The client must configure the authentication and authorization parameters
# in accordance with the API server security policy.
# Examples for each auth method are provided below, use the example that
# satisfies your auth use case.

# Configure API key authorization: BearerToken
configuration.api_key['BearerToken'] = 'YOUR_API_KEY'

# Uncomment below to setup prefix (e.g. Bearer) for API key, if needed
# configuration.api_key_prefix['BearerToken'] = 'Bearer'

# Enter a context with an instance of the API client
with argo_workflows.ApiClient(configuration) as api_client:
    # Create an instance of the API class
    api_instance = sensor_service_api.SensorServiceApi(api_client)
    namespace = "namespace_example" # str | 
    name = "name_example" # str | 
    get_options_resource_version = "getOptions.resourceVersion_example" # str | resourceVersion sets a constraint on what resource versions a request may be served from. See https://kubernetes.io/docs/reference/using-api/api-concepts/#resource-versions for details.  Defaults to unset +optional (optional)

    # example passing only required values which don't have defaults set
    try:
        api_response = api_instance.get_sensor(namespace, name)
        pprint(api_response)
    except argo_workflows.ApiException as e:
        print("Exception when calling SensorServiceApi->get_sensor: %s\n" % e)

    # example passing only required values which don't have defaults set
    # and optional values
    try:
        api_response = api_instance.get_sensor(namespace, name, get_options_resource_version=get_options_resource_version)
        pprint(api_response)
    except argo_workflows.ApiException as e:
        print("Exception when calling SensorServiceApi->get_sensor: %s\n" % e)
```


### Parameters

Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **namespace** | **str**|  |
 **name** | **str**|  |
 **get_options_resource_version** | **str**| resourceVersion sets a constraint on what resource versions a request may be served from. See https://kubernetes.io/docs/reference/using-api/api-concepts/#resource-versions for details.  Defaults to unset +optional | [optional]

### Return type

[**GithubComArgoprojArgoEventsPkgApisEventsV1alpha1Sensor**](GithubComArgoprojArgoEventsPkgApisEventsV1alpha1Sensor.md)

### Authorization

[BearerToken](../README.md#BearerToken)

### HTTP request headers

 - **Content-Type**: Not defined
 - **Accept**: application/json


### HTTP response details

| Status code | Description | Response headers |
|-------------|-------------|------------------|
**200** | A successful response. |  -  |
**0** | An unexpected error response. |  -  |

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

# **list_sensors**
> GithubComArgoprojArgoEventsPkgApisEventsV1alpha1SensorList list_sensors(namespace)



### Example

* Api Key Authentication (BearerToken):

```python
import time
import argo_workflows
from argo_workflows.api import sensor_service_api
from argo_workflows.model.grpc_gateway_runtime_error import GrpcGatewayRuntimeError
from argo_workflows.model.github_com_argoproj_argo_events_pkg_apis_events_v1alpha1_sensor_list import GithubComArgoprojArgoEventsPkgApisEventsV1alpha1SensorList
from pprint import pprint
# Defining the host is optional and defaults to http://localhost:2746
# See configuration.py for a list of all supported configuration parameters.
configuration = argo_workflows.Configuration(
    host = "http://localhost:2746"
)

# The client must configure the authentication and authorization parameters
# in accordance with the API server security policy.
# Examples for each auth method are provided below, use the example that
# satisfies your auth use case.

# Configure API key authorization: BearerToken
configuration.api_key['BearerToken'] = 'YOUR_API_KEY'

# Uncomment below to setup prefix (e.g. Bearer) for API key, if needed
# configuration.api_key_prefix['BearerToken'] = 'Bearer'

# Enter a context with an instance of the API client
with argo_workflows.ApiClient(configuration) as api_client:
    # Create an instance of the API class
    api_instance = sensor_service_api.SensorServiceApi(api_client)
    namespace = "namespace_example" # str | 
    list_options_label_selector = "listOptions.labelSelector_example" # str | A selector to restrict the list of returned objects by their labels. Defaults to everything. +optional. (optional)
    list_options_field_selector = "listOptions.fieldSelector_example" # str | A selector to restrict the list of returned objects by their fields. Defaults to everything. +optional. (optional)
    list_options_watch = True # bool | Watch for changes to the described resources and return them as a stream of add, update, and remove notifications. Specify resourceVersion. +optional. (optional)
    list_options_allow_watch_bookmarks = True # bool | allowWatchBookmarks requests watch events with type \"BOOKMARK\". Servers that do not implement bookmarks may ignore this flag and bookmarks are sent at the server's discretion. Clients should not assume bookmarks are returned at any specific interval, nor may they assume the server will send any BOOKMARK event during a session. If this is not a watch, this field is ignored. +optional. (optional)
    list_options_resource_version = "listOptions.resourceVersion_example" # str | resourceVersion sets a constraint on what resource versions a request may be served from. See https://kubernetes.io/docs/reference/using-api/api-concepts/#resource-versions for details.  Defaults to unset +optional (optional)
    list_options_resource_version_match = "listOptions.resourceVersionMatch_example" # str | resourceVersionMatch determines how resourceVersion is applied to list calls. It is highly recommended that resourceVersionMatch be set for list calls where resourceVersion is set See https://kubernetes.io/docs/reference/using-api/api-concepts/#resource-versions for details.  Defaults to unset +optional (optional)
    list_options_timeout_seconds = "listOptions.timeoutSeconds_example" # str | Timeout for the list/watch call. This limits the duration of the call, regardless of any activity or inactivity. +optional. (optional)
    list_options_limit = "listOptions.limit_example" # str | limit is a maximum number of responses to return for a list call. If more items exist, the server will set the `continue` field on the list metadata to a value that can be used with the same initial query to retrieve the next set of results. Setting a limit may return fewer than the requested amount of items (up to zero items) in the event all requested objects are filtered out and clients should only use the presence of the continue field to determine whether more results are available. Servers may choose not to support the limit argument and will return all of the available results. If limit is specified and the continue field is empty, clients may assume that no more results are available. This field is not supported if watch is true.  The server guarantees that the objects returned when using continue will be identical to issuing a single list call without a limit - that is, no objects created, modified, or deleted after the first request is issued will be included in any subsequent continued requests. This is sometimes referred to as a consistent snapshot, and ensures that a client that is using limit to receive smaller chunks of a very large result can ensure they see all possible objects. If objects are updated during a chunked list the version of the object that was present at the time the first list result was calculated is returned. (optional)
    list_options_continue = "listOptions.continue_example" # str | The continue option should be set when retrieving more results from the server. Since this value is server defined, clients may only use the continue value from a previous query result with identical query parameters (except for the value of continue) and the server may reject a continue value it does not recognize. If the specified continue value is no longer valid whether due to expiration (generally five to fifteen minutes) or a configuration change on the server, the server will respond with a 410 ResourceExpired error together with a continue token. If the client needs a consistent list, it must restart their list without the continue field. Otherwise, the client may send another list request with the token received with the 410 error, the server will respond with a list starting from the next key, but from the latest snapshot, which is inconsistent from the previous list results - objects that are created, modified, or deleted after the first list request will be included in the response, as long as their keys are after the \"next key\".  This field is not supported when watch is true. Clients may start a watch from the last resourceVersion value returned by the server and not miss any modifications. (optional)
    list_options_send_initial_events = True # bool | `sendInitialEvents=true` may be set together with `watch=true`. In that case, the watch stream will begin with synthetic events to produce the current state of objects in the collection. Once all such events have been sent, a synthetic \"Bookmark\" event  will be sent. The bookmark will report the ResourceVersion (RV) corresponding to the set of objects, and be marked with `\"io.k8s.initial-events-end\": \"true\"` annotation. Afterwards, the watch stream will proceed as usual, sending watch events corresponding to changes (subsequent to the RV) to objects watched.  When `sendInitialEvents` option is set, we require `resourceVersionMatch` option to also be set. The semantic of the watch request is as following: - `resourceVersionMatch` = NotOlderThan   is interpreted as \"data at least as new as the provided `resourceVersion`\"   and the bookmark event is send when the state is synced   to a `resourceVersion` at least as fresh as the one provided by the ListOptions.   If `resourceVersion` is unset, this is interpreted as \"consistent read\" and the   bookmark event is send when the state is synced at least to the moment   when request started being processed. - `resourceVersionMatch` set to any other value or unset   Invalid error is returned.  Defaults to true if `resourceVersion=\"\"` or `resourceVersion=\"0\"` (for backward compatibility reasons) and to false otherwise. +optional (optional)

    # example passing only required values which don't have defaults set
    try:
        api_response = api_instance.list_sensors(namespace)
        pprint(api_response)
    except argo_workflows.ApiException as e:
        print("Exception when calling SensorServiceApi->list_sensors: %s\n" % e)

    # example passing only required values which don't have defaults set
    # and optional values
    try:
        api_response = api_instance.list_sensors(namespace, list_options_label_selector=list_options_label_selector, list_options_field_selector=list_options_field_selector, list_options_watch=list_options_watch, list_options_allow_watch_bookmarks=list_options_allow_watch_bookmarks, list_options_resource_version=list_options_resource_version, list_options_resource_version_match=list_options_resource_version_match, list_options_timeout_seconds=list_options_timeout_seconds, list_options_limit=list_options_limit, list_options_continue=list_options_continue, list_options_send_initial_events=list_options_send_initial_events)
        pprint(api_response)
    except argo_workflows.ApiException as e:
        print("Exception when calling SensorServiceApi->list_sensors: %s\n" % e)
```


### Parameters

Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **namespace** | **str**|  |
 **list_options_label_selector** | **str**| A selector to restrict the list of returned objects by their labels. Defaults to everything. +optional. | [optional]
 **list_options_field_selector** | **str**| A selector to restrict the list of returned objects by their fields. Defaults to everything. +optional. | [optional]
 **list_options_watch** | **bool**| Watch for changes to the described resources and return them as a stream of add, update, and remove notifications. Specify resourceVersion. +optional. | [optional]
 **list_options_allow_watch_bookmarks** | **bool**| allowWatchBookmarks requests watch events with type \&quot;BOOKMARK\&quot;. Servers that do not implement bookmarks may ignore this flag and bookmarks are sent at the server&#39;s discretion. Clients should not assume bookmarks are returned at any specific interval, nor may they assume the server will send any BOOKMARK event during a session. If this is not a watch, this field is ignored. +optional. | [optional]
 **list_options_resource_version** | **str**| resourceVersion sets a constraint on what resource versions a request may be served from. See https://kubernetes.io/docs/reference/using-api/api-concepts/#resource-versions for details.  Defaults to unset +optional | [optional]
 **list_options_resource_version_match** | **str**| resourceVersionMatch determines how resourceVersion is applied to list calls. It is highly recommended that resourceVersionMatch be set for list calls where resourceVersion is set See https://kubernetes.io/docs/reference/using-api/api-concepts/#resource-versions for details.  Defaults to unset +optional | [optional]
 **list_options_timeout_seconds** | **str**| Timeout for the list/watch call. This limits the duration of the call, regardless of any activity or inactivity. +optional. | [optional]
 **list_options_limit** | **str**| limit is a maximum number of responses to return for a list call. If more items exist, the server will set the &#x60;continue&#x60; field on the list metadata to a value that can be used with the same initial query to retrieve the next set of results. Setting a limit may return fewer than the requested amount of items (up to zero items) in the event all requested objects are filtered out and clients should only use the presence of the continue field to determine whether more results are available. Servers may choose not to support the limit argument and will return all of the available results. If limit is specified and the continue field is empty, clients may assume that no more results are available. This field is not supported if watch is true.  The server guarantees that the objects returned when using continue will be identical to issuing a single list call without a limit - that is, no objects created, modified, or deleted after the first request is issued will be included in any subsequent continued requests. This is sometimes referred to as a consistent snapshot, and ensures that a client that is using limit to receive smaller chunks of a very large result can ensure they see all possible objects. If objects are updated during a chunked list the version of the object that was present at the time the first list result was calculated is returned. | [optional]
 **list_options_continue** | **str**| The continue option should be set when retrieving more results from the server. Since this value is server defined, clients may only use the continue value from a previous query result with identical query parameters (except for the value of continue) and the server may reject a continue value it does not recognize. If the specified continue value is no longer valid whether due to expiration (generally five to fifteen minutes) or a configuration change on the server, the server will respond with a 410 ResourceExpired error together with a continue token. If the client needs a consistent list, it must restart their list without the continue field. Otherwise, the client may send another list request with the token received with the 410 error, the server will respond with a list starting from the next key, but from the latest snapshot, which is inconsistent from the previous list results - objects that are created, modified, or deleted after the first list request will be included in the response, as long as their keys are after the \&quot;next key\&quot;.  This field is not supported when watch is true. Clients may start a watch from the last resourceVersion value returned by the server and not miss any modifications. | [optional]
 **list_options_send_initial_events** | **bool**| &#x60;sendInitialEvents&#x3D;true&#x60; may be set together with &#x60;watch&#x3D;true&#x60;. In that case, the watch stream will begin with synthetic events to produce the current state of objects in the collection. Once all such events have been sent, a synthetic \&quot;Bookmark\&quot; event  will be sent. The bookmark will report the ResourceVersion (RV) corresponding to the set of objects, and be marked with &#x60;\&quot;io.k8s.initial-events-end\&quot;: \&quot;true\&quot;&#x60; annotation. Afterwards, the watch stream will proceed as usual, sending watch events corresponding to changes (subsequent to the RV) to objects watched.  When &#x60;sendInitialEvents&#x60; option is set, we require &#x60;resourceVersionMatch&#x60; option to also be set. The semantic of the watch request is as following: - &#x60;resourceVersionMatch&#x60; &#x3D; NotOlderThan   is interpreted as \&quot;data at least as new as the provided &#x60;resourceVersion&#x60;\&quot;   and the bookmark event is send when the state is synced   to a &#x60;resourceVersion&#x60; at least as fresh as the one provided by the ListOptions.   If &#x60;resourceVersion&#x60; is unset, this is interpreted as \&quot;consistent read\&quot; and the   bookmark event is send when the state is synced at least to the moment   when request started being processed. - &#x60;resourceVersionMatch&#x60; set to any other value or unset   Invalid error is returned.  Defaults to true if &#x60;resourceVersion&#x3D;\&quot;\&quot;&#x60; or &#x60;resourceVersion&#x3D;\&quot;0\&quot;&#x60; (for backward compatibility reasons) and to false otherwise. +optional | [optional]

### Return type

[**GithubComArgoprojArgoEventsPkgApisEventsV1alpha1SensorList**](GithubComArgoprojArgoEventsPkgApisEventsV1alpha1SensorList.md)

### Authorization

[BearerToken](../README.md#BearerToken)

### HTTP request headers

 - **Content-Type**: Not defined
 - **Accept**: application/json


### HTTP response details

| Status code | Description | Response headers |
|-------------|-------------|------------------|
**200** | A successful response. |  -  |
**0** | An unexpected error response. |  -  |

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

# **sensors_logs**
> StreamResultOfSensorLogEntry sensors_logs(namespace)



### Example

* Api Key Authentication (BearerToken):

```python
import time
import argo_workflows
from argo_workflows.api import sensor_service_api
from argo_workflows.model.grpc_gateway_runtime_error import GrpcGatewayRuntimeError
from argo_workflows.model.stream_result_of_sensor_log_entry import StreamResultOfSensorLogEntry
from pprint import pprint
# Defining the host is optional and defaults to http://localhost:2746
# See configuration.py for a list of all supported configuration parameters.
configuration = argo_workflows.Configuration(
    host = "http://localhost:2746"
)

# The client must configure the authentication and authorization parameters
# in accordance with the API server security policy.
# Examples for each auth method are provided below, use the example that
# satisfies your auth use case.

# Configure API key authorization: BearerToken
configuration.api_key['BearerToken'] = 'YOUR_API_KEY'

# Uncomment below to setup prefix (e.g. Bearer) for API key, if needed
# configuration.api_key_prefix['BearerToken'] = 'Bearer'

# Enter a context with an instance of the API client
with argo_workflows.ApiClient(configuration) as api_client:
    # Create an instance of the API class
    api_instance = sensor_service_api.SensorServiceApi(api_client)
    namespace = "namespace_example" # str | 
    name = "name_example" # str | optional - only return entries for this sensor name. (optional)
    trigger_name = "triggerName_example" # str | optional - only return entries for this trigger. (optional)
    grep = "grep_example" # str | option - only return entries where `msg` contains this regular expressions. (optional)
    pod_log_options_container = "podLogOptions.container_example" # str | The container for which to stream logs. Defaults to only container if there is one container in the pod. +optional. (optional)
    pod_log_options_follow = True # bool | Follow the log stream of the pod. Defaults to false. +optional. (optional)
    pod_log_options_previous = True # bool | Return previous terminated container logs. Defaults to false. +optional. (optional)
    pod_log_options_since_seconds = "podLogOptions.sinceSeconds_example" # str | A relative time in seconds before the current time from which to show logs. If this value precedes the time a pod was started, only logs since the pod start will be returned. If this value is in the future, no logs will be returned. Only one of sinceSeconds or sinceTime may be specified. +optional. (optional)
    pod_log_options_since_time_seconds = "podLogOptions.sinceTime.seconds_example" # str | Represents seconds of UTC time since Unix epoch 1970-01-01T00:00:00Z. Must be from 0001-01-01T00:00:00Z to 9999-12-31T23:59:59Z inclusive. (optional)
    pod_log_options_since_time_nanos = 1 # int | Non-negative fractions of a second at nanosecond resolution. Negative second values with fractions must still have non-negative nanos values that count forward in time. Must be from 0 to 999,999,999 inclusive. This field may be limited in precision depending on context. (optional)
    pod_log_options_timestamps = True # bool | If true, add an RFC3339 or RFC3339Nano timestamp at the beginning of every line of log output. Defaults to false. +optional. (optional)
    pod_log_options_tail_lines = "podLogOptions.tailLines_example" # str | If set, the number of lines from the end of the logs to show. If not specified, logs are shown from the creation of the container or sinceSeconds or sinceTime. Note that when \"TailLines\" is specified, \"Stream\" can only be set to nil or \"All\". +optional. (optional)
    pod_log_options_limit_bytes = "podLogOptions.limitBytes_example" # str | If set, the number of bytes to read from the server before terminating the log output. This may not display a complete final line of logging, and may return slightly more or slightly less than the specified limit. +optional. (optional)
    pod_log_options_insecure_skip_tls_verify_backend = True # bool | insecureSkipTLSVerifyBackend indicates that the apiserver should not confirm the validity of the serving certificate of the backend it is connecting to.  This will make the HTTPS connection between the apiserver and the backend insecure. This means the apiserver cannot verify the log data it is receiving came from the real kubelet.  If the kubelet is configured to verify the apiserver's TLS credentials, it does not mean the connection to the real kubelet is vulnerable to a man in the middle attack (e.g. an attacker could not intercept the actual log data coming from the real kubelet). +optional. (optional)
    pod_log_options_stream = "podLogOptions.stream_example" # str | Specify which container log stream to return to the client. Acceptable values are \"All\", \"Stdout\" and \"Stderr\". If not specified, \"All\" is used, and both stdout and stderr are returned interleaved. Note that when \"TailLines\" is specified, \"Stream\" can only be set to nil or \"All\". +featureGate=PodLogsQuerySplitStreams +optional. (optional)

    # example passing only required values which don't have defaults set
    try:
        api_response = api_instance.sensors_logs(namespace)
        pprint(api_response)
    except argo_workflows.ApiException as e:
        print("Exception when calling SensorServiceApi->sensors_logs: %s\n" % e)

    # example passing only required values which don't have defaults set
    # and optional values
    try:
        api_response = api_instance.sensors_logs(namespace, name=name, trigger_name=trigger_name, grep=grep, pod_log_options_container=pod_log_options_container, pod_log_options_follow=pod_log_options_follow, pod_log_options_previous=pod_log_options_previous, pod_log_options_since_seconds=pod_log_options_since_seconds, pod_log_options_since_time_seconds=pod_log_options_since_time_seconds, pod_log_options_since_time_nanos=pod_log_options_since_time_nanos, pod_log_options_timestamps=pod_log_options_timestamps, pod_log_options_tail_lines=pod_log_options_tail_lines, pod_log_options_limit_bytes=pod_log_options_limit_bytes, pod_log_options_insecure_skip_tls_verify_backend=pod_log_options_insecure_skip_tls_verify_backend, pod_log_options_stream=pod_log_options_stream)
        pprint(api_response)
    except argo_workflows.ApiException as e:
        print("Exception when calling SensorServiceApi->sensors_logs: %s\n" % e)
```


### Parameters

Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **namespace** | **str**|  |
 **name** | **str**| optional - only return entries for this sensor name. | [optional]
 **trigger_name** | **str**| optional - only return entries for this trigger. | [optional]
 **grep** | **str**| option - only return entries where &#x60;msg&#x60; contains this regular expressions. | [optional]
 **pod_log_options_container** | **str**| The container for which to stream logs. Defaults to only container if there is one container in the pod. +optional. | [optional]
 **pod_log_options_follow** | **bool**| Follow the log stream of the pod. Defaults to false. +optional. | [optional]
 **pod_log_options_previous** | **bool**| Return previous terminated container logs. Defaults to false. +optional. | [optional]
 **pod_log_options_since_seconds** | **str**| A relative time in seconds before the current time from which to show logs. If this value precedes the time a pod was started, only logs since the pod start will be returned. If this value is in the future, no logs will be returned. Only one of sinceSeconds or sinceTime may be specified. +optional. | [optional]
 **pod_log_options_since_time_seconds** | **str**| Represents seconds of UTC time since Unix epoch 1970-01-01T00:00:00Z. Must be from 0001-01-01T00:00:00Z to 9999-12-31T23:59:59Z inclusive. | [optional]
 **pod_log_options_since_time_nanos** | **int**| Non-negative fractions of a second at nanosecond resolution. Negative second values with fractions must still have non-negative nanos values that count forward in time. Must be from 0 to 999,999,999 inclusive. This field may be limited in precision depending on context. | [optional]
 **pod_log_options_timestamps** | **bool**| If true, add an RFC3339 or RFC3339Nano timestamp at the beginning of every line of log output. Defaults to false. +optional. | [optional]
 **pod_log_options_tail_lines** | **str**| If set, the number of lines from the end of the logs to show. If not specified, logs are shown from the creation of the container or sinceSeconds or sinceTime. Note that when \&quot;TailLines\&quot; is specified, \&quot;Stream\&quot; can only be set to nil or \&quot;All\&quot;. +optional. | [optional]
 **pod_log_options_limit_bytes** | **str**| If set, the number of bytes to read from the server before terminating the log output. This may not display a complete final line of logging, and may return slightly more or slightly less than the specified limit. +optional. | [optional]
 **pod_log_options_insecure_skip_tls_verify_backend** | **bool**| insecureSkipTLSVerifyBackend indicates that the apiserver should not confirm the validity of the serving certificate of the backend it is connecting to.  This will make the HTTPS connection between the apiserver and the backend insecure. This means the apiserver cannot verify the log data it is receiving came from the real kubelet.  If the kubelet is configured to verify the apiserver&#39;s TLS credentials, it does not mean the connection to the real kubelet is vulnerable to a man in the middle attack (e.g. an attacker could not intercept the actual log data coming from the real kubelet). +optional. | [optional]
 **pod_log_options_stream** | **str**| Specify which container log stream to return to the client. Acceptable values are \&quot;All\&quot;, \&quot;Stdout\&quot; and \&quot;Stderr\&quot;. If not specified, \&quot;All\&quot; is used, and both stdout and stderr are returned interleaved. Note that when \&quot;TailLines\&quot; is specified, \&quot;Stream\&quot; can only be set to nil or \&quot;All\&quot;. +featureGate&#x3D;PodLogsQuerySplitStreams +optional. | [optional]

### Return type

[**StreamResultOfSensorLogEntry**](StreamResultOfSensorLogEntry.md)

### Authorization

[BearerToken](../README.md#BearerToken)

### HTTP request headers

 - **Content-Type**: Not defined
 - **Accept**: application/json


### HTTP response details

| Status code | Description | Response headers |
|-------------|-------------|------------------|
**200** | A successful response.(streaming responses) |  -  |
**0** | An unexpected error response. |  -  |

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

# **update_sensor**
> GithubComArgoprojArgoEventsPkgApisEventsV1alpha1Sensor update_sensor(namespace, name, body)



### Example

* Api Key Authentication (BearerToken):

```python
import time
import argo_workflows
from argo_workflows.api import sensor_service_api
from argo_workflows.model.grpc_gateway_runtime_error import GrpcGatewayRuntimeError
from argo_workflows.model.github_com_argoproj_argo_events_pkg_apis_events_v1alpha1_sensor import GithubComArgoprojArgoEventsPkgApisEventsV1alpha1Sensor
from argo_workflows.model.sensor_update_sensor_request import SensorUpdateSensorRequest
from pprint import pprint
# Defining the host is optional and defaults to http://localhost:2746
# See configuration.py for a list of all supported configuration parameters.
configuration = argo_workflows.Configuration(
    host = "http://localhost:2746"
)

# The client must configure the authentication and authorization parameters
# in accordance with the API server security policy.
# Examples for each auth method are provided below, use the example that
# satisfies your auth use case.

# Configure API key authorization: BearerToken
configuration.api_key['BearerToken'] = 'YOUR_API_KEY'

# Uncomment below to setup prefix (e.g. Bearer) for API key, if needed
# configuration.api_key_prefix['BearerToken'] = 'Bearer'

# Enter a context with an instance of the API client
with argo_workflows.ApiClient(configuration) as api_client:
    # Create an instance of the API class
    api_instance = sensor_service_api.SensorServiceApi(api_client)
    namespace = "namespace_example" # str | 
    name = "name_example" # str | 
    body = SensorUpdateSensorRequest(
        name="name_example",
        namespace="namespace_example",
        sensor=GithubComArgoprojArgoEventsPkgApisEventsV1alpha1Sensor(
            metadata=ObjectMeta(
                annotations={
                    "key": "key_example",
                },
                creation_timestamp=dateutil_parser('1970-01-01T00:00:00.00Z'),
                deletion_grace_period_seconds=1,
                deletion_timestamp=dateutil_parser('1970-01-01T00:00:00.00Z'),
                finalizers=[
                    "finalizers_example",
                ],
                generate_name="generate_name_example",
                generation=1,
                labels={
                    "key": "key_example",
                },
                managed_fields=[
                    ManagedFieldsEntry(
                        api_version="api_version_example",
                        fields_type="fields_type_example",
                        fields_v1={},
                        manager="manager_example",
                        operation="operation_example",
                        subresource="subresource_example",
                        time=dateutil_parser('1970-01-01T00:00:00.00Z'),
                    ),
                ],
                name="name_example",
                namespace="namespace_example",
                owner_references=[
                    OwnerReference(
                        api_version="api_version_example",
                        block_owner_deletion=True,
                        controller=True,
                        kind="kind_example",
                        name="name_example",
                        uid="uid_example",
                    ),
                ],
                resource_version="resource_version_example",
                self_link="self_link_example",
                uid="uid_example",
            ),
            spec=GithubComArgoprojArgoEventsPkgApisEventsV1alpha1SensorSpec(
                dependencies=[
                    GithubComArgoprojArgoEventsPkgApisEventsV1alpha1EventDependency(
                        event_name="event_name_example",
                        event_source_name="event_source_name_example",
                        filters=GithubComArgoprojArgoEventsPkgApisEventsV1alpha1EventDependencyFilter(
                            context=GithubComArgoprojArgoEventsPkgApisEventsV1alpha1EventContext(
                                datacontenttype="datacontenttype_example",
                                id="id_example",
                                source="source_example",
                                specversion="specversion_example",
                                subject="subject_example",
                                time=dateutil_parser('1970-01-01T00:00:00.00Z'),
                                type="type_example",
                            ),
                            data=[
                                GithubComArgoprojArgoEventsPkgApisEventsV1alpha1DataFilter(
                                    comparator="comparator_example",
                                    path="path_example",
                                    template="template_example",
                                    type="type_example",
                                    value=[
                                        "value_example",
                                    ],
                                ),
                            ],
                            data_logical_operator="data_logical_operator_example",
                            expr_logical_operator="expr_logical_operator_example",
                            exprs=[
                                GithubComArgoprojArgoEventsPkgApisEventsV1alpha1ExprFilter(
                                    expr="expr_example",
                                    fields=[
                                        GithubComArgoprojArgoEventsPkgApisEventsV1alpha1PayloadField(
                                            name="name_example",
                                            path="path_example",
                                        ),
                                    ],
                                ),
                            ],
                            script="script_example",
                            time=GithubComArgoprojArgoEventsPkgApisEventsV1alpha1TimeFilter(
                                start="start_example",
                                stop="stop_example",
                            ),
                        ),
                        filters_logical_operator="filters_logical_operator_example",
                        name="name_example",
                        transform=GithubComArgoprojArgoEventsPkgApisEventsV1alpha1EventDependencyTransformer(
                            jq="jq_example",
                            script="script_example",
                        ),
                    ),
                ],
                error_on_failed_round=True,
                event_bus_name="event_bus_name_example",
                logging_fields={
                    "key": "key_example",
                },
                replicas=1,
                revision_history_limit=1,
                template=GithubComArgoprojArgoEventsPkgApisEventsV1alpha1Template(
                    affinity=Affinity(
                        node_affinity=NodeAffinity(
                            preferred_during_scheduling_ignored_during_execution=[
                                PreferredSchedulingTerm(
                                    preference=NodeSelectorTerm(
                                        match_expressions=[
                                            NodeSelectorRequirement(
                                                key="key_example",
                                                operator="operator_example",
                                                values=[
                                                    "values_example",
                                                ],
                                            ),
                                        ],
                                        match_fields=[
                                            NodeSelectorRequirement(
                                                key="key_example",
                                                operator="operator_example",
                                                values=[
                                                    "values_example",
                                                ],
                                            ),
                                        ],
                                    ),
                                    weight=1,
                                ),
                            ],
                            required_during_scheduling_ignored_during_execution=NodeSelector(
                                node_selector_terms=[
                                    NodeSelectorTerm(
                                        match_expressions=[
                                            NodeSelectorRequirement(
                                                key="key_example",
                                                operator="operator_example",
                                                values=[
                                                    "values_example",
                                                ],
                                            ),
                                        ],
                                        match_fields=[
                                            NodeSelectorRequirement(
                                                key="key_example",
                                                operator="operator_example",
                                                values=[
                                                    "values_example",
                                                ],
                                            ),
                                        ],
                                    ),
                                ],
                            ),
                        ),
                        pod_affinity=PodAffinity(
                            preferred_during_scheduling_ignored_during_execution=[
                                WeightedPodAffinityTerm(
                                    pod_affinity_term=PodAffinityTerm(
                                        label_selector=LabelSelector(
                                            match_expressions=[
                                                LabelSelectorRequirement(
                                                    key="key_example",
                                                    operator="operator_example",
                                                    values=[
                                                        "values_example",
                                                    ],
                                                ),
                                            ],
                                            match_labels={
                                                "key": "key_example",
                                            },
                                        ),
                                        match_label_keys=[
                                            "match_label_keys_example",
                                        ],
                                        mismatch_label_keys=[
                                            "mismatch_label_keys_example",
                                        ],
                                        namespace_selector=LabelSelector(
                                            match_expressions=[
                                                LabelSelectorRequirement(
                                                    key="key_example",
                                                    operator="operator_example",
                                                    values=[
                                                        "values_example",
                                                    ],
                                                ),
                                            ],
                                            match_labels={
                                                "key": "key_example",
                                            },
                                        ),
                                        namespaces=[
                                            "namespaces_example",
                                        ],
                                        topology_key="topology_key_example",
                                    ),
                                    weight=1,
                                ),
                            ],
                            required_during_scheduling_ignored_during_execution=[
                                PodAffinityTerm(
                                    label_selector=LabelSelector(
                                        match_expressions=[
                                            LabelSelectorRequirement(
                                                key="key_example",
                                                operator="operator_example",
                                                values=[
                                                    "values_example",
                                                ],
                                            ),
                                        ],
                                        match_labels={
                                            "key": "key_example",
                                        },
                                    ),
                                    match_label_keys=[
                                        "match_label_keys_example",
                                    ],
                                    mismatch_label_keys=[
                                        "mismatch_label_keys_example",
                                    ],
                                    namespace_selector=LabelSelector(
                                        match_expressions=[
                                            LabelSelectorRequirement(
                                                key="key_example",
                                                operator="operator_example",
                                                values=[
                                                    "values_example",
                                                ],
                                            ),
                                        ],
                                        match_labels={
                                            "key": "key_example",
                                        },
                                    ),
                                    namespaces=[
                                        "namespaces_example",
                                    ],
                                    topology_key="topology_key_example",
                                ),
                            ],
                        ),
                        pod_anti_affinity=PodAntiAffinity(
                            preferred_during_scheduling_ignored_during_execution=[
                                WeightedPodAffinityTerm(
                                    pod_affinity_term=PodAffinityTerm(
                                        label_selector=LabelSelector(
                                            match_expressions=[
                                                LabelSelectorRequirement(
                                                    key="key_example",
                                                    operator="operator_example",
                                                    values=[
                                                        "values_example",
                                                    ],
                                                ),
                                            ],
                                            match_labels={
                                                "key": "key_example",
                                            },
                                        ),
                                        match_label_keys=[
                                            "match_label_keys_example",
                                        ],
                                        mismatch_label_keys=[
                                            "mismatch_label_keys_example",
                                        ],
                                        namespace_selector=LabelSelector(
                                            match_expressions=[
                                                LabelSelectorRequirement(
                                                    key="key_example",
                                                    operator="operator_example",
                                                    values=[
                                                        "values_example",
                                                    ],
                                                ),
                                            ],
                                            match_labels={
                                                "key": "key_example",
                                            },
                                        ),
                                        namespaces=[
                                            "namespaces_example",
                                        ],
                                        topology_key="topology_key_example",
                                    ),
                                    weight=1,
                                ),
                            ],
                            required_during_scheduling_ignored_during_execution=[
                                PodAffinityTerm(
                                    label_selector=LabelSelector(
                                        match_expressions=[
                                            LabelSelectorRequirement(
                                                key="key_example",
                                                operator="operator_example",
                                                values=[
                                                    "values_example",
                                                ],
                                            ),
                                        ],
                                        match_labels={
                                            "key": "key_example",
                                        },
                                    ),
                                    match_label_keys=[
                                        "match_label_keys_example",
                                    ],
                                    mismatch_label_keys=[
                                        "mismatch_label_keys_example",
                                    ],
                                    namespace_selector=LabelSelector(
                                        match_expressions=[
                                            LabelSelectorRequirement(
                                                key="key_example",
                                                operator="operator_example",
                                                values=[
                                                    "values_example",
                                                ],
                                            ),
                                        ],
                                        match_labels={
                                            "key": "key_example",
                                        },
                                    ),
                                    namespaces=[
                                        "namespaces_example",
                                    ],
                                    topology_key="topology_key_example",
                                ),
                            ],
                        ),
                    ),
                    container=GithubComArgoprojArgoEventsPkgApisEventsV1alpha1Container(
                        env=[
                            EnvVar(
                                name="name_example",
                                value="value_example",
                                value_from=EnvVarSource(
                                    config_map_key_ref=ConfigMapKeySelector(
                                        key="key_example",
                                        name="name_example",
                                        optional=True,
                                    ),
                                    field_ref=ObjectFieldSelector(
                                        api_version="api_version_example",
                                        field_path="field_path_example",
                                    ),
                                    resource_field_ref=ResourceFieldSelector(
                                        container_name="container_name_example",
                                        divisor="divisor_example",
                                        resource="resource_example",
                                    ),
                                    secret_key_ref=SecretKeySelector(
                                        key="key_example",
                                        name="name_example",
                                        optional=True,
                                    ),
                                ),
                            ),
                        ],
                        env_from=[
                            EnvFromSource(
                                config_map_ref=ConfigMapEnvSource(
                                    name="name_example",
                                    optional=True,
                                ),
                                prefix="prefix_example",
                                secret_ref=SecretEnvSource(
                                    name="name_example",
                                    optional=True,
                                ),
                            ),
                        ],
                        image_pull_policy="image_pull_policy_example",
                        resources=ResourceRequirements(
                            claims=[
                                ResourceClaim(
                                    name="name_example",
                                    request="request_example",
                                ),
                            ],
                            limits={
                                "key": "key_example",
                            },
                            requests={
                                "key": "key_example",
                            },
                        ),
                        security_context=SecurityContext(
                            allow_privilege_escalation=True,
                            app_armor_profile=AppArmorProfile(
                                localhost_profile="localhost_profile_example",
                                type="type_example",
                            ),
                            capabilities=Capabilities(
                                add=[
                                    "add_example",
                                ],
                                drop=[
                                    "drop_example",
                                ],
                            ),
                            privileged=True,
                            proc_mount="proc_mount_example",
                            read_only_root_filesystem=True,
                            run_as_group=1,
                            run_as_non_root=True,
                            run_as_user=1,
                            se_linux_options=SELinuxOptions(
                                level="level_example",
                                role="role_example",
                                type="type_example",
                                user="user_example",
                            ),
                            seccomp_profile=SeccompProfile(
                                localhost_profile="localhost_profile_example",
                                type="type_example",
                            ),
                            windows_options=WindowsSecurityContextOptions(
                                gmsa_credential_spec="gmsa_credential_spec_example",
                                gmsa_credential_spec_name="gmsa_credential_spec_name_example",
                                host_process=True,
                                run_as_user_name="run_as_user_name_example",
                            ),
                        ),
                        volume_mounts=[
                            VolumeMount(
                                mount_path="mount_path_example",
                                mount_propagation="mount_propagation_example",
                                name="name_example",
                                read_only=True,
                                recursive_read_only="recursive_read_only_example",
                                sub_path="sub_path_example",
                                sub_path_expr="sub_path_expr_example",
                            ),
                        ],
                    ),
                    image_pull_secrets=[
                        LocalObjectReference(
                            name="name_example",
                        ),
                    ],
                    metadata=GithubComArgoprojArgoEventsPkgApisEventsV1alpha1Metadata(
                        annotations={
                            "key": "key_example",
                        },
                        labels={
                            "key": "key_example",
                        },
                    ),
                    node_selector={
                        "key": "key_example",
                    },
                    priority=1,
                    priority_class_name="priority_class_name_example",
                    security_context=PodSecurityContext(
                        app_armor_profile=AppArmorProfile(
                            localhost_profile="localhost_profile_example",
                            type="type_example",
                        ),
                        fs_group=1,
                        fs_group_change_policy="fs_group_change_policy_example",
                        run_as_group=1,
                        run_as_non_root=True,
                        run_as_user=1,
                        se_linux_change_policy="se_linux_change_policy_example",
                        se_linux_options=SELinuxOptions(
                            level="level_example",
                            role="role_example",
                            type="type_example",
                            user="user_example",
                        ),
                        seccomp_profile=SeccompProfile(
                            localhost_profile="localhost_profile_example",
                            type="type_example",
                        ),
                        supplemental_groups=[
                            1,
                        ],
                        supplemental_groups_policy="supplemental_groups_policy_example",
                        sysctls=[
                            Sysctl(
                                name="name_example",
                                value="value_example",
                            ),
                        ],
                        windows_options=WindowsSecurityContextOptions(
                            gmsa_credential_spec="gmsa_credential_spec_example",
                            gmsa_credential_spec_name="gmsa_credential_spec_name_example",
                            host_process=True,
                            run_as_user_name="run_as_user_name_example",
                        ),
                    ),
                    service_account_name="service_account_name_example",
                    tolerations=[
                        Toleration(
                            effect="effect_example",
                            key="key_example",
                            operator="operator_example",
                            toleration_seconds=1,
                            value="value_example",
                        ),
                    ],
                    volumes=[
                        Volume(
                            aws_elastic_block_store=AWSElasticBlockStoreVolumeSource(
                                fs_type="fs_type_example",
                                partition=1,
                                read_only=True,
                                volume_id="volume_id_example",
                            ),
                            azure_disk=AzureDiskVolumeSource(
                                caching_mode="caching_mode_example",
                                disk_name="disk_name_example",
                                disk_uri="disk_uri_example",
                                fs_type="fs_type_example",
                                kind="kind_example",
                                read_only=True,
                            ),
                            azure_file=AzureFileVolumeSource(
                                read_only=True,
                                secret_name="secret_name_example",
                                share_name="share_name_example",
                            ),
                            cephfs=CephFSVolumeSource(
                                monitors=[
                                    "monitors_example",
                                ],
                                path="path_example",
                                read_only=True,
                                secret_file="secret_file_example",
                                secret_ref=LocalObjectReference(
                                    name="name_example",
                                ),
                                user="user_example",
                            ),
                            cinder=CinderVolumeSource(
                                fs_type="fs_type_example",
                                read_only=True,
                                secret_ref=LocalObjectReference(
                                    name="name_example",
                                ),
                                volume_id="volume_id_example",
                            ),
                            config_map=ConfigMapVolumeSource(
                                default_mode=1,
                                items=[
                                    KeyToPath(
                                        key="key_example",
                                        mode=1,
                                        path="path_example",
                                    ),
                                ],
                                name="name_example",
                                optional=True,
                            ),
                            csi=CSIVolumeSource(
                                driver="driver_example",
                                fs_type="fs_type_example",
                                node_publish_secret_ref=LocalObjectReference(
                                    name="name_example",
                                ),
                                read_only=True,
                                volume_attributes={
                                    "key": "key_example",
                                },
                            ),
                            downward_api=DownwardAPIVolumeSource(
                                default_mode=1,
                                items=[
                                    DownwardAPIVolumeFile(
                                        field_ref=ObjectFieldSelector(
                                            api_version="api_version_example",
                                            field_path="field_path_example",
                                        ),
                                        mode=1,
                                        path="path_example",
                                        resource_field_ref=ResourceFieldSelector(
                                            container_name="container_name_example",
                                            divisor="divisor_example",
                                            resource="resource_example",
                                        ),
                                    ),
                                ],
                            ),
                            empty_dir=EmptyDirVolumeSource(
                                medium="medium_example",
                                size_limit="size_limit_example",
                            ),
                            ephemeral=EphemeralVolumeSource(
                                volume_claim_template=PersistentVolumeClaimTemplate(
                                    metadata=ObjectMeta(
                                        annotations={
                                            "key": "key_example",
                                        },
                                        creation_timestamp=dateutil_parser('1970-01-01T00:00:00.00Z'),
                                        deletion_grace_period_seconds=1,
                                        deletion_timestamp=dateutil_parser('1970-01-01T00:00:00.00Z'),
                                        finalizers=[
                                            "finalizers_example",
                                        ],
                                        generate_name="generate_name_example",
                                        generation=1,
                                        labels={
                                            "key": "key_example",
                                        },
                                        managed_fields=[
                                            ManagedFieldsEntry(
                                                api_version="api_version_example",
                                                fields_type="fields_type_example",
                                                fields_v1={},
                                                manager="manager_example",
                                                operation="operation_example",
                                                subresource="subresource_example",
                                                time=dateutil_parser('1970-01-01T00:00:00.00Z'),
                                            ),
                                        ],
                                        name="name_example",
                                        namespace="namespace_example",
                                        owner_references=[
                                            OwnerReference(
                                                api_version="api_version_example",
                                                block_owner_deletion=True,
                                                controller=True,
                                                kind="kind_example",
                                                name="name_example",
                                                uid="uid_example",
                                            ),
                                        ],
                                        resource_version="resource_version_example",
                                        self_link="self_link_example",
                                        uid="uid_example",
                                    ),
                                    spec=PersistentVolumeClaimSpec(
                                        access_modes=[
                                            "access_modes_example",
                                        ],
                                        data_source=TypedLocalObjectReference(
                                            api_group="api_group_example",
                                            kind="kind_example",
                                            name="name_example",
                                        ),
                                        data_source_ref=TypedObjectReference(
                                            api_group="api_group_example",
                                            kind="kind_example",
                                            name="name_example",
                                            namespace="namespace_example",
                                        ),
                                        resources=VolumeResourceRequirements(
                                            limits={
                                                "key": "key_example",
                                            },
                                            requests={
                                                "key": "key_example",
                                            },
                                        ),
                                        selector=LabelSelector(
                                            match_expressions=[
                                                LabelSelectorRequirement(
                                                    key="key_example",
                                                    operator="operator_example",
                                                    values=[
                                                        "values_example",
                                                    ],
                                                ),
                                            ],
                                            match_labels={
                                                "key": "key_example",
                                            },
                                        ),
                                        storage_class_name="storage_class_name_example",
                                        volume_attributes_class_name="volume_attributes_class_name_example",
                                        volume_mode="volume_mode_example",
                                        volume_name="volume_name_example",
                                    ),
                                ),
                            ),
                            fc=FCVolumeSource(
                                fs_type="fs_type_example",
                                lun=1,
                                read_only=True,
                                target_wwns=[
                                    "target_wwns_example",
                                ],
                                wwids=[
                                    "wwids_example",
                                ],
                            ),
                            flex_volume=FlexVolumeSource(
                                driver="driver_example",
                                fs_type="fs_type_example",
                                options={
                                    "key": "key_example",
                                },
                                read_only=True,
                                secret_ref=LocalObjectReference(
                                    name="name_example",
                                ),
                            ),
                            flocker=FlockerVolumeSource(
                                dataset_name="dataset_name_example",
                                dataset_uuid="dataset_uuid_example",
                            ),
                            gce_persistent_disk=GCEPersistentDiskVolumeSource(
                                fs_type="fs_type_example",
                                partition=1,
                                pd_name="pd_name_example",
                                read_only=True,
                            ),
                            git_repo=GitRepoVolumeSource(
                                directory="directory_example",
                                repository="repository_example",
                                revision="revision_example",
                            ),
                            glusterfs=GlusterfsVolumeSource(
                                endpoints="endpoints_example",
                                path="path_example",
                                read_only=True,
                            ),
                            host_path=HostPathVolumeSource(
                                path="path_example",
                                type="type_example",
                            ),
                            image=ImageVolumeSource(
                                pull_policy="pull_policy_example",
                                reference="reference_example",
                            ),
                            iscsi=ISCSIVolumeSource(
                                chap_auth_discovery=True,
                                chap_auth_session=True,
                                fs_type="fs_type_example",
                                initiator_name="initiator_name_example",
                                iqn="iqn_example",
                                iscsi_interface="iscsi_interface_example",
                                lun=1,
                                portals=[
                                    "portals_example",
                                ],
                                read_only=True,
                                secret_ref=LocalObjectReference(
                                    name="name_example",
                                ),
                                target_portal="target_portal_example",
                            ),
                            name="name_example",
                            nfs=NFSVolumeSource(
                                path="path_example",
                                read_only=True,
                                server="server_example",
                            ),
                            persistent_volume_claim=PersistentVolumeClaimVolumeSource(
                                claim_name="claim_name_example",
                                read_only=True,
                            ),
                            photon_persistent_disk=PhotonPersistentDiskVolumeSource(
                                fs_type="fs_type_example",
                                pd_id="pd_id_example",
                            ),
                            portworx_volume=PortworxVolumeSource(
                                fs_type="fs_type_example",
                                read_only=True,
                                volume_id="volume_id_example",
                            ),
                            projected=ProjectedVolumeSource(
                                default_mode=1,
                                sources=[
                                    VolumeProjection(
                                        cluster_trust_bundle=ClusterTrustBundleProjection(
                                            label_selector=LabelSelector(
                                                match_expressions=[
                                                    LabelSelectorRequirement(
                                                        key="key_example",
                                                        operator="operator_example",
                                                        values=[
                                                            "values_example",
                                                        ],
                                                    ),
                                                ],
                                                match_labels={
                                                    "key": "key_example",
                                                },
                                            ),
                                            name="name_example",
                                            optional=True,
                                            path="path_example",
                                            signer_name="signer_name_example",
                                        ),
                                        config_map=ConfigMapProjection(
                                            items=[
                                                KeyToPath(
                                                    key="key_example",
                                                    mode=1,
                                                    path="path_example",
                                                ),
                                            ],
                                            name="name_example",
                                            optional=True,
                                        ),
                                        downward_api=DownwardAPIProjection(
                                            items=[
                                                DownwardAPIVolumeFile(
                                                    field_ref=ObjectFieldSelector(
                                                        api_version="api_version_example",
                                                        field_path="field_path_example",
                                                    ),
                                                    mode=1,
                                                    path="path_example",
                                                    resource_field_ref=ResourceFieldSelector(
                                                        container_name="container_name_example",
                                                        divisor="divisor_example",
                                                        resource="resource_example",
                                                    ),
                                                ),
                                            ],
                                        ),
                                        secret=SecretProjection(
                                            items=[
                                                KeyToPath(
                                                    key="key_example",
                                                    mode=1,
                                                    path="path_example",
                                                ),
                                            ],
                                            name="name_example",
                                            optional=True,
                                        ),
                                        service_account_token=ServiceAccountTokenProjection(
                                            audience="audience_example",
                                            expiration_seconds=1,
                                            path="path_example",
                                        ),
                                    ),
                                ],
                            ),
                            quobyte=QuobyteVolumeSource(
                                group="group_example",
                                read_only=True,
                                registry="registry_example",
                                tenant="tenant_example",
                                user="user_example",
                                volume="volume_example",
                            ),
                            rbd=RBDVolumeSource(
                                fs_type="fs_type_example",
                                image="image_example",
                                keyring="keyring_example",
                                monitors=[
                                    "monitors_example",
                                ],
                                pool="pool_example",
                                read_only=True,
                                secret_ref=LocalObjectReference(
                                    name="name_example",
                                ),
                                user="user_example",
                            ),
                            scale_io=ScaleIOVolumeSource(
                                fs_type="fs_type_example",
                                gateway="gateway_example",
                                protection_domain="protection_domain_example",
                                read_only=True,
                                secret_ref=LocalObjectReference(
                                    name="name_example",
                                ),
                                ssl_enabled=True,
                                storage_mode="storage_mode_example",
                                storage_pool="storage_pool_example",
                                system="system_example",
                                volume_name="volume_name_example",
                            ),
                            secret=SecretVolumeSource(
                                default_mode=1,
                                items=[
                                    KeyToPath(
                                        key="key_example",
                                        mode=1,
                                        path="path_example",
                                    ),
                                ],
                                optional=True,
                                secret_name="secret_name_example",
                            ),
                            storageos=StorageOSVolumeSource(
                                fs_type="fs_type_example",
                                read_only=True,
                                secret_ref=LocalObjectReference(
                                    name="name_example",
                                ),
                                volume_name="volume_name_example",
                                volume_namespace="volume_namespace_example",
                            ),
                            vsphere_volume=VsphereVirtualDiskVolumeSource(
                                fs_type="fs_type_example",
                                storage_policy_id="storage_policy_id_example",
                                storage_policy_name="storage_policy_name_example",
                                volume_path="volume_path_example",
                            ),
                        ),
                    ],
                ),
                triggers=[
                    GithubComArgoprojArgoEventsPkgApisEventsV1alpha1Trigger(
                        at_least_once=True,
                        dlq_trigger=GithubComArgoprojArgoEventsPkgApisEventsV1alpha1Trigger(),
                        parameters=[
                            GithubComArgoprojArgoEventsPkgApisEventsV1alpha1TriggerParameter(
                                dest="dest_example",
                                operation="operation_example",
                                src=GithubComArgoprojArgoEventsPkgApisEventsV1alpha1TriggerParameterSource(
                                    context_key="context_key_example",
                                    context_template="context_template_example",
                                    data_key="data_key_example",
                                    data_template="data_template_example",
                                    dependency_name="dependency_name_example",
                                    use_raw_data=True,
                                    value="value_example",
                                ),
                            ),
                        ],
                        policy=GithubComArgoprojArgoEventsPkgApisEventsV1alpha1TriggerPolicy(
                            k8s=GithubComArgoprojArgoEventsPkgApisEventsV1alpha1K8SResourcePolicy(
                                backoff=GithubComArgoprojArgoEventsPkgApisEventsV1alpha1Backoff(
                                    duration=GithubComArgoprojArgoEventsPkgApisEventsV1alpha1Int64OrString(
                                        int64_val="int64_val_example",
                                        str_val="str_val_example",
                                        type="type_example",
                                    ),
                                    factor=GithubComArgoprojArgoEventsPkgApisEventsV1alpha1Amount(
                                        value='YQ==',
                                    ),
                                    jitter=GithubComArgoprojArgoEventsPkgApisEventsV1alpha1Amount(
                                        value='YQ==',
                                    ),
                                    steps=1,
                                ),
                                error_on_backoff_timeout=True,
                                labels={
                                    "key": "key_example",
                                },
                            ),
                            status=GithubComArgoprojArgoEventsPkgApisEventsV1alpha1StatusPolicy(
                                allow=[
                                    1,
                                ],
                            ),
                        ),
                        rate_limit=GithubComArgoprojArgoEventsPkgApisEventsV1alpha1RateLimit(
                            requests_per_unit=1,
                            unit="unit_example",
                        ),
                        retry_strategy=GithubComArgoprojArgoEventsPkgApisEventsV1alpha1Backoff(
                            duration=GithubComArgoprojArgoEventsPkgApisEventsV1alpha1Int64OrString(
                                int64_val="int64_val_example",
                                str_val="str_val_example",
                                type="type_example",
                            ),
                            factor=GithubComArgoprojArgoEventsPkgApisEventsV1alpha1Amount(
                                value='YQ==',
                            ),
                            jitter=GithubComArgoprojArgoEventsPkgApisEventsV1alpha1Amount(
                                value='YQ==',
                            ),
                            steps=1,
                        ),
                        template=GithubComArgoprojArgoEventsPkgApisEventsV1alpha1TriggerTemplate(
                            argo_workflow=GithubComArgoprojArgoEventsPkgApisEventsV1alpha1ArgoWorkflowTrigger(
                                args=[
                                    "args_example",
                                ],
                                operation="operation_example",
                                parameters=[
                                    GithubComArgoprojArgoEventsPkgApisEventsV1alpha1TriggerParameter(
                                        dest="dest_example",
                                        operation="operation_example",
                                        src=GithubComArgoprojArgoEventsPkgApisEventsV1alpha1TriggerParameterSource(
                                            context_key="context_key_example",
                                            context_template="context_template_example",
                                            data_key="data_key_example",
                                            data_template="data_template_example",
                                            dependency_name="dependency_name_example",
                                            use_raw_data=True,
                                            value="value_example",
                                        ),
                                    ),
                                ],
                                source=GithubComArgoprojArgoEventsPkgApisEventsV1alpha1ArtifactLocation(
                                    configmap=ConfigMapKeySelector(
                                        key="key_example",
                                        name="name_example",
                                        optional=True,
                                    ),
                                    file=GithubComArgoprojArgoEventsPkgApisEventsV1alpha1FileArtifact(
                                        path="path_example",
                                    ),
                                    git=GithubComArgoprojArgoEventsPkgApisEventsV1alpha1GitArtifact(
                                        branch="branch_example",
                                        clone_directory="clone_directory_example",
                                        creds=GithubComArgoprojArgoEventsPkgApisEventsV1alpha1GitCreds(
                                            password=SecretKeySelector(
                                                key="key_example",
                                                name="name_example",
                                                optional=True,
                                            ),
                                            username=SecretKeySelector(
                                                key="key_example",
                                                name="name_example",
                                                optional=True,
                                            ),
                                        ),
                                        file_path="file_path_example",
                                        insecure_ignore_host_key=True,
                                        ref="ref_example",
                                        remote=GithubComArgoprojArgoEventsPkgApisEventsV1alpha1GitRemoteConfig(
                                            name="name_example",
                                            urls=[
                                                "urls_example",
                                            ],
                                        ),
                                        ssh_key_secret=SecretKeySelector(
                                            key="key_example",
                                            name="name_example",
                                            optional=True,
                                        ),
                                        tag="tag_example",
                                        url="url_example",
                                    ),
                                    inline="inline_example",
                                    resource=GithubComArgoprojArgoEventsPkgApisEventsV1alpha1K8SResource(
                                        value='YQ==',
                                    ),
                                    s3=GithubComArgoprojArgoEventsPkgApisEventsV1alpha1S3Artifact(
                                        access_key=SecretKeySelector(
                                            key="key_example",
                                            name="name_example",
                                            optional=True,
                                        ),
                                        bucket=GithubComArgoprojArgoEventsPkgApisEventsV1alpha1S3Bucket(
                                            key="key_example",
                                            name="name_example",
                                        ),
                                        ca_certificate=SecretKeySelector(
                                            key="key_example",
                                            name="name_example",
                                            optional=True,
                                        ),
                                        endpoint="endpoint_example",
                                        events=[
                                            "events_example",
                                        ],
                                        filter=GithubComArgoprojArgoEventsPkgApisEventsV1alpha1S3Filter(
                                            prefix="prefix_example",
                                            suffix="suffix_example",
                                        ),
                                        insecure=True,
                                        metadata={
                                            "key": "key_example",
                                        },
                                        region="region_example",
                                        secret_key=SecretKeySelector(
                                            key="key_example",
                                            name="name_example",
                                            optional=True,
                                        ),
                                    ),
                                    url=GithubComArgoprojArgoEventsPkgApisEventsV1alpha1URLArtifact(
                                        path="path_example",
                                        verify_cert=True,
                                    ),
                                ),
                            ),
                            aws_lambda=GithubComArgoprojArgoEventsPkgApisEventsV1alpha1AWSLambdaTrigger(
                                access_key=SecretKeySelector(
                                    key="key_example",
                                    name="name_example",
                                    optional=True,
                                ),
                                function_name="function_name_example",
                                invocation_type="invocation_type_example",
                                parameters=[
                                    GithubComArgoprojArgoEventsPkgApisEventsV1alpha1TriggerParameter(
                                        dest="dest_example",
                                        operation="operation_example",
                                        src=GithubComArgoprojArgoEventsPkgApisEventsV1alpha1TriggerParameterSource(
                                            context_key="context_key_example",
                                            context_template="context_template_example",
                                            data_key="data_key_example",
                                            data_template="data_template_example",
                                            dependency_name="dependency_name_example",
                                            use_raw_data=True,
                                            value="value_example",
                                        ),
                                    ),
                                ],
                                payload=[
                                    GithubComArgoprojArgoEventsPkgApisEventsV1alpha1TriggerParameter(
                                        dest="dest_example",
                                        operation="operation_example",
                                        src=GithubComArgoprojArgoEventsPkgApisEventsV1alpha1TriggerParameterSource(
                                            context_key="context_key_example",
                                            context_template="context_template_example",
                                            data_key="data_key_example",
                                            data_template="data_template_example",
                                            dependency_name="dependency_name_example",
                                            use_raw_data=True,
                                            value="value_example",
                                        ),
                                    ),
                                ],
                                region="region_example",
                                role_arn="role_arn_example",
                                secret_key=SecretKeySelector(
                                    key="key_example",
                                    name="name_example",
                                    optional=True,
                                ),
                            ),
                            azure_event_hubs=GithubComArgoprojArgoEventsPkgApisEventsV1alpha1AzureEventHubsTrigger(
                                fqdn="fqdn_example",
                                hub_name="hub_name_example",
                                parameters=[
                                    GithubComArgoprojArgoEventsPkgApisEventsV1alpha1TriggerParameter(
                                        dest="dest_example",
                                        operation="operation_example",
                                        src=GithubComArgoprojArgoEventsPkgApisEventsV1alpha1TriggerParameterSource(
                                            context_key="context_key_example",
                                            context_template="context_template_example",
                                            data_key="data_key_example",
                                            data_template="data_template_example",
                                            dependency_name="dependency_name_example",
                                            use_raw_data=True,
                                            value="value_example",
                                        ),
                                    ),
                                ],
                                payload=[
                                    GithubComArgoprojArgoEventsPkgApisEventsV1alpha1TriggerParameter(
                                        dest="dest_example",
                                        operation="operation_example",
                                        src=GithubComArgoprojArgoEventsPkgApisEventsV1alpha1TriggerParameterSource(
                                            context_key="context_key_example",
                                            context_template="context_template_example",
                                            data_key="data_key_example",
                                            data_template="data_template_example",
                                            dependency_name="dependency_name_example",
                                            use_raw_data=True,
                                            value="value_example",
                                        ),
                                    ),
                                ],
                                shared_access_key=SecretKeySelector(
                                    key="key_example",
                                    name="name_example",
                                    optional=True,
                                ),
                                shared_access_key_name=SecretKeySelector(
                                    key="key_example",
                                    name="name_example",
                                    optional=True,
                                ),
                            ),
                            azure_service_bus=GithubComArgoprojArgoEventsPkgApisEventsV1alpha1AzureServiceBusTrigger(
                                connection_string=SecretKeySelector(
                                    key="key_example",
                                    name="name_example",
                                    optional=True,
                                ),
                                parameters=[
                                    GithubComArgoprojArgoEventsPkgApisEventsV1alpha1TriggerParameter(
                                        dest="dest_example",
                                        operation="operation_example",
                                        src=GithubComArgoprojArgoEventsPkgApisEventsV1alpha1TriggerParameterSource(
                                            context_key="context_key_example",
                                            context_template="context_template_example",
                                            data_key="data_key_example",
                                            data_template="data_template_example",
                                            dependency_name="dependency_name_example",
                                            use_raw_data=True,
                                            value="value_example",
                                        ),
                                    ),
                                ],
                                payload=[
                                    GithubComArgoprojArgoEventsPkgApisEventsV1alpha1TriggerParameter(
                                        dest="dest_example",
                                        operation="operation_example",
                                        src=GithubComArgoprojArgoEventsPkgApisEventsV1alpha1TriggerParameterSource(
                                            context_key="context_key_example",
                                            context_template="context_template_example",
                                            data_key="data_key_example",
                                            data_template="data_template_example",
                                            dependency_name="dependency_name_example",
                                            use_raw_data=True,
                                            value="value_example",
                                        ),
                                    ),
                                ],
                                queue_name="queue_name_example",
                                subscription_name="subscription_name_example",
                                tls=GithubComArgoprojArgoEventsPkgApisEventsV1alpha1TLSConfig(
                                    ca_cert_secret=SecretKeySelector(
                                        key="key_example",
                                        name="name_example",
                                        optional=True,
                                    ),
                                    client_cert_secret=SecretKeySelector(
                                        key="key_example",
                                        name="name_example",
                                        optional=True,
                                    ),
                                    client_key_secret=SecretKeySelector(
                                        key="key_example",
                                        name="name_example",
                                        optional=True,
                                    ),
                                    insecure_skip_verify=True,
                                ),
                                topic_name="topic_name_example",
                            ),
                            conditions="conditions_example",
                            conditions_reset=[
                                GithubComArgoprojArgoEventsPkgApisEventsV1alpha1ConditionsResetCriteria(
                                    by_time=GithubComArgoprojArgoEventsPkgApisEventsV1alpha1ConditionsResetByTime(
                                        cron="cron_example",
                                        timezone="timezone_example",
                                    ),
                                ),
                            ],
                            custom=GithubComArgoprojArgoEventsPkgApisEventsV1alpha1CustomTrigger(
                                cert_secret=SecretKeySelector(
                                    key="key_example",
                                    name="name_example",
                                    optional=True,
                                ),
                                parameters=[
                                    GithubComArgoprojArgoEventsPkgApisEventsV1alpha1TriggerParameter(
                                        dest="dest_example",
                                        operation="operation_example",
                                        src=GithubComArgoprojArgoEventsPkgApisEventsV1alpha1TriggerParameterSource(
                                            context_key="context_key_example",
                                            context_template="context_template_example",
                                            data_key="data_key_example",
                                            data_template="data_template_example",
                                            dependency_name="dependency_name_example",
                                            use_raw_data=True,
                                            value="value_example",
                                        ),
                                    ),
                                ],
                                payload=[
                                    GithubComArgoprojArgoEventsPkgApisEventsV1alpha1TriggerParameter(
                                        dest="dest_example",
                                        operation="operation_example",
                                        src=GithubComArgoprojArgoEventsPkgApisEventsV1alpha1TriggerParameterSource(
                                            context_key="context_key_example",
                                            context_template="context_template_example",
                                            data_key="data_key_example",
                                            data_template="data_template_example",
                                            dependency_name="dependency_name_example",
                                            use_raw_data=True,
                                            value="value_example",
                                        ),
                                    ),
                                ],
                                secure=True,
                                server_name_override="server_name_override_example",
                                server_url="server_url_example",
                                spec={
                                    "key": "key_example",
                                },
                            ),
                            email=GithubComArgoprojArgoEventsPkgApisEventsV1alpha1EmailTrigger(
                                body="body_example",
                                _from="_from_example",
                                host="host_example",
                                parameters=[
                                    GithubComArgoprojArgoEventsPkgApisEventsV1alpha1TriggerParameter(
                                        dest="dest_example",
                                        operation="operation_example",
                                        src=GithubComArgoprojArgoEventsPkgApisEventsV1alpha1TriggerParameterSource(
                                            context_key="context_key_example",
                                            context_template="context_template_example",
                                            data_key="data_key_example",
                                            data_template="data_template_example",
                                            dependency_name="dependency_name_example",
                                            use_raw_data=True,
                                            value="value_example",
                                        ),
                                    ),
                                ],
                                port=1,
                                smtp_password=SecretKeySelector(
                                    key="key_example",
                                    name="name_example",
                                    optional=True,
                                ),
                                subject="subject_example",
                                to=[
                                    "to_example",
                                ],
                                username="username_example",
                            ),
                            http=GithubComArgoprojArgoEventsPkgApisEventsV1alpha1HTTPTrigger(
                                basic_auth=GithubComArgoprojArgoEventsPkgApisEventsV1alpha1BasicAuth(
                                    password=SecretKeySelector(
                                        key="key_example",
                                        name="name_example",
                                        optional=True,
                                    ),
                                    username=SecretKeySelector(
                                        key="key_example",
                                        name="name_example",
                                        optional=True,
                                    ),
                                ),
                                headers={
                                    "key": "key_example",
                                },
                                method="method_example",
                                parameters=[
                                    GithubComArgoprojArgoEventsPkgApisEventsV1alpha1TriggerParameter(
                                        dest="dest_example",
                                        operation="operation_example",
                                        src=GithubComArgoprojArgoEventsPkgApisEventsV1alpha1TriggerParameterSource(
                                            context_key="context_key_example",
                                            context_template="context_template_example",
                                            data_key="data_key_example",
                                            data_template="data_template_example",
                                            dependency_name="dependency_name_example",
                                            use_raw_data=True,
                                            value="value_example",
                                        ),
                                    ),
                                ],
                                payload=[
                                    GithubComArgoprojArgoEventsPkgApisEventsV1alpha1TriggerParameter(
                                        dest="dest_example",
                                        operation="operation_example",
                                        src=GithubComArgoprojArgoEventsPkgApisEventsV1alpha1TriggerParameterSource(
                                            context_key="context_key_example",
                                            context_template="context_template_example",
                                            data_key="data_key_example",
                                            data_template="data_template_example",
                                            dependency_name="dependency_name_example",
                                            use_raw_data=True,
                                            value="value_example",
                                        ),
                                    ),
                                ],
                                secure_headers=[
                                    GithubComArgoprojArgoEventsPkgApisEventsV1alpha1SecureHeader(
                                        name="name_example",
                                        value_from=GithubComArgoprojArgoEventsPkgApisEventsV1alpha1ValueFromSource(
                                            config_map_key_ref=ConfigMapKeySelector(
                                                key="key_example",
                                                name="name_example",
                                                optional=True,
                                            ),
                                            secret_key_ref=SecretKeySelector(
                                                key="key_example",
                                                name="name_example",
                                                optional=True,
                                            ),
                                        ),
                                    ),
                                ],
                                timeout="timeout_example",
                                tls=GithubComArgoprojArgoEventsPkgApisEventsV1alpha1TLSConfig(
                                    ca_cert_secret=SecretKeySelector(
                                        key="key_example",
                                        name="name_example",
                                        optional=True,
                                    ),
                                    client_cert_secret=SecretKeySelector(
                                        key="key_example",
                                        name="name_example",
                                        optional=True,
                                    ),
                                    client_key_secret=SecretKeySelector(
                                        key="key_example",
                                        name="name_example",
                                        optional=True,
                                    ),
                                    insecure_skip_verify=True,
                                ),
                                url="url_example",
                            ),
                            k8s=GithubComArgoprojArgoEventsPkgApisEventsV1alpha1StandardK8STrigger(
                                live_object=True,
                                operation="operation_example",
                                parameters=[
                                    GithubComArgoprojArgoEventsPkgApisEventsV1alpha1TriggerParameter(
                                        dest="dest_example",
                                        operation="operation_example",
                                        src=GithubComArgoprojArgoEventsPkgApisEventsV1alpha1TriggerParameterSource(
                                            context_key="context_key_example",
                                            context_template="context_template_example",
                                            data_key="data_key_example",
                                            data_template="data_template_example",
                                            dependency_name="dependency_name_example",
                                            use_raw_data=True,
                                            value="value_example",
                                        ),
                                    ),
                                ],
                                patch_strategy="patch_strategy_example",
                                source=GithubComArgoprojArgoEventsPkgApisEventsV1alpha1ArtifactLocation(
                                    configmap=ConfigMapKeySelector(
                                        key="key_example",
                                        name="name_example",
                                        optional=True,
                                    ),
                                    file=GithubComArgoprojArgoEventsPkgApisEventsV1alpha1FileArtifact(
                                        path="path_example",
                                    ),
                                    git=GithubComArgoprojArgoEventsPkgApisEventsV1alpha1GitArtifact(
                                        branch="branch_example",
                                        clone_directory="clone_directory_example",
                                        creds=GithubComArgoprojArgoEventsPkgApisEventsV1alpha1GitCreds(
                                            password=SecretKeySelector(
                                                key="key_example",
                                                name="name_example",
                                                optional=True,
                                            ),
                                            username=SecretKeySelector(
                                                key="key_example",
                                                name="name_example",
                                                optional=True,
                                            ),
                                        ),
                                        file_path="file_path_example",
                                        insecure_ignore_host_key=True,
                                        ref="ref_example",
                                        remote=GithubComArgoprojArgoEventsPkgApisEventsV1alpha1GitRemoteConfig(
                                            name="name_example",
                                            urls=[
                                                "urls_example",
                                            ],
                                        ),
                                        ssh_key_secret=SecretKeySelector(
                                            key="key_example",
                                            name="name_example",
                                            optional=True,
                                        ),
                                        tag="tag_example",
                                        url="url_example",
                                    ),
                                    inline="inline_example",
                                    resource=GithubComArgoprojArgoEventsPkgApisEventsV1alpha1K8SResource(
                                        value='YQ==',
                                    ),
                                    s3=GithubComArgoprojArgoEventsPkgApisEventsV1alpha1S3Artifact(
                                        access_key=SecretKeySelector(
                                            key="key_example",
                                            name="name_example",
                                            optional=True,
                                        ),
                                        bucket=GithubComArgoprojArgoEventsPkgApisEventsV1alpha1S3Bucket(
                                            key="key_example",
                                            name="name_example",
                                        ),
                                        ca_certificate=SecretKeySelector(
                                            key="key_example",
                                            name="name_example",
                                            optional=True,
                                        ),
                                        endpoint="endpoint_example",
                                        events=[
                                            "events_example",
                                        ],
                                        filter=GithubComArgoprojArgoEventsPkgApisEventsV1alpha1S3Filter(
                                            prefix="prefix_example",
                                            suffix="suffix_example",
                                        ),
                                        insecure=True,
                                        metadata={
                                            "key": "key_example",
                                        },
                                        region="region_example",
                                        secret_key=SecretKeySelector(
                                            key="key_example",
                                            name="name_example",
                                            optional=True,
                                        ),
                                    ),
                                    url=GithubComArgoprojArgoEventsPkgApisEventsV1alpha1URLArtifact(
                                        path="path_example",
                                        verify_cert=True,
                                    ),
                                ),
                            ),
                            kafka=GithubComArgoprojArgoEventsPkgApisEventsV1alpha1KafkaTrigger(
                                compress=True,
                                flush_frequency=1,
                                headers={
                                    "key": "key_example",
                                },
                                parameters=[
                                    GithubComArgoprojArgoEventsPkgApisEventsV1alpha1TriggerParameter(
                                        dest="dest_example",
                                        operation="operation_example",
                                        src=GithubComArgoprojArgoEventsPkgApisEventsV1alpha1TriggerParameterSource(
                                            context_key="context_key_example",
                                            context_template="context_template_example",
                                            data_key="data_key_example",
                                            data_template="data_template_example",
                                            dependency_name="dependency_name_example",
                                            use_raw_data=True,
                                            value="value_example",
                                        ),
                                    ),
                                ],
                                partition=1,
                                partitioning_key="partitioning_key_example",
                                payload=[
                                    GithubComArgoprojArgoEventsPkgApisEventsV1alpha1TriggerParameter(
                                        dest="dest_example",
                                        operation="operation_example",
                                        src=GithubComArgoprojArgoEventsPkgApisEventsV1alpha1TriggerParameterSource(
                                            context_key="context_key_example",
                                            context_template="context_template_example",
                                            data_key="data_key_example",
                                            data_template="data_template_example",
                                            dependency_name="dependency_name_example",
                                            use_raw_data=True,
                                            value="value_example",
                                        ),
                                    ),
                                ],
                                required_acks=1,
                                sasl=GithubComArgoprojArgoEventsPkgApisEventsV1alpha1SASLConfig(
                                    mechanism="mechanism_example",
                                    password_secret=SecretKeySelector(
                                        key="key_example",
                                        name="name_example",
                                        optional=True,
                                    ),
                                    user_secret=SecretKeySelector(
                                        key="key_example",
                                        name="name_example",
                                        optional=True,
                                    ),
                                ),
                                schema_registry=GithubComArgoprojArgoEventsPkgApisEventsV1alpha1SchemaRegistryConfig(
                                    auth=GithubComArgoprojArgoEventsPkgApisEventsV1alpha1BasicAuth(
                                        password=SecretKeySelector(
                                            key="key_example",
                                            name="name_example",
                                            optional=True,
                                        ),
                                        username=SecretKeySelector(
                                            key="key_example",
                                            name="name_example",
                                            optional=True,
                                        ),
                                    ),
                                    schema_id=1,
                                    url="url_example",
                                ),
                                secure_headers=[
                                    GithubComArgoprojArgoEventsPkgApisEventsV1alpha1SecureHeader(
                                        name="name_example",
                                        value_from=GithubComArgoprojArgoEventsPkgApisEventsV1alpha1ValueFromSource(
                                            config_map_key_ref=ConfigMapKeySelector(
                                                key="key_example",
                                                name="name_example",
                                                optional=True,
                                            ),
                                            secret_key_ref=SecretKeySelector(
                                                key="key_example",
                                                name="name_example",
                                                optional=True,
                                            ),
                                        ),
                                    ),
                                ],
                                tls=GithubComArgoprojArgoEventsPkgApisEventsV1alpha1TLSConfig(
                                    ca_cert_secret=SecretKeySelector(
                                        key="key_example",
                                        name="name_example",
                                        optional=True,
                                    ),
                                    client_cert_secret=SecretKeySelector(
                                        key="key_example",
                                        name="name_example",
                                        optional=True,
                                    ),
                                    client_key_secret=SecretKeySelector(
                                        key="key_example",
                                        name="name_example",
                                        optional=True,
                                    ),
                                    insecure_skip_verify=True,
                                ),
                                topic="topic_example",
                                url="url_example",
                                version="version_example",
                            ),
                            log=GithubComArgoprojArgoEventsPkgApisEventsV1alpha1LogTrigger(
                                interval_seconds="interval_seconds_example",
                            ),
                            name="name_example",
                            nats=GithubComArgoprojArgoEventsPkgApisEventsV1alpha1NATSTrigger(
                                auth=GithubComArgoprojArgoEventsPkgApisEventsV1alpha1NATSAuth(
                                    basic=GithubComArgoprojArgoEventsPkgApisEventsV1alpha1BasicAuth(
                                        password=SecretKeySelector(
                                            key="key_example",
                                            name="name_example",
                                            optional=True,
                                        ),
                                        username=SecretKeySelector(
                                            key="key_example",
                                            name="name_example",
                                            optional=True,
                                        ),
                                    ),
                                    credential=SecretKeySelector(
                                        key="key_example",
                                        name="name_example",
                                        optional=True,
                                    ),
                                    nkey=SecretKeySelector(
                                        key="key_example",
                                        name="name_example",
                                        optional=True,
                                    ),
                                    token=SecretKeySelector(
                                        key="key_example",
                                        name="name_example",
                                        optional=True,
                                    ),
                                ),
                                parameters=[
                                    GithubComArgoprojArgoEventsPkgApisEventsV1alpha1TriggerParameter(
                                        dest="dest_example",
                                        operation="operation_example",
                                        src=GithubComArgoprojArgoEventsPkgApisEventsV1alpha1TriggerParameterSource(
                                            context_key="context_key_example",
                                            context_template="context_template_example",
                                            data_key="data_key_example",
                                            data_template="data_template_example",
                                            dependency_name="dependency_name_example",
                                            use_raw_data=True,
                                            value="value_example",
                                        ),
                                    ),
                                ],
                                payload=[
                                    GithubComArgoprojArgoEventsPkgApisEventsV1alpha1TriggerParameter(
                                        dest="dest_example",
                                        operation="operation_example",
                                        src=GithubComArgoprojArgoEventsPkgApisEventsV1alpha1TriggerParameterSource(
                                            context_key="context_key_example",
                                            context_template="context_template_example",
                                            data_key="data_key_example",
                                            data_template="data_template_example",
                                            dependency_name="dependency_name_example",
                                            use_raw_data=True,
                                            value="value_example",
                                        ),
                                    ),
                                ],
                                subject="subject_example",
                                tls=GithubComArgoprojArgoEventsPkgApisEventsV1alpha1TLSConfig(
                                    ca_cert_secret=SecretKeySelector(
                                        key="key_example",
                                        name="name_example",
                                        optional=True,
                                    ),
                                    client_cert_secret=SecretKeySelector(
                                        key="key_example",
                                        name="name_example",
                                        optional=True,
                                    ),
                                    client_key_secret=SecretKeySelector(
                                        key="key_example",
                                        name="name_example",
                                        optional=True,
                                    ),
                                    insecure_skip_verify=True,
                                ),
                                url="url_example",
                            ),
                            open_whisk=GithubComArgoprojArgoEventsPkgApisEventsV1alpha1OpenWhiskTrigger(
                                action_name="action_name_example",
                                auth_token=SecretKeySelector(
                                    key="key_example",
                                    name="name_example",
                                    optional=True,
                                ),
                                host="host_example",
                                namespace="namespace_example",
                                parameters=[
                                    GithubComArgoprojArgoEventsPkgApisEventsV1alpha1TriggerParameter(
                                        dest="dest_example",
                                        operation="operation_example",
                                        src=GithubComArgoprojArgoEventsPkgApisEventsV1alpha1TriggerParameterSource(
                                            context_key="context_key_example",
                                            context_template="context_template_example",
                                            data_key="data_key_example",
                                            data_template="data_template_example",
                                            dependency_name="dependency_name_example",
                                            use_raw_data=True,
                                            value="value_example",
                                        ),
                                    ),
                                ],
                                payload=[
                                    GithubComArgoprojArgoEventsPkgApisEventsV1alpha1TriggerParameter(
                                        dest="dest_example",
                                        operation="operation_example",
                                        src=GithubComArgoprojArgoEventsPkgApisEventsV1alpha1TriggerParameterSource(
                                            context_key="context_key_example",
                                            context_template="context_template_example",
                                            data_key="data_key_example",
                                            data_template="data_template_example",
                                            dependency_name="dependency_name_example",
                                            use_raw_data=True,
                                            value="value_example",
                                        ),
                                    ),
                                ],
                                version="version_example",
                            ),
                            pulsar=GithubComArgoprojArgoEventsPkgApisEventsV1alpha1PulsarTrigger(
                                auth_athenz_params={
                                    "key": "key_example",
                                },
                                auth_athenz_secret=SecretKeySelector(
                                    key="key_example",
                                    name="name_example",
                                    optional=True,
                                ),
                                auth_token_secret=SecretKeySelector(
                                    key="key_example",
                                    name="name_example",
                                    optional=True,
                                ),
                                connection_backoff=GithubComArgoprojArgoEventsPkgApisEventsV1alpha1Backoff(
                                    duration=GithubComArgoprojArgoEventsPkgApisEventsV1alpha1Int64OrString(
                                        int64_val="int64_val_example",
                                        str_val="str_val_example",
                                        type="type_example",
                                    ),
                                    factor=GithubComArgoprojArgoEventsPkgApisEventsV1alpha1Amount(
                                        value='YQ==',
                                    ),
                                    jitter=GithubComArgoprojArgoEventsPkgApisEventsV1alpha1Amount(
                                        value='YQ==',
                                    ),
                                    steps=1,
                                ),
                                parameters=[
                                    GithubComArgoprojArgoEventsPkgApisEventsV1alpha1TriggerParameter(
                                        dest="dest_example",
                                        operation="operation_example",
                                        src=GithubComArgoprojArgoEventsPkgApisEventsV1alpha1TriggerParameterSource(
                                            context_key="context_key_example",
                                            context_template="context_template_example",
                                            data_key="data_key_example",
                                            data_template="data_template_example",
                                            dependency_name="dependency_name_example",
                                            use_raw_data=True,
                                            value="value_example",
                                        ),
                                    ),
                                ],
                                payload=[
                                    GithubComArgoprojArgoEventsPkgApisEventsV1alpha1TriggerParameter(
                                        dest="dest_example",
                                        operation="operation_example",
                                        src=GithubComArgoprojArgoEventsPkgApisEventsV1alpha1TriggerParameterSource(
                                            context_key="context_key_example",
                                            context_template="context_template_example",
                                            data_key="data_key_example",
                                            data_template="data_template_example",
                                            dependency_name="dependency_name_example",
                                            use_raw_data=True,
                                            value="value_example",
                                        ),
                                    ),
                                ],
                                tls=GithubComArgoprojArgoEventsPkgApisEventsV1alpha1TLSConfig(
                                    ca_cert_secret=SecretKeySelector(
                                        key="key_example",
                                        name="name_example",
                                        optional=True,
                                    ),
                                    client_cert_secret=SecretKeySelector(
                                        key="key_example",
                                        name="name_example",
                                        optional=True,
                                    ),
                                    client_key_secret=SecretKeySelector(
                                        key="key_example",
                                        name="name_example",
                                        optional=True,
                                    ),
                                    insecure_skip_verify=True,
                                ),
                                tls_allow_insecure_connection=True,
                                tls_trust_certs_secret=SecretKeySelector(
                                    key="key_example",
                                    name="name_example",
                                    optional=True,
                                ),
                                tls_validate_hostname=True,
                                topic="topic_example",
                                url="url_example",
                            ),
                            slack=GithubComArgoprojArgoEventsPkgApisEventsV1alpha1SlackTrigger(
                                attachments="attachments_example",
                                blocks="blocks_example",
                                channel="channel_example",
                                message="message_example",
                                parameters=[
                                    GithubComArgoprojArgoEventsPkgApisEventsV1alpha1TriggerParameter(
                                        dest="dest_example",
                                        operation="operation_example",
                                        src=GithubComArgoprojArgoEventsPkgApisEventsV1alpha1TriggerParameterSource(
                                            context_key="context_key_example",
                                            context_template="context_template_example",
                                            data_key="data_key_example",
                                            data_template="data_template_example",
                                            dependency_name="dependency_name_example",
                                            use_raw_data=True,
                                            value="value_example",
                                        ),
                                    ),
                                ],
                                sender=GithubComArgoprojArgoEventsPkgApisEventsV1alpha1SlackSender(
                                    icon="icon_example",
                                    username="username_example",
                                ),
                                slack_token=SecretKeySelector(
                                    key="key_example",
                                    name="name_example",
                                    optional=True,
                                ),
                                thread=GithubComArgoprojArgoEventsPkgApisEventsV1alpha1SlackThread(
                                    broadcast_message_to_channel=True,
                                    message_aggregation_key="message_aggregation_key_example",
                                ),
                            ),
                        ),
                    ),
                ],
            ),
            status=GithubComArgoprojArgoEventsPkgApisEventsV1alpha1SensorStatus(
                status=GithubComArgoprojArgoEventsPkgApisEventsV1alpha1Status(
                    conditions=[
                        GithubComArgoprojArgoEventsPkgApisEventsV1alpha1Condition(
                            last_transition_time=dateutil_parser('1970-01-01T00:00:00.00Z'),
                            message="message_example",
                            reason="reason_example",
                            status="status_example",
                            type="type_example",
                        ),
                    ],
                ),
            ),
        ),
    ) # SensorUpdateSensorRequest | 

    # example passing only required values which don't have defaults set
    try:
        api_response = api_instance.update_sensor(namespace, name, body)
        pprint(api_response)
    except argo_workflows.ApiException as e:
        print("Exception when calling SensorServiceApi->update_sensor: %s\n" % e)
```


### Parameters

Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **namespace** | **str**|  |
 **name** | **str**|  |
 **body** | [**SensorUpdateSensorRequest**](SensorUpdateSensorRequest.md)|  |

### Return type

[**GithubComArgoprojArgoEventsPkgApisEventsV1alpha1Sensor**](GithubComArgoprojArgoEventsPkgApisEventsV1alpha1Sensor.md)

### Authorization

[BearerToken](../README.md#BearerToken)

### HTTP request headers

 - **Content-Type**: application/json
 - **Accept**: application/json


### HTTP response details

| Status code | Description | Response headers |
|-------------|-------------|------------------|
**200** | A successful response. |  -  |
**0** | An unexpected error response. |  -  |

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

# **watch_sensors**
> StreamResultOfSensorSensorWatchEvent watch_sensors(namespace)



### Example

* Api Key Authentication (BearerToken):

```python
import time
import argo_workflows
from argo_workflows.api import sensor_service_api
from argo_workflows.model.grpc_gateway_runtime_error import GrpcGatewayRuntimeError
from argo_workflows.model.stream_result_of_sensor_sensor_watch_event import StreamResultOfSensorSensorWatchEvent
from pprint import pprint
# Defining the host is optional and defaults to http://localhost:2746
# See configuration.py for a list of all supported configuration parameters.
configuration = argo_workflows.Configuration(
    host = "http://localhost:2746"
)

# The client must configure the authentication and authorization parameters
# in accordance with the API server security policy.
# Examples for each auth method are provided below, use the example that
# satisfies your auth use case.

# Configure API key authorization: BearerToken
configuration.api_key['BearerToken'] = 'YOUR_API_KEY'

# Uncomment below to setup prefix (e.g. Bearer) for API key, if needed
# configuration.api_key_prefix['BearerToken'] = 'Bearer'

# Enter a context with an instance of the API client
with argo_workflows.ApiClient(configuration) as api_client:
    # Create an instance of the API class
    api_instance = sensor_service_api.SensorServiceApi(api_client)
    namespace = "namespace_example" # str | 
    list_options_label_selector = "listOptions.labelSelector_example" # str | A selector to restrict the list of returned objects by their labels. Defaults to everything. +optional. (optional)
    list_options_field_selector = "listOptions.fieldSelector_example" # str | A selector to restrict the list of returned objects by their fields. Defaults to everything. +optional. (optional)
    list_options_watch = True # bool | Watch for changes to the described resources and return them as a stream of add, update, and remove notifications. Specify resourceVersion. +optional. (optional)
    list_options_allow_watch_bookmarks = True # bool | allowWatchBookmarks requests watch events with type \"BOOKMARK\". Servers that do not implement bookmarks may ignore this flag and bookmarks are sent at the server's discretion. Clients should not assume bookmarks are returned at any specific interval, nor may they assume the server will send any BOOKMARK event during a session. If this is not a watch, this field is ignored. +optional. (optional)
    list_options_resource_version = "listOptions.resourceVersion_example" # str | resourceVersion sets a constraint on what resource versions a request may be served from. See https://kubernetes.io/docs/reference/using-api/api-concepts/#resource-versions for details.  Defaults to unset +optional (optional)
    list_options_resource_version_match = "listOptions.resourceVersionMatch_example" # str | resourceVersionMatch determines how resourceVersion is applied to list calls. It is highly recommended that resourceVersionMatch be set for list calls where resourceVersion is set See https://kubernetes.io/docs/reference/using-api/api-concepts/#resource-versions for details.  Defaults to unset +optional (optional)
    list_options_timeout_seconds = "listOptions.timeoutSeconds_example" # str | Timeout for the list/watch call. This limits the duration of the call, regardless of any activity or inactivity. +optional. (optional)
    list_options_limit = "listOptions.limit_example" # str | limit is a maximum number of responses to return for a list call. If more items exist, the server will set the `continue` field on the list metadata to a value that can be used with the same initial query to retrieve the next set of results. Setting a limit may return fewer than the requested amount of items (up to zero items) in the event all requested objects are filtered out and clients should only use the presence of the continue field to determine whether more results are available. Servers may choose not to support the limit argument and will return all of the available results. If limit is specified and the continue field is empty, clients may assume that no more results are available. This field is not supported if watch is true.  The server guarantees that the objects returned when using continue will be identical to issuing a single list call without a limit - that is, no objects created, modified, or deleted after the first request is issued will be included in any subsequent continued requests. This is sometimes referred to as a consistent snapshot, and ensures that a client that is using limit to receive smaller chunks of a very large result can ensure they see all possible objects. If objects are updated during a chunked list the version of the object that was present at the time the first list result was calculated is returned. (optional)
    list_options_continue = "listOptions.continue_example" # str | The continue option should be set when retrieving more results from the server. Since this value is server defined, clients may only use the continue value from a previous query result with identical query parameters (except for the value of continue) and the server may reject a continue value it does not recognize. If the specified continue value is no longer valid whether due to expiration (generally five to fifteen minutes) or a configuration change on the server, the server will respond with a 410 ResourceExpired error together with a continue token. If the client needs a consistent list, it must restart their list without the continue field. Otherwise, the client may send another list request with the token received with the 410 error, the server will respond with a list starting from the next key, but from the latest snapshot, which is inconsistent from the previous list results - objects that are created, modified, or deleted after the first list request will be included in the response, as long as their keys are after the \"next key\".  This field is not supported when watch is true. Clients may start a watch from the last resourceVersion value returned by the server and not miss any modifications. (optional)
    list_options_send_initial_events = True # bool | `sendInitialEvents=true` may be set together with `watch=true`. In that case, the watch stream will begin with synthetic events to produce the current state of objects in the collection. Once all such events have been sent, a synthetic \"Bookmark\" event  will be sent. The bookmark will report the ResourceVersion (RV) corresponding to the set of objects, and be marked with `\"io.k8s.initial-events-end\": \"true\"` annotation. Afterwards, the watch stream will proceed as usual, sending watch events corresponding to changes (subsequent to the RV) to objects watched.  When `sendInitialEvents` option is set, we require `resourceVersionMatch` option to also be set. The semantic of the watch request is as following: - `resourceVersionMatch` = NotOlderThan   is interpreted as \"data at least as new as the provided `resourceVersion`\"   and the bookmark event is send when the state is synced   to a `resourceVersion` at least as fresh as the one provided by the ListOptions.   If `resourceVersion` is unset, this is interpreted as \"consistent read\" and the   bookmark event is send when the state is synced at least to the moment   when request started being processed. - `resourceVersionMatch` set to any other value or unset   Invalid error is returned.  Defaults to true if `resourceVersion=\"\"` or `resourceVersion=\"0\"` (for backward compatibility reasons) and to false otherwise. +optional (optional)

    # example passing only required values which don't have defaults set
    try:
        api_response = api_instance.watch_sensors(namespace)
        pprint(api_response)
    except argo_workflows.ApiException as e:
        print("Exception when calling SensorServiceApi->watch_sensors: %s\n" % e)

    # example passing only required values which don't have defaults set
    # and optional values
    try:
        api_response = api_instance.watch_sensors(namespace, list_options_label_selector=list_options_label_selector, list_options_field_selector=list_options_field_selector, list_options_watch=list_options_watch, list_options_allow_watch_bookmarks=list_options_allow_watch_bookmarks, list_options_resource_version=list_options_resource_version, list_options_resource_version_match=list_options_resource_version_match, list_options_timeout_seconds=list_options_timeout_seconds, list_options_limit=list_options_limit, list_options_continue=list_options_continue, list_options_send_initial_events=list_options_send_initial_events)
        pprint(api_response)
    except argo_workflows.ApiException as e:
        print("Exception when calling SensorServiceApi->watch_sensors: %s\n" % e)
```


### Parameters

Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **namespace** | **str**|  |
 **list_options_label_selector** | **str**| A selector to restrict the list of returned objects by their labels. Defaults to everything. +optional. | [optional]
 **list_options_field_selector** | **str**| A selector to restrict the list of returned objects by their fields. Defaults to everything. +optional. | [optional]
 **list_options_watch** | **bool**| Watch for changes to the described resources and return them as a stream of add, update, and remove notifications. Specify resourceVersion. +optional. | [optional]
 **list_options_allow_watch_bookmarks** | **bool**| allowWatchBookmarks requests watch events with type \&quot;BOOKMARK\&quot;. Servers that do not implement bookmarks may ignore this flag and bookmarks are sent at the server&#39;s discretion. Clients should not assume bookmarks are returned at any specific interval, nor may they assume the server will send any BOOKMARK event during a session. If this is not a watch, this field is ignored. +optional. | [optional]
 **list_options_resource_version** | **str**| resourceVersion sets a constraint on what resource versions a request may be served from. See https://kubernetes.io/docs/reference/using-api/api-concepts/#resource-versions for details.  Defaults to unset +optional | [optional]
 **list_options_resource_version_match** | **str**| resourceVersionMatch determines how resourceVersion is applied to list calls. It is highly recommended that resourceVersionMatch be set for list calls where resourceVersion is set See https://kubernetes.io/docs/reference/using-api/api-concepts/#resource-versions for details.  Defaults to unset +optional | [optional]
 **list_options_timeout_seconds** | **str**| Timeout for the list/watch call. This limits the duration of the call, regardless of any activity or inactivity. +optional. | [optional]
 **list_options_limit** | **str**| limit is a maximum number of responses to return for a list call. If more items exist, the server will set the &#x60;continue&#x60; field on the list metadata to a value that can be used with the same initial query to retrieve the next set of results. Setting a limit may return fewer than the requested amount of items (up to zero items) in the event all requested objects are filtered out and clients should only use the presence of the continue field to determine whether more results are available. Servers may choose not to support the limit argument and will return all of the available results. If limit is specified and the continue field is empty, clients may assume that no more results are available. This field is not supported if watch is true.  The server guarantees that the objects returned when using continue will be identical to issuing a single list call without a limit - that is, no objects created, modified, or deleted after the first request is issued will be included in any subsequent continued requests. This is sometimes referred to as a consistent snapshot, and ensures that a client that is using limit to receive smaller chunks of a very large result can ensure they see all possible objects. If objects are updated during a chunked list the version of the object that was present at the time the first list result was calculated is returned. | [optional]
 **list_options_continue** | **str**| The continue option should be set when retrieving more results from the server. Since this value is server defined, clients may only use the continue value from a previous query result with identical query parameters (except for the value of continue) and the server may reject a continue value it does not recognize. If the specified continue value is no longer valid whether due to expiration (generally five to fifteen minutes) or a configuration change on the server, the server will respond with a 410 ResourceExpired error together with a continue token. If the client needs a consistent list, it must restart their list without the continue field. Otherwise, the client may send another list request with the token received with the 410 error, the server will respond with a list starting from the next key, but from the latest snapshot, which is inconsistent from the previous list results - objects that are created, modified, or deleted after the first list request will be included in the response, as long as their keys are after the \&quot;next key\&quot;.  This field is not supported when watch is true. Clients may start a watch from the last resourceVersion value returned by the server and not miss any modifications. | [optional]
 **list_options_send_initial_events** | **bool**| &#x60;sendInitialEvents&#x3D;true&#x60; may be set together with &#x60;watch&#x3D;true&#x60;. In that case, the watch stream will begin with synthetic events to produce the current state of objects in the collection. Once all such events have been sent, a synthetic \&quot;Bookmark\&quot; event  will be sent. The bookmark will report the ResourceVersion (RV) corresponding to the set of objects, and be marked with &#x60;\&quot;io.k8s.initial-events-end\&quot;: \&quot;true\&quot;&#x60; annotation. Afterwards, the watch stream will proceed as usual, sending watch events corresponding to changes (subsequent to the RV) to objects watched.  When &#x60;sendInitialEvents&#x60; option is set, we require &#x60;resourceVersionMatch&#x60; option to also be set. The semantic of the watch request is as following: - &#x60;resourceVersionMatch&#x60; &#x3D; NotOlderThan   is interpreted as \&quot;data at least as new as the provided &#x60;resourceVersion&#x60;\&quot;   and the bookmark event is send when the state is synced   to a &#x60;resourceVersion&#x60; at least as fresh as the one provided by the ListOptions.   If &#x60;resourceVersion&#x60; is unset, this is interpreted as \&quot;consistent read\&quot; and the   bookmark event is send when the state is synced at least to the moment   when request started being processed. - &#x60;resourceVersionMatch&#x60; set to any other value or unset   Invalid error is returned.  Defaults to true if &#x60;resourceVersion&#x3D;\&quot;\&quot;&#x60; or &#x60;resourceVersion&#x3D;\&quot;0\&quot;&#x60; (for backward compatibility reasons) and to false otherwise. +optional | [optional]

### Return type

[**StreamResultOfSensorSensorWatchEvent**](StreamResultOfSensorSensorWatchEvent.md)

### Authorization

[BearerToken](../README.md#BearerToken)

### HTTP request headers

 - **Content-Type**: Not defined
 - **Accept**: application/json


### HTTP response details

| Status code | Description | Response headers |
|-------------|-------------|------------------|
**200** | A successful response.(streaming responses) |  -  |
**0** | An unexpected error response. |  -  |

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

