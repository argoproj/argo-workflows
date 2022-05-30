"""
    Argo Workflows API

    Argo Workflows is an open source container-native workflow engine for orchestrating parallel jobs on Kubernetes. For more information, please see https://argoproj.github.io/argo-workflows/  # noqa: E501

    The version of the OpenAPI document: VERSION
    Generated by: https://openapi-generator.tech
"""


import re  # noqa: F401
import sys  # noqa: F401

from argo_workflows.model_utils import (  # noqa: F401
    ApiTypeError,
    ModelComposed,
    ModelNormal,
    ModelSimple,
    cached_property,
    change_keys_js_to_python,
    convert_js_args_to_python_args,
    date,
    datetime,
    file_type,
    none_type,
    validate_get_composed_info,
)
from ..model_utils import OpenApiModel
from argo_workflows.exceptions import ApiAttributeError


def lazy_import():
    from argo_workflows.model.io_argoproj_workflow_v1alpha1_archive_strategy import IoArgoprojWorkflowV1alpha1ArchiveStrategy
    from argo_workflows.model.io_argoproj_workflow_v1alpha1_artifact_gc import IoArgoprojWorkflowV1alpha1ArtifactGC
    from argo_workflows.model.io_argoproj_workflow_v1alpha1_artifactory_artifact import IoArgoprojWorkflowV1alpha1ArtifactoryArtifact
    from argo_workflows.model.io_argoproj_workflow_v1alpha1_gcs_artifact import IoArgoprojWorkflowV1alpha1GCSArtifact
    from argo_workflows.model.io_argoproj_workflow_v1alpha1_git_artifact import IoArgoprojWorkflowV1alpha1GitArtifact
    from argo_workflows.model.io_argoproj_workflow_v1alpha1_hdfs_artifact import IoArgoprojWorkflowV1alpha1HDFSArtifact
    from argo_workflows.model.io_argoproj_workflow_v1alpha1_http_artifact import IoArgoprojWorkflowV1alpha1HTTPArtifact
    from argo_workflows.model.io_argoproj_workflow_v1alpha1_oss_artifact import IoArgoprojWorkflowV1alpha1OSSArtifact
    from argo_workflows.model.io_argoproj_workflow_v1alpha1_raw_artifact import IoArgoprojWorkflowV1alpha1RawArtifact
    from argo_workflows.model.io_argoproj_workflow_v1alpha1_s3_artifact import IoArgoprojWorkflowV1alpha1S3Artifact
    globals()['IoArgoprojWorkflowV1alpha1ArchiveStrategy'] = IoArgoprojWorkflowV1alpha1ArchiveStrategy
    globals()['IoArgoprojWorkflowV1alpha1ArtifactGC'] = IoArgoprojWorkflowV1alpha1ArtifactGC
    globals()['IoArgoprojWorkflowV1alpha1ArtifactoryArtifact'] = IoArgoprojWorkflowV1alpha1ArtifactoryArtifact
    globals()['IoArgoprojWorkflowV1alpha1GCSArtifact'] = IoArgoprojWorkflowV1alpha1GCSArtifact
    globals()['IoArgoprojWorkflowV1alpha1GitArtifact'] = IoArgoprojWorkflowV1alpha1GitArtifact
    globals()['IoArgoprojWorkflowV1alpha1HDFSArtifact'] = IoArgoprojWorkflowV1alpha1HDFSArtifact
    globals()['IoArgoprojWorkflowV1alpha1HTTPArtifact'] = IoArgoprojWorkflowV1alpha1HTTPArtifact
    globals()['IoArgoprojWorkflowV1alpha1OSSArtifact'] = IoArgoprojWorkflowV1alpha1OSSArtifact
    globals()['IoArgoprojWorkflowV1alpha1RawArtifact'] = IoArgoprojWorkflowV1alpha1RawArtifact
    globals()['IoArgoprojWorkflowV1alpha1S3Artifact'] = IoArgoprojWorkflowV1alpha1S3Artifact


