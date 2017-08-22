FROM ubuntu:16.04

RUN apt-get update -y \
    && apt-get install -y wget vim python ffmpeg \
    && apt-get install -y build-essential autoconf libtool pkg-config python-opengl python-imaging python-pyrex python-pyside.qtopengl idle-python2.7 qt4-dev-tools qt4-designer libqtgui4 libqtcore4 libqt4-xml libqt4-test libqt4-script libqt4-network libqt4-dbus python-qt4 python-qt4-gl libgle3 python-dev libav-tools \
    && wget -P /tmp/ https://pypi.python.org/packages/1e/8e/40c71faa24e19dab555eeb25d6c07efbc503e98b0344f0b4c3131f59947f/vnc2flv-20100207.tar.gz \
    && cd /tmp \
    && tar zfx vnc2flv-20100207.tar.gz \
    && cd /tmp/vnc2flv-20100207 \
    && python setup.py install --prefix=/usr/local/ \
    && rm -Rf /tmp/vnc2flv-20100207.tar.gz /tmp/vnc2flv-20100207

COPY run.sh /
RUN chmod +x /run.sh

CMD ["/run.sh"]