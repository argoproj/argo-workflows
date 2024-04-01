# IoArgoprojEventsV1alpha1HDFSEventSource


## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**addresses** | **List[str]** |  | [optional] 
**check_interval** | **str** |  | [optional] 
**filter** | [**IoArgoprojEventsV1alpha1EventSourceFilter**](IoArgoprojEventsV1alpha1EventSourceFilter.md) |  | [optional] 
**hdfs_user** | **str** | HDFSUser is the user to access HDFS file system. It is ignored if either ccache or keytab is used. | [optional] 
**krb_c_cache_secret** | [**SecretKeySelector**](SecretKeySelector.md) |  | [optional] 
**krb_config_config_map** | [**ConfigMapKeySelector**](ConfigMapKeySelector.md) |  | [optional] 
**krb_keytab_secret** | [**SecretKeySelector**](SecretKeySelector.md) |  | [optional] 
**krb_realm** | **str** | KrbRealm is the Kerberos realm used with Kerberos keytab It must be set if keytab is used. | [optional] 
**krb_service_principal_name** | **str** | KrbServicePrincipalName is the principal name of Kerberos service It must be set if either ccache or keytab is used. | [optional] 
**krb_username** | **str** | KrbUsername is the Kerberos username used with Kerberos keytab It must be set if keytab is used. | [optional] 
**metadata** | **Dict[str, str]** |  | [optional] 
**type** | **str** |  | [optional] 
**watch_path_config** | [**IoArgoprojEventsV1alpha1WatchPathConfig**](IoArgoprojEventsV1alpha1WatchPathConfig.md) |  | [optional] 

## Example

```python
from argo_workflows.models.io_argoproj_events_v1alpha1_hdfs_event_source import IoArgoprojEventsV1alpha1HDFSEventSource

# TODO update the JSON string below
json = "{}"
# create an instance of IoArgoprojEventsV1alpha1HDFSEventSource from a JSON string
io_argoproj_events_v1alpha1_hdfs_event_source_instance = IoArgoprojEventsV1alpha1HDFSEventSource.from_json(json)
# print the JSON string representation of the object
print(IoArgoprojEventsV1alpha1HDFSEventSource.to_json())

# convert the object into a dict
io_argoproj_events_v1alpha1_hdfs_event_source_dict = io_argoproj_events_v1alpha1_hdfs_event_source_instance.to_dict()
# create an instance of IoArgoprojEventsV1alpha1HDFSEventSource from a dict
io_argoproj_events_v1alpha1_hdfs_event_source_form_dict = io_argoproj_events_v1alpha1_hdfs_event_source.from_dict(io_argoproj_events_v1alpha1_hdfs_event_source_dict)
```
[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


