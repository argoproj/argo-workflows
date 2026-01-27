#!/usr/bin/env python3
"""
Clean up kubebuilder and other annotations in generated swagger markdown.

Converts annotations to human-readable text in a "Validation" section.
"""

import re
import sys


def extract_and_remove(text, search_pattern, remove_pattern):
    """Extract a value using search_pattern, then remove all matches of remove_pattern."""
    match = re.search(search_pattern, text)
    value = match.group(1) if match else None
    text = re.sub(r'</br>' + remove_pattern, '', text)
    text = re.sub(remove_pattern, '', text)
    return text, value


def parse_annotations(text):
    """Extract annotations from text and return (clean_text, annotations_dict)."""
    annotations = {}

    # Boolean flags
    for name in ('optional', 'required'):
        annotations[name] = bool(re.search(rf'\+{name}\b', text))
        text = re.sub(rf'</br>\+{name}\b', '', text)
        text = re.sub(rf'\+{name}\b', '', text)

    # +default="value" or +default=value or +kubebuilder:default="value"
    match = re.search(r'\+(?:kubebuilder:)?default="([^"]*)"', text)
    if match:
        annotations['default'] = f'"{match.group(1)}"'
    else:
        match = re.search(r'\+(?:kubebuilder:)?default=(\S+?)(?:</br>|\s|\||$)', text)
        annotations['default'] = match.group(1) if match else None
    for prefix in (r'\+kubebuilder:default', r'\+default'):
        text = re.sub(rf'</br>{prefix}="[^"]*"', '', text)
        text = re.sub(rf'</br>{prefix}=\S+?(?=</br>|\s|\||$)', '', text)
        text = re.sub(rf'{prefix}="[^"]*"', '', text)
        text = re.sub(rf'{prefix}=\S+?(?=</br>|\s|\||$)', '', text)

    # +kubebuilder:validation:Enum=A;B;C (special: semicolons become commas)
    text, enum_val = extract_and_remove(
        text,
        r'\+kubebuilder:validation:Enum=([^|<\s]+)',
        r'\+kubebuilder:validation:Enum=[^|<\s]+')
    annotations['enum_values'] = enum_val.replace(';', ', ') if enum_val else None

    # Numeric kubebuilder validations
    validation_fields = [
        ('minimum', 'Minimum'),
        ('maximum', 'Maximum'),
        ('min_length', 'MinLength'),
        ('max_length', 'MaxLength'),
        ('min_items', 'MinItems'),
        ('max_items', 'MaxItems'),
    ]
    for key, validator in validation_fields:
        text, annotations[key] = extract_and_remove(
            text,
            rf'\+kubebuilder:validation:{validator}=([+-]?\d+)',
            rf'\+kubebuilder:validation:{validator}=[+-]?\d+')

    # +kubebuilder:validation:Pattern=`X`
    text, annotations['pattern'] = extract_and_remove(
        text,
        r'\+kubebuilder:validation:Pattern=`([^`]+)`',
        r'\+kubebuilder:validation:Pattern=`[^`]+`')

    # Handle +enum - just remove it (it's a type marker, not useful in docs)
    text = re.sub(r'\+enum\b', '', text)

    # Remove annotations we don't convert (internal implementation details)
    patterns_to_remove = [
        r'\+kubebuilder:validation:XValidation:[^|<]*',
        r'\+kubebuilder:validation:Type=[^|<\s]+',
        r'\+kubebuilder:validation:Schemaless',
        r'\+kubebuilder:pruning:PreserveUnknownFields',
        r'\+kubebuilder:[^|<]*',
        r'\+patchStrategy=[^|<\s]+',
        r'\+patchMergeKey=[^|<\s]+',
        r'\+listType=[^|<\s]+',
        r'\+listMapKey=[^|<\s]+',
        r'\+featureGate=[^|<\s]+',
        r'\+structType=[^|<\s]+',
        r'\+protobuf[^|<]*',
        r'\+union\b',
        r'\+k8s:[^|<\s]+',
    ]

    for pattern in patterns_to_remove:
        text = re.sub(r'</br>' + pattern, '', text)
        text = re.sub(pattern, '', text)

    return text, annotations


