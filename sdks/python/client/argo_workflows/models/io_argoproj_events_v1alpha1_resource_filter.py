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

from datetime import datetime
from pydantic import BaseModel, ConfigDict, Field, StrictBool, StrictStr
from typing import Any, ClassVar, Dict, List, Optional
from argo_workflows.models.io_argoproj_events_v1alpha1_selector import IoArgoprojEventsV1alpha1Selector
from typing import Optional, Set
from typing_extensions import Self

class IoArgoprojEventsV1alpha1ResourceFilter(BaseModel):
    """
    IoArgoprojEventsV1alpha1ResourceFilter
    """ # noqa: E501
    after_start: Optional[StrictBool] = Field(default=None, alias="afterStart")
    created_by: Optional[datetime] = Field(default=None, description="Time is a wrapper around time.Time which supports correct marshaling to YAML and JSON.  Wrappers are provided for many of the factory methods that the time package offers.", alias="createdBy")
    fields: Optional[List[IoArgoprojEventsV1alpha1Selector]] = None
    labels: Optional[List[IoArgoprojEventsV1alpha1Selector]] = None
    prefix: Optional[StrictStr] = None
    __properties: ClassVar[List[str]] = ["afterStart", "createdBy", "fields", "labels", "prefix"]

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
        """Create an instance of IoArgoprojEventsV1alpha1ResourceFilter from a JSON string"""
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
        # override the default output from pydantic by calling `to_dict()` of each item in fields (list)
        _items = []
        if self.fields:
            for _item in self.fields:
                if _item:
                    _items.append(_item.to_dict())
            _dict['fields'] = _items
        # override the default output from pydantic by calling `to_dict()` of each item in labels (list)
        _items = []
        if self.labels:
            for _item in self.labels:
                if _item:
                    _items.append(_item.to_dict())
            _dict['labels'] = _items
        return _dict

    @classmethod
    def from_dict(cls, obj: Optional[Dict[str, Any]]) -> Optional[Self]:
        """Create an instance of IoArgoprojEventsV1alpha1ResourceFilter from a dict"""
        if obj is None:
            return None

        if not isinstance(obj, dict):
            return cls.model_validate(obj)

        _obj = cls.model_validate({
            "afterStart": obj.get("afterStart"),
            "createdBy": obj.get("createdBy"),
            "fields": [IoArgoprojEventsV1alpha1Selector.from_dict(_item) for _item in obj["fields"]] if obj.get("fields") is not None else None,
            "labels": [IoArgoprojEventsV1alpha1Selector.from_dict(_item) for _item in obj["labels"]] if obj.get("labels") is not None else None,
            "prefix": obj.get("prefix")
        })
        return _obj


