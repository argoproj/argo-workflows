# coding: utf-8

"""
    Argo Workflows API

    Argo Workflows is an open source container-native workflow engine for orchestrating parallel jobs on Kubernetes. For more information, please see https://argo-workflows.readthedocs.io/en/latest/

    The version of the OpenAPI document: VERSION
    Generated by OpenAPI Generator (https://openapi-generator.tech)

    Do not edit the class manually.
"""  # noqa: E501


from __future__ import annotations
import pprint
import re  # noqa: F401
import json

from pydantic import BaseModel, ConfigDict, Field, StrictBool, StrictStr
from typing import Any, ClassVar, Dict, List, Optional
from argo_workflows.models.io_argoproj_workflow_v1alpha1_oss_lifecycle_rule import IoArgoprojWorkflowV1alpha1OSSLifecycleRule
from argo_workflows.models.secret_key_selector import SecretKeySelector
from typing import Optional, Set
from typing_extensions import Self

class IoArgoprojWorkflowV1alpha1OSSArtifactRepository(BaseModel):
    """
    OSSArtifactRepository defines the controller configuration for an OSS artifact repository
    """ # noqa: E501
    access_key_secret: Optional[SecretKeySelector] = Field(default=None, alias="accessKeySecret")
    bucket: Optional[StrictStr] = Field(default=None, description="Bucket is the name of the bucket")
    create_bucket_if_not_present: Optional[StrictBool] = Field(default=None, description="CreateBucketIfNotPresent tells the driver to attempt to create the OSS bucket for output artifacts, if it doesn't exist", alias="createBucketIfNotPresent")
    endpoint: Optional[StrictStr] = Field(default=None, description="Endpoint is the hostname of the bucket endpoint")
    key_format: Optional[StrictStr] = Field(default=None, description="KeyFormat defines the format of how to store keys and can reference workflow variables.", alias="keyFormat")
    lifecycle_rule: Optional[IoArgoprojWorkflowV1alpha1OSSLifecycleRule] = Field(default=None, alias="lifecycleRule")
    secret_key_secret: Optional[SecretKeySelector] = Field(default=None, alias="secretKeySecret")
    security_token: Optional[StrictStr] = Field(default=None, description="SecurityToken is the user's temporary security token. For more details, check out: https://www.alibabacloud.com/help/doc-detail/100624.htm", alias="securityToken")
    use_sdk_creds: Optional[StrictBool] = Field(default=None, description="UseSDKCreds tells the driver to figure out credentials based on sdk defaults.", alias="useSDKCreds")
    __properties: ClassVar[List[str]] = ["accessKeySecret", "bucket", "createBucketIfNotPresent", "endpoint", "keyFormat", "lifecycleRule", "secretKeySecret", "securityToken", "useSDKCreds"]

    model_config = ConfigDict(
        populate_by_name=True,
        validate_assignment=True,
        protected_namespaces=(),
    )


    def to_str(self) -> str:
        """Returns the string representation of the model using alias"""
        return pprint.pformat(self.model_dump(by_alias=True))

    def to_json(self) -> str:
        """Returns the JSON representation of the model using alias"""
        # TODO: pydantic v2: use .model_dump_json(by_alias=True, exclude_unset=True) instead
        return json.dumps(self.to_dict())

    @classmethod
    def from_json(cls, json_str: str) -> Optional[Self]:
        """Create an instance of IoArgoprojWorkflowV1alpha1OSSArtifactRepository from a JSON string"""
        return cls.from_dict(json.loads(json_str))

    def to_dict(self) -> Dict[str, Any]:
        """Return the dictionary representation of the model using alias.

        This has the following differences from calling pydantic's
        `self.model_dump(by_alias=True)`:

        * `None` is only added to the output dict for nullable fields that
          were set at model initialization. Other fields with value `None`
          are ignored.
        """
        excluded_fields: Set[str] = set([
        ])

        _dict = self.model_dump(
            by_alias=True,
            exclude=excluded_fields,
            exclude_none=True,
        )
        # override the default output from pydantic by calling `to_dict()` of access_key_secret
        if self.access_key_secret:
            _dict['accessKeySecret'] = self.access_key_secret.to_dict()
        # override the default output from pydantic by calling `to_dict()` of lifecycle_rule
        if self.lifecycle_rule:
            _dict['lifecycleRule'] = self.lifecycle_rule.to_dict()
        # override the default output from pydantic by calling `to_dict()` of secret_key_secret
        if self.secret_key_secret:
            _dict['secretKeySecret'] = self.secret_key_secret.to_dict()
        return _dict

    @classmethod
    def from_dict(cls, obj: Optional[Dict[str, Any]]) -> Optional[Self]:
        """Create an instance of IoArgoprojWorkflowV1alpha1OSSArtifactRepository from a dict"""
        if obj is None:
            return None

        if not isinstance(obj, dict):
            return cls.model_validate(obj)

        _obj = cls.model_validate({
            "accessKeySecret": SecretKeySelector.from_dict(obj["accessKeySecret"]) if obj.get("accessKeySecret") is not None else None,
            "bucket": obj.get("bucket"),
            "createBucketIfNotPresent": obj.get("createBucketIfNotPresent"),
            "endpoint": obj.get("endpoint"),
            "keyFormat": obj.get("keyFormat"),
            "lifecycleRule": IoArgoprojWorkflowV1alpha1OSSLifecycleRule.from_dict(obj["lifecycleRule"]) if obj.get("lifecycleRule") is not None else None,
            "secretKeySecret": SecretKeySelector.from_dict(obj["secretKeySecret"]) if obj.get("secretKeySecret") is not None else None,
            "securityToken": obj.get("securityToken"),
            "useSDKCreds": obj.get("useSDKCreds")
        })
        return _obj


