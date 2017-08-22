"""
FixtureClass structure
"""
import json
import logging
import re
import uuid

import voluptuous
from voluptuous import Schema, Any, Required, Optional, All, Length, Invalid, REMOVE_EXTRA

from ax.exceptions import AXApiInvalidParam
from .util import humanize_error, new_status_detail
from .common import FixtureClassStatus

logger = logging.getLogger(__name__)

"""
Attribute definition JSON representation:
{
    "type": "string",
    "flags": "required",
    "options": [
        "m3.medium",
        "m3.large",
        "m3.xlarge",
        "m3.2xlarge"
    ],
    "default": "m3.2xlarge"
}
"""
attribute_definition_schema = Schema({
    Required('type') : Any('int', 'string', 'bool', 'float'),
    Optional('flags') : str,
    Optional('options') : list,
    Optional('default'): Any(str, int, float, bool, list),
    Optional('on_success'): str,
    Optional('on_failure'): str
})

def bool_validator(val):
    """Accepts bool as a value, or the strings 'true' and 'false'"""
    if isinstance(val, bool):
        return val
    if isinstance(val, str):
        val = val.lower().strip()
        if val not in ['true', 'false']:
            raise Invalid("Boolean values must be either 'true' or 'false'")
        return val == 'true'
    else:
        raise Invalid("'{}' is not a valid boolean value".format(val))

def string_validator(val):
    """Accepts any primitive type as a value and recasts to string"""
    if type(val) in [int, str, float]:
        return str(val)
    if isinstance(val, bool):
        return 'true' if val else 'false'
    raise Invalid("'{}' is not a valid string".format(val))

def int_validator(val):
    """Accepts int or string as value and recasts to int"""
    if isinstance(val, float) or isinstance(val, bool):
        raise Invalid("'{}' is not an integer".format(val))
    try:
        return int(val)
    except ValueError:
        raise Invalid("'{}' is not an integer".format(val))

def float_validator(val):
    """Accepts int or string as value and recasts to float"""
    try:
        return float(val)
    except ValueError:
        raise Invalid("'{}' is not a float".format(val))

data_type_validators = {
    'string': string_validator,
    'bool': bool_validator,
    'int': int_validator,
    'float': float_validator
}

def build_option_validator(data_type, options):
    """Returns a validator which checks if the incoming value is in the set of valid options"""
    valid_options = set()
    # check and normalize the options to agree with the data type
    for option in options:
        try:
            if data_type == 'int':
                valid_options.add(int(option))
            elif data_type == 'string':
                valid_options.add(str(option))
            elif data_type == 'bool':
                valid_options.add(bool_validator(option))
            elif data_type == 'float':
                valid_options.add(float(option))
        except Exception:
            raise AXApiInvalidParam("'{}' is not of data type {}".format(option, data_type))

    def option_validator(val):
        if val not in valid_options:
            raise Invalid("'{}' is not a valid option. Expected: {}".format(val, ', '.join(valid_options)))
        return val
    return option_validator


