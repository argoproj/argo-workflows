#!/usr/bin/env python3
"""
Apply urllib3 compatibility fix to rest.py after OpenAPI Generator generates it.
This fixes compatibility with urllib3 >= 2.0 which removed getheader/getheaders methods.
"""
import sys
import re

def fix_rest_py(file_path):
    with open(file_path, 'r') as f:
        content = f.read()
    
    # Fix getheaders method
    getheaders_pattern = r'(    def getheaders\(self\):\s*""".*?""")\s*return self\.urllib3_response\.getheaders\(\)'
    getheaders_replacement = r'''\1
        # Compatibility with both old and new urllib3 versions
        if hasattr(self.urllib3_response, 'getheaders'):
            return self.urllib3_response.getheaders()
        elif hasattr(self.urllib3_response, 'headers'):
            return dict(self.urllib3_response.headers)
        else:
            return {}'''
    
    content = re.sub(getheaders_pattern, getheaders_replacement, content, flags=re.DOTALL)
    
    # Fix getheader method
    getheader_pattern = r'(    def getheader\(self, name, default=None\):\s*""".*?""")\s*return self\.urllib3_response\.getheader\(name, default\)'
    getheader_replacement = r'''\1
        # Compatibility with both old and new urllib3 versions
        if hasattr(self.urllib3_response, 'getheader'):
            return self.urllib3_response.getheader(name, default)
        elif hasattr(self.urllib3_response, 'headers'):
            return self.urllib3_response.headers.get(name.lower(), default)
        else:
            return default'''
    
    content = re.sub(getheader_pattern, getheader_replacement, content, flags=re.DOTALL)
    
    with open(file_path, 'w') as f:
        f.write(content)

if __name__ == '__main__':
    if len(sys.argv) != 2:
        print(f"Usage: {sys.argv[0]} <path_to_rest.py>")
        sys.exit(1)
    
    fix_rest_py(sys.argv[1])

