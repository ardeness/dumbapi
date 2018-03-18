#!/bin/bash

if [[ $# -eq 0 ]] ; then
	echo "Usage : ./setup_api.sh num_of_api_instance"
	exit 0
fi

if [[ $1 -lt 0 ]]; then
	echo "Usage : ./setup_api.sh num_of_api_instance(0 or greater)"
	exit 0
fi

mkdir -p conf.d

# check dummapi docker image is exists
dummapiimageid=$(sudo docker images | grep dummapi)

# if not, build image
if [[ -z $dummapiimageid ]]; then
	# check host has go compiler
	if ! type "go" 2> /dev/null; then
		sudo docker build . -t dummapi -f Dockerfile.withoutgo
	else
		make
		sudo docker build . -t dummapi -f Dockerfile.withgo
	fi
fi


# check redis container is running
redisid=$(sudo docker ps -a | grep dummredis | awk '{print $1}')

if [[ ! -z $redisid ]]; then
	sudo docker rm -f $redisid
fi

dummapiids=$(sudo docker ps -a | grep dummapi | awk '{print $1}')

for cid in $dummapiids;
do
	sudo docker rm -f $cid
done

sudo docker run -d --name dummredis redis

# get redis server ip
redisip=$(sudo docker inspect --format="{{ .NetworkSettings.IPAddress }}" dummredis)

# popup dummapi containers
apicount=$1

rm -f conf.d/default.conf
touch conf.d/default.conf

echo "upstream dummapigroup {" >> conf.d/default.conf
echo "server localhost:10000 down;" >> conf.d/default.conf
for i in `seq 1 $1`;
do
	sudo docker run -d -e HOSTNAME=host$i -e REDISSERVER=$redisip -h host$i --name dummapihost$i dummapi
	hostip=$(sudo docker inspect --format="{{ .NetworkSettings.IPAddress }}" dummapihost$i)
	echo "server $hostip:10000;" >> conf.d/default.conf
done

echo "}" >> conf.d/default.conf
cat default.conf >> conf.d/default.conf

# re pop-up main contaer
nginxid=$(sudo docker ps -a | grep dummnginx | awk '{print $1}')

if [[ ! -z $nginxid ]]; then
	sudo docker rm -f $nginxid
fi

sudo docker run -d -p 80:80 -v `pwd`/conf.d:/etc/nginx/conf.d:ro --name dummnginx nginx
