from scm.models import SCM

from rest_framework import serializers


class SCMSerializer(serializers.ModelSerializer):
    """Serializer for SCM."""

    class Meta:
        model = SCM
