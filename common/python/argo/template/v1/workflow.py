#!/usr/bin/env python
# -*- coding: utf-8 -*-
#
# Copyright 2015-2017 Applatix, Inc. All rights reserved.
#

from future.utils import iteritems

from .base import BaseTemplate
from argo.parser.parser import ObjectParser
from .common import Inputs, Outputs
from .container import ContainerTemplate


class WorkflowTemplateRef(ObjectParser):

    def __init__(self):
        self.template = None

        self._object_parser_fields = {
            "template": None
        }
        super(WorkflowTemplateRef, self).__init__()


class WorkflowTemplate(BaseTemplate):
    
    def __init__(self):

        # call the super constructor to get base class field map
        super(WorkflowTemplate, self).__init__()
        self.inputs = {}
        self.outputs = {}
        self.steps = []
        self.fixtures = []
        self.volumes = {}
        self.artifact_tags = []
        self.termination_policy = None

        # append to base class field map
        fields = {
            "inputs": Inputs.fqcn(),
            "outputs": Outputs.fqcn(),
            "steps": None,
        }

        self._object_parser_fields.update(fields)

    def parse(self, data):

        # first call generic parse from parent
        super(WorkflowTemplate, self).parse(data)

        # now process steps specially
        new_steps = []
        for serial_step in self.steps:
            new_serial_step = {}
            for parallel_step_name, parallel_step_val in iteritems(serial_step):
                if "template" in parallel_step_val:
                    # not inlined
                    new_parallel_step = WorkflowTemplateRef()
                    new_parallel_step.parse(parallel_step_val)
                    new_serial_step[parallel_step_name] = new_parallel_step
                else:
                    # inlined
                    new_parallel_step = ContainerTemplate()
                    new_parallel_step.parse(parallel_step_val)
                    new_serial_step[parallel_step_name] = new_parallel_step

            new_steps.append(new_serial_step)

        self.steps = new_steps