class FixtureClass(object):
    """A Fixture Class

    JSON representation:
    {
        "id" : "d896ec3c-b739-423a-a2ce-9ea0bca0f407",
        "name":"test-fixture",
        "description":"fixture for test purposes",
        "repo": "https://repo.org/company/prod.git",
        "branch": "master",
        "attributes":{
            "instance_type":{
                "type":"string",
                "flags":"required"
                "options":[
                    "m3.medium",
                    "m3.large",
                    "m3.xlarge",
                    "m3.2xlarge"
                ],
                "default":"m3.2xlarge"
            },
            "ip_address":{
                "type":"string",
                "required":true,
                "mutable":true
            }
        },
        "actions":{
            "create":{
                "template":"test-fixture-action",
                "parameters":{
                    "ACTION":"create",
                    "INSTANCE_TYPE":"%%attributes.instance_type%%"
                }
            },
            "delete":{
                "template":"test-fixture-action",
                "parameters":{
                    "ACTION":"delete"
                }
            },
            "snapshot":{
                "template":"test-fixture-action",
                "parameters":{
                    "ACTION":"snapshot"
                }
            }
        }
        "action_templates":{
            "test-fixture-action": {
                "type":"service_template",
                "subtype":"container",
                "name":"test-fixture-action",
                "cost":0,
                "container":{
                    "resources":{
                        "mem_mib":200,
                        "cpu_cores":0.05
                    },
                    "image":"debian:8.5",
                    "docker_options":"",
                    "command":"echo 'performing action %%ACTION%% instance_type: %%INSTANCE_TYPE%%'; sleep 60; echo '{\"ip_address\": \"1.2.3.4\"}' \u003e /tmp/fix_attrs.json"
                },
                "inputs":{
                    "parameters":{
                        "ACTION":{},
                        "INSTANCE_TYPE":{}
                    }
                },
                "outputs":{
                    "artifacts":{
                        "fixture_attributes":{
                            "path":"/tmp/fix_attrs.json"
                        }
                    }
                },
                "jobs_fail":0,
                "jobs_success":0
            }
        }
    }
    """
    FIELDS = frozenset(['id', 'name', 'description', 'repo', 'branch', 'attributes', 'actions', 'action_templates', 'revision', 'status', 'status_detail'])
    JSON_FIELDS = frozenset(['attributes', 'actions', 'action_templates', 'status_detail'])
    TEMPLATE_FIELDS = FIELDS - frozenset(['id', 'action_templates', 'status', 'status_detail'])

    def __init__(self):
        """Fixture class object model"""
        self.id = None
        self.name = None
        self.description = None
        self.repo = None
        self.branch = None
        self._attributes = {}
        self.actions = {}
        self.action_templates = {}
        self._schema = None
        self._attribute_schemas = None
        self.status = None
        self.status_detail = None

    def __str__(self):
        return 'FixtureClass {} (id:{})'.format(self.name, self.id)

    @property
    def attributes(self):
        return self._attributes

    @attributes.setter
    def attributes(self, attributes):
        # Build a schema validator for all attribute
        schema = {}
        attribute_schemas = {}
        for attr_name, attr_def in attributes.items():
            key, val = FixtureClass.build_schema(attr_name, attr_def)
            schema[key] = val
            attribute_schemas[attr_name] = Schema({key: val}, extra=REMOVE_EXTRA)
        self._schema = Schema(schema, extra=REMOVE_EXTRA)
        self._attributes = attributes
        self._attribute_schemas = attribute_schemas

    @staticmethod
    def build_schema(attr_name, attr_def):
        """Dynamically builds a voluptuous schema to validate an instance attributes
        :returns: a voluptuous key name and validator"""
        try:
            validator = data_type_validators[attr_def['type']]
        except KeyError:
            raise AXApiInvalidParam("Unsupported data type: {}".format(attr_def['type']))

        if 'flags' in attr_def:
            flags = set([f.strip() for f in attr_def['flags'].split(',')])
        else:
            flags = set()

        option_validator = None
        if 'options' in attr_def:
            option_validator = build_option_validator(attr_def['type'], attr_def['options'])

        if 'array' in flags:
            validator = [validator]
            if 'required' in flags:
                key = Required(attr_name)
                if option_validator:
                    val = All(validator, [option_validator], Length(min=1))
                else:
                    val = All(validator, Length(min=1))
            else:
                key = Optional(attr_name, default=list)
                if option_validator:
                    val = All(validator, [option_validator])
                else:
                    val = validator
        else:
            if 'required' in flags:
                if attr_def['type'] == 'string':
                    # disallow empty strings if it is required
                    validator = All(validator, Length(min=1))
                key = Required(attr_name)
                if option_validator:
                    val = All(validator, option_validator)
                else:
                    val = validator
            else:
                key = Optional(attr_name)
                if option_validator:
                    val = All(validator, option_validator)
                else:
                    val = validator
        return key, val

    def validate(self, attributes):
        """Validate a fixture json attributes against the class definition"""
        try:
            return self._schema(attributes)
        except voluptuous.Error as err:
            raise AXApiInvalidParam(humanize_error(str(err)))

    def validate_attribute(self, attribute, val):
        """Validate a single attribute value against the class definition"""
        try:
            validated_doc = self._attribute_schemas[attribute]({attribute:val})
            return validated_doc[attribute]
        except (KeyError, voluptuous.Error) as err:
            raise AXApiInvalidParam(humanize_error(str(err)))

    def json(self, fields=None):
        """Returns dictionary representation of the class"""
        res = {}
        for field in FixtureClass.FIELDS:
            if fields and field not in fields:
                continue
            res[field] = getattr(self, field)
        return res

    def axdbdoc(self, fields=None):
        """Normalizes json document for storage to axdb"""
        doc = {
            'id': self.id,
        }
        for field, val in self.json().items():
            if fields is None or field in fields:
                doc[field] = val
        FixtureClass.serialize_axdbdoc(doc)
        return doc

    @classmethod
    def from_template(cls, template, action_templates):
        """Returns a FixtureClass from a template, generating a new UUID
        :param template: fixture template dictionary
        :param action_templates: mapping of action names to service template
        """
        fixclass = FixtureClass()
        fixclass.id = str(uuid.uuid4())
        for field in cls.TEMPLATE_FIELDS:
            if field == 'actions' and 'actions' not in template:
                # actions might be nil for a fixture class with no actions
                continue
            setattr(fixclass, field, template[field])
        fixclass.action_templates = action_templates
        fixclass.status = FixtureClassStatus.ACTIVE
        fixclass.status_detail = new_status_detail(FixtureClassStatus.ACTIVE, "Fixture class is active")
        return fixclass

    @classmethod
    def deserialize_axdbdoc(cls, doc):
        """Deserializes a axdb dictionary to a FixtureClass object"""
        fixclass = FixtureClass()
        for field in cls.FIELDS:
            if doc[field] in [None, '']:
                # Temporary hack to allow a few clusters with transient schema to upgrade. Remove when we merge to master
                if field == 'status_detail':
                    fixclass.status_detail = new_status_detail(FixtureClassStatus.ACTIVE, "Fixture class is active")
                    continue
                if field == 'status':
                    fixclass.status = FixtureClassStatus.ACTIVE
                    continue

            if field in cls.JSON_FIELDS:
                setattr(fixclass, field, json.loads(doc[field]))
            else:
                setattr(fixclass, field, doc[field])
        return fixclass

    @classmethod
    def serialize_axdbdoc(cls, doc):
        """Serializes a dictionary to a json structure suitable for storing to axdb"""
        for json_field in cls.JSON_FIELDS:
            if json_field in doc:
                val = doc.get(json_field)
                if val is not None:
                    doc[json_field] = json.dumps(val)
        return doc


def compare_fixture_classes(old_cls, new_cls):
    """
    Compares two fixture classes and returns a tuple of (new attributes, modified attributes, deleted attributes)
    :param old_cls: old class
    :param new_cls: new class
    """
    new_attrs = []
    modified_attributes = []
    deleted_attrs = list(set(old_cls.attributes.keys()) - set(new_cls.attributes.keys()))

    for attr_name, new_attr in new_cls.attributes.items():
        if attr_name not in old_cls.attributes:
            new_attrs.append(attr_name)
        elif new_attr != old_cls.attributes[attr_name]:
            modified_attributes.append(attr_name)

    if any([new_attrs, modified_attributes, deleted_attrs]):
        logger.info("Attributes changed: new: %s. modified: %s. deleted: %s", new_attrs, modified_attributes, deleted_attrs)
    else:
        logger.info("No change in attributes detected")
    return new_attrs, modified_attributes, deleted_attrs
