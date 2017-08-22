{% if grains['roles'][0] == 'kubernetes-master' -%}
/etc/td-agent/td-agent.conf:
  file.managed:
    - source: salt://fluentd-master-ax/td-agent.conf
    - user: root
    - group: root
    - mode: 644
    - makedirs: true
    - dir_mode: 755
/etc/kubernetes/manifests/fluentd-master-ax.yaml:
  file.managed:
    - source: salt://fluentd-master-ax/fluentd-master-ax.yaml
    - user: root
    - group: root
    - mode: 644
    - makedirs: true
    - dir_mode: 755
{% endif %}
