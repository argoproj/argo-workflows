{% if pillar.get('ax_tcp_keepalive', '').lower() == 'true' %}
net.ipv4.tcp_keepalive_intvl:
  sysctl.present:
    - value: 10

net.ipv4.tcp_keepalive_probes:
  sysctl.present:
    - value: 6

net.ipv4.tcp_keepalive_time:
  sysctl.present:
    - value: 60
{% endif %}
