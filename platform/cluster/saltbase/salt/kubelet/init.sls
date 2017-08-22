{% if pillar.get('is_systemd') %}
{% set environment_file = '/etc/sysconfig/kubelet' %}
{% else %}
{% set environment_file = '/etc/default/kubelet' %}
{% endif %}

{{ environment_file}}:
  file.managed:
    - source: salt://kubelet/default
    - template: jinja
    - user: root
    - group: root
    - mode: 644

/usr/local/bin/kubelet:
  file.managed:
    - source: salt://kube-bins/kubelet
    - user: root
    - group: root
    - mode: 755

# The default here is that this file is blank. If this is the case, the kubelet
# won't be able to parse it as JSON and it will not be able to publish events
# to the apiserver. You'll see a single error line in the kubelet start up file
# about this.
/var/lib/kubelet/kubeconfig:
  file.managed:
    - source: salt://kubelet/kubeconfig
    - user: root
    - group: root
    - mode: 400
    - makedirs: true


{% if pillar.get('is_systemd') %}

{{ pillar.get('systemd_system_path') }}/kubelet.service:
  file.managed:
    - source: salt://kubelet/kubelet.service
    - user: root
    - group: root

# The service.running block below doesn't work reliably
# Instead we run our script which e.g. does a systemd daemon-reload
# But we keep the service block below, so it can be used by dependencies
# TODO: Fix this
fix-service-kubelet:
  cmd.wait:
    - name: /opt/kubernetes/helpers/services bounce kubelet
    - watch:
      - file: /usr/local/bin/kubelet
      - file: {{ pillar.get('systemd_system_path') }}/kubelet.service
      - file: {{ environment_file }}
      - file: /var/lib/kubelet/kubeconfig

/opt/kubernetes/helpers/kubelet-healthcheck:
  file.managed:
    - source: salt://kubelet/kubelet-healthcheck
    - user: root
    - group: root
    - mode: 755

{{ pillar.get('systemd_system_path') }}/kubelet-healthcheck.service:
  file.managed:
    - source: salt://kubelet/kubelet-healthcheck.service
    - template: jinja
    - user: root
    - group: root
    - mode: 644

{{ pillar.get('systemd_system_path') }}/kubelet-healthcheck.timer:
  file.managed:
    - source: salt://kubelet/kubelet-healthcheck.timer
    - template: jinja
    - user: root
    - group: root
    - mode: 644

# Tell systemd to load the timer
fix-systemd-kubelet-healthcheck-timer:
  cmd.wait:
    - name: /opt/kubernetes/helpers/services bounce kubelet-healthcheck.timer
    - watch:
      - file: {{ pillar.get('systemd_system_path') }}/kubelet-healthcheck.timer

# Trigger a first run of docker-healthcheck; needed because the timer fires 10s after the previous run.
fix-systemd-kubelet-healthcheck-service:
  cmd.wait:
    - name: /opt/kubernetes/helpers/services bounce kubelet-healthcheck.service
    - watch:
      - file: {{ pillar.get('systemd_system_path') }}/kubelet-healthcheck.service
    - require:
      - cmd: fix-service-kubelet

{% else %}

/etc/init.d/kubelet:
  file.managed:
    - source: salt://kubelet/initd
    - user: root
    - group: root
    - mode: 755

{% endif %}

kubelet:
  service.running:
    - enable: True
    - watch:
      - file: /usr/local/bin/kubelet
{% if pillar.get('is_systemd') %}
      - file: {{ pillar.get('systemd_system_path') }}/kubelet.service
{% else %}
      - file: /etc/init.d/kubelet
{% endif %}
{% if grains['os_family'] == 'RedHat' %}
      - file: /usr/lib/systemd/system/kubelet.service
{% endif %}
      - file: {{ environment_file }}
      - file: /var/lib/kubelet/kubeconfig
{% if pillar.get('is_systemd') %}
    - provider:
      - service: systemd
{%- endif %}
