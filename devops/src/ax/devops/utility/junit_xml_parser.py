#!/usr/bin/env python
#
# Copyright 2015-2017 Applatix, Inc.  All rights reserved.
#
# -*- coding: utf-8 -*-
#
"""
Parse a junit report file into a family of objects
JUnit XML specification: http://llg.cubic.org/docs/junit/
"""

import xml.etree.ElementTree as ElementTree

TESTCASE = 'testcase'
TESTSUITE = 'testsuite'
TESTSUITES = 'testsuites'

ERROR = 'error'
FAILURE = 'failure'
PASS = 'pass'
SKIPPED = 'skipped'

SYSTEM_ERR = 'system-err'
SYSTEM_OUT = 'system-out'


class JunitXml(object):

    def __init__(self):
        pass


class Suites(JunitXml):
    """
    Test Suites
    """
    def __init__(self):
        """
        :return:
        """
        super(Suites, self).__init__()
        self.name = None  # xml.name
        self.duration = 0  # xml.time, in seconds
        self.num_of_tests = 0  # xml.tests
        self.num_of_errors = 0  # xml.errors
        self.num_of_failures = 0  # xml.failures
        self.num_of_disabled = 0  # xml.disabled


class Suite(JunitXml):
    """
    Test suite, usually only one suite per report
    """
    def __init__(self):
        """
        :return:
        """
        super(Suite, self).__init__()
        self.id = None  # xml.id
        self.name = None  # xml.name
        self.duration = 0  # xml.time, in seconds

        self.num_of_tests = 0  # xml.tests
        self.num_of_errors = 0  # xml.errors
        self.num_of_failures = 0  # xml.failures
        self.num_of_disabled = 0  # xml.disabled
        self.num_of_skipped = 0  # xml.skipped

        self.hostname = None  # xml.hostname
        self.package = None  # xml.package
        self.timestamp = None  # xml.timestamp
        self.properties = {}  # xml.properties


class Case(JunitXml):
    """
    Test case
    """
    STATUS = (PASS, FAILURE, SKIPPED, ERROR)

    def __init__(self):
        super(Case, self).__init__()

        self.name = None  # xml.name
        self.assertions = None  # xml.assertions
        self.classname = None  # xml.classname
        self.duration = 0  # xml.time
        self.stderr = None  # xml.system-err
        self.stdout = None  # xml.system-out

        self.testsuite = None
        self.testsuites = None

        self.message = None
        self.status = PASS

        assert self.status in self.STATUS

    def failed(self):
        """
        Return True if this test failed
        :return:
        """
        return self.status == FAILURE


class Parser(object):
    """
    Parse a single junit xml report
    """

    def __init__(self, filename):
        """
        Parse the file, or a string
        :param filename:
        :return:
        """
        tree = ElementTree.parse(filename)

        if isinstance(tree, ElementTree.Element):
            self.root = tree
        else:
            self.root = tree.getroot()

    def run(self):
        """
        populate the report from the xml
        :return:
        """
        result_suites = None
        result_suite_list = []
        result_case_list = []

        if self.root.tag == TESTSUITES:
            result_suites = Parser.parse_suites(self.root)
            suite_list = [each for each in self.root]
        elif self.root.tag == TESTSUITE:
            suite_list = [self.root]
        else:
            raise ValueError('Incompatible Junit XML format %s', self.root.tag)

        for each_suite_element in suite_list:
            if each_suite_element.tag != TESTSUITE:
                continue  # Skip, only search test suite element

            suite_obj = Parser.parse_suite(each_suite_element)
            result_suite_list.append(suite_obj)

            for each_case_element in each_suite_element:
                if each_case_element.tag != TESTCASE:
                    continue  # Skip, only search test case element

                case_obj = Parser.parse_case(each_case_element)
                # Add testsuites/testsuite info
                case_obj.testsuite = suite_obj.name
                case_obj.testsuites = result_suites.name if result_suites else None

                result_case_list.append(case_obj)

        return result_suites, result_suite_list, result_case_list

    @staticmethod
    def parse_suites(element):
        assert element.tag == TESTSUITES, 'Wrong parser for {}'.format(element.tag)
        sus = Suites()
        sus.name = element.attrib['name']
        return sus

    @staticmethod
    def parse_suite(element):
        assert element.tag == TESTSUITE, 'Wrong parser for {}'.format(element.tag)
        su = Suite()
        su.name = element.attrib['name']
        return su

    @staticmethod
    def parse_case(element):
        assert element.tag == TESTCASE, 'Wrong parser for {}'.format(element.tag)
        case = Case()
        case.name = element.attrib['name']
        case.assertions = element.attrib.get('assertions')
        case.classname = element.attrib.get('classname')
        case.duration = float(element.attrib.get('time', '0'))

        for child in element:
            if child.tag in [SKIPPED, FAILURE, ERROR]:
                case.status = child.tag
                case.message = child.attrib.get('message', None)
            elif child.tag == SYSTEM_OUT:
                case.stdout = child.text
            elif child.tag == SYSTEM_ERR:
                case.stderr = child.text
        return case
