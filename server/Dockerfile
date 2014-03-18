FROM stackato/base

RUN apt-get install -qy redis-server

ADD . /source
ADD stackon.json /

CMD ["redis-server", "/source/redis-server.conf"]

EXPOSE 7474
VOLUME ["/var/stackato/data/config_redis"]