class IoArgoprojWorkflowV1alpha1ArtifactPaths(ModelNormal):
    """NOTE: This class is auto generated by OpenAPI Generator.
    Ref: https://openapi-generator.tech

    Do not edit the class manually.

    Attributes:
      allowed_values (dict): The key is the tuple path to the attribute
          and the for var_name this is (var_name,). The value is a dict
          with a capitalized key describing the allowed value and an allowed
          value. These dicts store the allowed enum values.
      attribute_map (dict): The key is attribute name
          and the value is json key in definition.
      discriminator_value_class_map (dict): A dict to go from the discriminator
          variable value to the discriminator class name.
      validations (dict): The key is the tuple path to the attribute
          and the for var_name this is (var_name,). The value is a dict
          that stores validations for max_length, min_length, max_items,
          min_items, exclusive_maximum, inclusive_maximum, exclusive_minimum,
          inclusive_minimum, and regex.
      additional_properties_type (tuple): A tuple of classes accepted
          as additional properties values.
    """

    allowed_values = {
    }

    validations = {
    }

    @cached_property
    def additional_properties_type():
        """
        This must be a method because a model may have properties that are
        of type self, this must run after the class is loaded
        """
        lazy_import()
        return (bool, date, datetime, dict, float, int, list, str, none_type,)  # noqa: E501

    _nullable = False

    @cached_property
    def openapi_types():
        """
        This must be a method because a model may have properties that are
        of type self, this must run after the class is loaded

        Returns
            openapi_types (dict): The key is attribute name
                and the value is attribute type.
        """
        lazy_import()
        return {
            'name': (str,),  # noqa: E501
            'archive': (IoArgoprojWorkflowV1alpha1ArchiveStrategy,),  # noqa: E501
            'archive_logs': (bool,),  # noqa: E501
            'artifact_gc': (IoArgoprojWorkflowV1alpha1ArtifactGC,),  # noqa: E501
            'artifactory': (IoArgoprojWorkflowV1alpha1ArtifactoryArtifact,),  # noqa: E501
            '_from': (str,),  # noqa: E501
            'from_expression': (str,),  # noqa: E501
            'gcs': (IoArgoprojWorkflowV1alpha1GCSArtifact,),  # noqa: E501
            'git': (IoArgoprojWorkflowV1alpha1GitArtifact,),  # noqa: E501
            'global_name': (str,),  # noqa: E501
            'hdfs': (IoArgoprojWorkflowV1alpha1HDFSArtifact,),  # noqa: E501
            'http': (IoArgoprojWorkflowV1alpha1HTTPArtifact,),  # noqa: E501
            'mode': (int,),  # noqa: E501
            'optional': (bool,),  # noqa: E501
            'oss': (IoArgoprojWorkflowV1alpha1OSSArtifact,),  # noqa: E501
            'path': (str,),  # noqa: E501
            'raw': (IoArgoprojWorkflowV1alpha1RawArtifact,),  # noqa: E501
            'recurse_mode': (bool,),  # noqa: E501
            's3': (IoArgoprojWorkflowV1alpha1S3Artifact,),  # noqa: E501
            'sub_path': (str,),  # noqa: E501
        }

    @cached_property
    def discriminator():
        return None


    attribute_map = {
        'name': 'name',  # noqa: E501
        'archive': 'archive',  # noqa: E501
        'archive_logs': 'archiveLogs',  # noqa: E501
        'artifact_gc': 'artifactGC',  # noqa: E501
        'artifactory': 'artifactory',  # noqa: E501
        '_from': 'from',  # noqa: E501
        'from_expression': 'fromExpression',  # noqa: E501
        'gcs': 'gcs',  # noqa: E501
        'git': 'git',  # noqa: E501
        'global_name': 'globalName',  # noqa: E501
        'hdfs': 'hdfs',  # noqa: E501
        'http': 'http',  # noqa: E501
        'mode': 'mode',  # noqa: E501
        'optional': 'optional',  # noqa: E501
        'oss': 'oss',  # noqa: E501
        'path': 'path',  # noqa: E501
        'raw': 'raw',  # noqa: E501
        'recurse_mode': 'recurseMode',  # noqa: E501
        's3': 's3',  # noqa: E501
        'sub_path': 'subPath',  # noqa: E501
    }

    read_only_vars = {
    }

    _composed_schemas = {}

    @classmethod
    @convert_js_args_to_python_args
    def _from_openapi_data(cls, name, *args, **kwargs):  # noqa: E501
        """IoArgoprojWorkflowV1alpha1ArtifactPaths - a model defined in OpenAPI

        Args:
            name (str): name of the artifact. must be unique within a template's inputs/outputs.

        Keyword Args:
            _check_type (bool): if True, values for parameters in openapi_types
                                will be type checked and a TypeError will be
                                raised if the wrong type is input.
                                Defaults to True
            _path_to_item (tuple/list): This is a list of keys or values to
                                drill down to the model in received_data
                                when deserializing a response
            _spec_property_naming (bool): True if the variable names in the input data
                                are serialized names, as specified in the OpenAPI document.
                                False if the variable names in the input data
                                are pythonic names, e.g. snake case (default)
            _configuration (Configuration): the instance to use when
                                deserializing a file_type parameter.
                                If passed, type conversion is attempted
                                If omitted no type conversion is done.
            _visited_composed_classes (tuple): This stores a tuple of
                                classes that we have traveled through so that
                                if we see that class again we will not use its
                                discriminator again.
                                When traveling through a discriminator, the
                                composed schema that is
                                is traveled through is added to this set.
                                For example if Animal has a discriminator
                                petType and we pass in "Dog", and the class Dog
                                allOf includes Animal, we move through Animal
                                once using the discriminator, and pick Dog.
                                Then in Dog, we will make an instance of the
                                Animal class but this time we won't travel
                                through its discriminator because we passed in
                                _visited_composed_classes = (Animal,)
            archive (IoArgoprojWorkflowV1alpha1ArchiveStrategy): [optional]  # noqa: E501
            archive_logs (bool): ArchiveLogs indicates if the container logs should be archived. [optional]  # noqa: E501
            artifact_gc (IoArgoprojWorkflowV1alpha1ArtifactGC): [optional]  # noqa: E501
            artifactory (IoArgoprojWorkflowV1alpha1ArtifactoryArtifact): [optional]  # noqa: E501
            _from (str): From allows an artifact to reference an artifact from a previous step. [optional]  # noqa: E501
            from_expression (str): FromExpression, if defined, is evaluated to specify the value for the artifact. [optional]  # noqa: E501
            gcs (IoArgoprojWorkflowV1alpha1GCSArtifact): [optional]  # noqa: E501
            git (IoArgoprojWorkflowV1alpha1GitArtifact): [optional]  # noqa: E501
            global_name (str): GlobalName exports an output artifact to the global scope, making it available as '{{io.argoproj.workflow.v1alpha1.outputs.artifacts.XXXX}} and in workflow.status.outputs.artifacts. [optional]  # noqa: E501
            hdfs (IoArgoprojWorkflowV1alpha1HDFSArtifact): [optional]  # noqa: E501
            http (IoArgoprojWorkflowV1alpha1HTTPArtifact): [optional]  # noqa: E501
            mode (int): mode bits to use on this file, must be a value between 0 and 0777 set when loading input artifacts.. [optional]  # noqa: E501
            optional (bool): Make Artifacts optional, if Artifacts doesn't generate or exist. [optional]  # noqa: E501
            oss (IoArgoprojWorkflowV1alpha1OSSArtifact): [optional]  # noqa: E501
            path (str): Path is the container path to the artifact. [optional]  # noqa: E501
            raw (IoArgoprojWorkflowV1alpha1RawArtifact): [optional]  # noqa: E501
            recurse_mode (bool): If mode is set, apply the permission recursively into the artifact if it is a folder. [optional]  # noqa: E501
            s3 (IoArgoprojWorkflowV1alpha1S3Artifact): [optional]  # noqa: E501
            sub_path (str): SubPath allows an artifact to be sourced from a subpath within the specified source. [optional]  # noqa: E501
        """

        _check_type = kwargs.pop('_check_type', True)
        _spec_property_naming = kwargs.pop('_spec_property_naming', False)
        _path_to_item = kwargs.pop('_path_to_item', ())
        _configuration = kwargs.pop('_configuration', None)
        _visited_composed_classes = kwargs.pop('_visited_composed_classes', ())

        self = super(OpenApiModel, cls).__new__(cls)

        if args:
            raise ApiTypeError(
                "Invalid positional arguments=%s passed to %s. Remove those invalid positional arguments." % (
                    args,
                    self.__class__.__name__,
                ),
                path_to_item=_path_to_item,
                valid_classes=(self.__class__,),
            )

        self._data_store = {}
        self._check_type = _check_type
        self._spec_property_naming = _spec_property_naming
        self._path_to_item = _path_to_item
        self._configuration = _configuration
        self._visited_composed_classes = _visited_composed_classes + (self.__class__,)

        self.name = name
        for var_name, var_value in kwargs.items():
            if var_name not in self.attribute_map and \
                        self._configuration is not None and \
                        self._configuration.discard_unknown_keys and \
                        self.additional_properties_type is None:
                # discard variable.
                continue
            setattr(self, var_name, var_value)
        return self

    required_properties = set([
        '_data_store',
        '_check_type',
        '_spec_property_naming',
        '_path_to_item',
        '_configuration',
        '_visited_composed_classes',
    ])

    @convert_js_args_to_python_args
    def __init__(self, name, *args, **kwargs):  # noqa: E501
        """IoArgoprojWorkflowV1alpha1ArtifactPaths - a model defined in OpenAPI

        Args:
            name (str): name of the artifact. must be unique within a template's inputs/outputs.

        Keyword Args:
            _check_type (bool): if True, values for parameters in openapi_types
                                will be type checked and a TypeError will be
                                raised if the wrong type is input.
                                Defaults to True
            _path_to_item (tuple/list): This is a list of keys or values to
                                drill down to the model in received_data
                                when deserializing a response
            _spec_property_naming (bool): True if the variable names in the input data
                                are serialized names, as specified in the OpenAPI document.
                                False if the variable names in the input data
                                are pythonic names, e.g. snake case (default)
            _configuration (Configuration): the instance to use when
                                deserializing a file_type parameter.
                                If passed, type conversion is attempted
                                If omitted no type conversion is done.
            _visited_composed_classes (tuple): This stores a tuple of
                                classes that we have traveled through so that
                                if we see that class again we will not use its
                                discriminator again.
                                When traveling through a discriminator, the
                                composed schema that is
                                is traveled through is added to this set.
                                For example if Animal has a discriminator
                                petType and we pass in "Dog", and the class Dog
                                allOf includes Animal, we move through Animal
                                once using the discriminator, and pick Dog.
                                Then in Dog, we will make an instance of the
                                Animal class but this time we won't travel
                                through its discriminator because we passed in
                                _visited_composed_classes = (Animal,)
            archive (IoArgoprojWorkflowV1alpha1ArchiveStrategy): [optional]  # noqa: E501
            archive_logs (bool): ArchiveLogs indicates if the container logs should be archived. [optional]  # noqa: E501
            artifact_gc (IoArgoprojWorkflowV1alpha1ArtifactGC): [optional]  # noqa: E501
            artifactory (IoArgoprojWorkflowV1alpha1ArtifactoryArtifact): [optional]  # noqa: E501
            _from (str): From allows an artifact to reference an artifact from a previous step. [optional]  # noqa: E501
            from_expression (str): FromExpression, if defined, is evaluated to specify the value for the artifact. [optional]  # noqa: E501
            gcs (IoArgoprojWorkflowV1alpha1GCSArtifact): [optional]  # noqa: E501
            git (IoArgoprojWorkflowV1alpha1GitArtifact): [optional]  # noqa: E501
            global_name (str): GlobalName exports an output artifact to the global scope, making it available as '{{io.argoproj.workflow.v1alpha1.outputs.artifacts.XXXX}} and in workflow.status.outputs.artifacts. [optional]  # noqa: E501
            hdfs (IoArgoprojWorkflowV1alpha1HDFSArtifact): [optional]  # noqa: E501
            http (IoArgoprojWorkflowV1alpha1HTTPArtifact): [optional]  # noqa: E501
            mode (int): mode bits to use on this file, must be a value between 0 and 0777 set when loading input artifacts.. [optional]  # noqa: E501
            optional (bool): Make Artifacts optional, if Artifacts doesn't generate or exist. [optional]  # noqa: E501
            oss (IoArgoprojWorkflowV1alpha1OSSArtifact): [optional]  # noqa: E501
            path (str): Path is the container path to the artifact. [optional]  # noqa: E501
            raw (IoArgoprojWorkflowV1alpha1RawArtifact): [optional]  # noqa: E501
            recurse_mode (bool): If mode is set, apply the permission recursively into the artifact if it is a folder. [optional]  # noqa: E501
            s3 (IoArgoprojWorkflowV1alpha1S3Artifact): [optional]  # noqa: E501
            sub_path (str): SubPath allows an artifact to be sourced from a subpath within the specified source. [optional]  # noqa: E501
        """

        _check_type = kwargs.pop('_check_type', True)
        _spec_property_naming = kwargs.pop('_spec_property_naming', False)
        _path_to_item = kwargs.pop('_path_to_item', ())
        _configuration = kwargs.pop('_configuration', None)
        _visited_composed_classes = kwargs.pop('_visited_composed_classes', ())

        if args:
            raise ApiTypeError(
                "Invalid positional arguments=%s passed to %s. Remove those invalid positional arguments." % (
                    args,
                    self.__class__.__name__,
                ),
                path_to_item=_path_to_item,
                valid_classes=(self.__class__,),
            )

        self._data_store = {}
        self._check_type = _check_type
        self._spec_property_naming = _spec_property_naming
        self._path_to_item = _path_to_item
        self._configuration = _configuration
        self._visited_composed_classes = _visited_composed_classes + (self.__class__,)

        self.name = name
        for var_name, var_value in kwargs.items():
            if var_name not in self.attribute_map and \
                        self._configuration is not None and \
                        self._configuration.discard_unknown_keys and \
                        self.additional_properties_type is None:
                # discard variable.
                continue
            setattr(self, var_name, var_value)
            if var_name in self.read_only_vars:
                raise ApiAttributeError(f"`{var_name}` is a read-only attribute. Use `from_openapi_data` to instantiate "
                                     f"class with read only attributes.")
