import unittest
import main

class TestSwagger2JsonSchemaTransform(unittest.TestCase):
    
    def test_string_value_unchanged(self):
        self.assertEqual({'foo', 'bar'}, main.walk_transform([], {'foo', 'bar'}))
    
    def test_intxx_formats_removed(self):
        self.assertEqual(
            main.walk_transform([], {
                'int32': {'format': 'int32','foo': 'bar'},
                'int64': {'format': 'int64','foo': 'bar'},
                'int': {'format': 'int','foo': 'bar'}
            }),
            {
                'int32': {'foo': 'bar'},
                'int64': {'foo': 'bar'},
                'int': {'format': 'int','foo': 'bar'}
            }
        )
    
    def test_param_int_or_num_applied(self):
        self.assertEqual(
            main.walk_transform([], {
                'definitions': {
                    'io.argoproj.workflow.v1alpha1.Parameter': {
                        'properties': {
                            'default': {
                                'type': ['string']
                            },
                            'value': {
                                'type': ['string']
                            }
                        }
                    }
                }
            }),
            {
                'definitions': {
                    'io.argoproj.workflow.v1alpha1.Parameter': {
                        'properties': {
                            'default': {
                                'type': ['string', 'number']
                            },
                            'value': {
                                'type': ['string', 'number']
                            }
                        }
                    }
                }
            }
        )

    def test_k8s_str_or_int_fixed(self):
        self.assertEqual(
            main.walk_transform([], {
                'definitions': {
                    'io.k8s.apimachinery.pkg.util.intstr.IntOrString': {
                        'type': ['string'],
                        'format': 'int-or-string'
                    }
                }
            }),
            {
                'definitions': {
                    'io.k8s.apimachinery.pkg.util.intstr.IntOrString': {
                        'type': ['string', 'integer']
                    }
                }
            }
        )