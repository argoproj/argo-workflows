#!/usr/bin/env python3
from pathlib import Path

api_client_path = str(Path(__file__).parent / 'client' / 'argo_workflows' / 'api_client.py')
api_client_lines = open(api_client_path, 'r').readlines()
line_to_insert_at = -1

for i, line in enumerate(api_client_lines):
    if 'deserialized_data = validate_and_convert_types' in line:
        line_to_insert_at = i

if line_to_insert_at == -1:
    raise Exception('could not find line to insert SDK fix at')

fix = """
        # TODO: fix this deserialization hack"
        from datetime import timezone
        now = datetime.now(tz=timezone.utc)
        received_data['status']['startedAt'], received_data['status']['finishedAt'] = now, now
"""
api_client_lines.insert(line_to_insert_at, fix)
with open(api_client_path, 'w') as f:
    f.write(''.join(api_client_lines))