def build_validation_section(annotations):
    """Build a human-readable validation section from annotations."""
    parts = []

    if annotations.get('optional'):
        parts.append('Optional')

    # Fields with labels
    labeled_fields = [
        ('default', 'Default value'),
        ('enum_values', 'Allowed values'),
        ('minimum', 'Minimum value'),
        ('maximum', 'Maximum value'),
        ('min_length', 'Minimum length'),
        ('max_length', 'Maximum length'),
        ('min_items', 'Minimum items'),
        ('max_items', 'Maximum items'),
    ]
    for key, label in labeled_fields:
        if annotations.get(key) is not None:
            parts.append(f"{label}: {annotations[key]}")

    if annotations.get('pattern'):
        parts.append(f"Validation regex: `{annotations['pattern']}`")

    if not parts:
        return ''

    return '</br>*' + '; '.join(parts) + '.*'


def process_table_cell(cell):
    """Process a table cell that might contain annotations."""
    clean_text, annotations = parse_annotations(cell)
    validation_section = build_validation_section(annotations)

    # Clean up any trailing </br> before adding validation (but preserve cell spacing)
    clean_text = re.sub(r'(</br>)+\s*$', '', clean_text)

    if validation_section:
        clean_text = clean_text + validation_section

    return clean_text


def process_line(line):
    """Process a single line, handling table rows specially."""
    # Check if this is a table row
    if line.startswith('|') and '|' in line[1:]:
        # Split into cells
        cells = line.split('|')
        # Process each cell (skip first and last which are empty due to leading/trailing |)
        processed_cells = []
        for i, cell in enumerate(cells):
            if i == 0 or i == len(cells) - 1:
                processed_cells.append(cell)
            else:
                processed_cells.append(process_table_cell(cell))
        return '|'.join(processed_cells)

    # Check for standalone annotation lines like "> +kubebuilder:..." or "+kubebuilder:..."
    if re.match(r'^>?\s*\+kubebuilder:', line):
        return None  # Remove this line
    if re.match(r'^>?\s*\+enum\s*$', line):
        return None  # Remove standalone +enum lines
    if re.match(r'^>?\s*\+union\s*$', line):
        return None  # Remove standalone +union lines
    if re.match(r'^>?\s*\+structType=', line):
        return None
    if re.match(r'^>?\s*\+protobuf', line):
        return None

    # For description lines, also clean up annotations
    if '|' not in line:
        clean_text, annotations = parse_annotations(line)
        validation_section = build_validation_section(annotations)
        if validation_section:
            clean_text = clean_text.rstrip() + ' ' + validation_section.replace('</br>', ' ')
        # Clean up +enum from inline text
        clean_text = re.sub(r'\s*\+enum\s*', '', clean_text)
        return clean_text

    return line


def process_markdown(content):
    """Process the entire markdown content."""
    lines = content.split('\n')
    processed_lines = []

    for line in lines:
        processed = process_line(line)
        if processed is not None:
            processed_lines.append(processed)

    result = '\n'.join(processed_lines)

    # Also fix the any links issue
    result = re.sub(r'\[any\]\(#any\)', '`any`', result)

    # Clean up any multiple </br> (but don't touch other whitespace)
    result = re.sub(r'(</br>){2,}', '</br>', result)

    return result


def main():
    if len(sys.argv) != 2:
        print(f"Usage: {sys.argv[0]} <markdown_file>", file=sys.stderr)
        sys.exit(1)

    filepath = sys.argv[1]

    with open(filepath, 'r', encoding='utf-8') as f:
        content = f.read()

    processed = process_markdown(content)

    with open(filepath, 'w', encoding='utf-8') as f:
        f.write(processed)

    print(f"Processed {filepath}")


if __name__ == '__main__':
    main()
