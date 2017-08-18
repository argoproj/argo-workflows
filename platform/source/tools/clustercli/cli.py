# -*- coding: utf-8 -*-
#
# Copyright 2015-2017 Applatix, Inc. All rights reserved.
#

import re

from anytree import Node
from anytree.iterators import PreOrderIter
from abc import ABCMeta, abstractmethod
from future.utils import with_metaclass
from prompt_toolkit import prompt
from prompt_toolkit.auto_suggest import AutoSuggestFromHistory
from prompt_toolkit.contrib.completers import WordCompleter
from prompt_toolkit.validation import Validator, ValidationError
from prompt_toolkit.token import Token
from prompt_toolkit.styles import style_from_dict
from prompt_toolkit.shortcuts import print_tokens


class AbstractPrompt(with_metaclass(ABCMeta, object)):
    """
    This class implements an abstract implementation for prompt based CLI
    All subclasses need to do is implement the following methods
    
    get_root(): returns the root of a graph of type Node
    get_history(): returns an object of type History. This is used to store the history of the prompt for suggestions
    
    Each Node of the graph represents a question (or prompt) and must have the following properties
    Node("somename",
         prompt=The unicode string for the prompt (REQUIRED)
         default=unicode string for default value OR a function that returns a unicode with prototype f(Node) : unicode (OPTIONAL)
         validator=A regex string OR function that raises any exception on error f(Document) : raises Exception on fail else pass (OPTIONAL)
         values=An array of unicode strings (OPTIONAL). If values and validator are both specified validator is used (values is ignored)
                Values automatically populates autocomplete array and hence makes input very convenient.
         help=A unicode string that displays some help message on toolbar (OPTIONAL)
         function=A function that should return True if the children nodes should be followed else False. I takes as input the user
                  input string. f(unicode): (unicode, boolean)
                  This function can also morph the user input if so desired. (OPTIONAL)
         parent=The Node object that is parent of current Node
         )
    This class will add another attribute "value" to this graph after each user input
    """

    @abstractmethod
    def get_root(self):
        pass

    @abstractmethod
    def get_history(self):
        pass

    @abstractmethod
    def get_header(self):
        pass

    def get_value(self, node_name, default=None):
        """
        This function will get the value of the node from the tree and return 
        its value.
        :param node_name: Name of node
        :type node_name: str or unicode
        :param default: The default value if node does not have a value yet
        :type default: None or str or unicode
        :return: Value of node or default
        :rtype: str or unicode
        """
        try:
            node = self.get_node(node_name)
            return getattr(node, "value", default)
        except ValueError:
            pass
        return default

    def get_node(self, node_name):
        for node in PreOrderIter(self.get_root()):
            if node.name == node_name:
                return node

        raise ValueError("Node {} not found".format(node_name))

    def run_cli(self):
        print_tokens([(Token, '\n'), (Token.Green, self.get_header()), (Token, '\n')], style=style_from_dict({Token.Green: '#ansigreen'}))
        stack = [self.get_root()]
        while stack:
            node = stack.pop()
            cont = self.prompt_node(node)
            if cont:
                stack.extend([x for x in node.children])

    def prompt_node(self, node):

        style = style_from_dict({
            Token.Toolbar: '#ffffff bg:#333333',
        })

        user_input = prompt(u'\n' + node.prompt + u' > ',
                            default=self.node_default(node),
                            history=self.get_history(),
                            auto_suggest=AutoSuggestFromHistory(),
                            completer=self.node_completions(node),
                            validator=self.node_validator(node),
                            get_bottom_toolbar_tokens=self.node_toolbar(node),
                            style=style
                            )

        # now call the user defined function
        # if that function returns True, the children are traversed otherwise the walk at this node
        user_input, cont = (self.node_f(node))(user_input)
        setattr(node, "value", user_input)
        return cont

    @staticmethod
    def node_default(node):
        default = getattr(node, "default", u'')
        if callable(default):
            return unicode(default(node))
        else:
            return unicode(default)

    @staticmethod
    def node_completions(node):
        vals = getattr(node, "values", None)
        if callable(vals):
            arr_vals = vals(node)
            setattr(node, "values", arr_vals)
            return WordCompleter(arr_vals)
        elif isinstance(vals, list):
            return WordCompleter(vals)
        else:
            return None

    @staticmethod
    def node_validator(node):
        return GenericValidator(node)

    @staticmethod
    def node_toolbar(node):
        toolbar = getattr(node, "help", None)
        if not toolbar:
            return lambda(x) : [(Token.Toolbar, "Default values are prepoulated. You can press backspace to type a different value or tab to see list of values")]
        return lambda (x): [(Token.Toolbar, toolbar)]

    @staticmethod
    def node_f(node):
        f = getattr(node, "function", None)
        if not f:
            return lambda x: (x, True)
        else:
            return f


class GenericValidator(Validator):

    def __init__(self, node):
        """
        :type node: Node
        """
        self.node = node
        self.validator = getattr(node, "validator", None)
        self.values = getattr(node, "values", None)

    def validate(self, document):
        if self.validator:
            if isinstance(self.validator, str) or isinstance(self.validator, unicode):
                try:
                    matched = re.match(self.validator, document.text)
                    if not matched:
                        raise ValidationError(message="{} does not match validator {}".format(document.text, self.validator),
                                              cursor_position=len(document.text))
                except Exception as e:
                    raise ValidationError(message="{} fails validation due to {}".format(document.text, e),
                                          cursor_position=len(document.text))

            elif callable(self.validator):
                try:
                    self.validator(document.text)
                except Exception as e:
                    raise ValidationError(message="{} fails validation due to {}".format(document.text, e))
            else:
                assert False, "Validator can be a regex string or a function"

        elif self.values:
            if document.text not in self.values:
                raise ValidationError(message="{} is not a valid input. Possible values are {}".format(document.text, self.values),
                                      cursor_position=len(document.text))
        else:
            pass
