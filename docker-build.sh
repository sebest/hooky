#!/bin/sh

NAME=hooky
TAG=`git rev-parse --short=8 HEAD`

TMPD=`mktemp -d /tmp/$NAME-build.XXXXXX` || exit 1
git archive --format=tar $TAG | (cd $TMPD ; tar -xpf -)

pushd $TMPD

BV=$RANDOM
cp dist/Dockerfile.build Dockerfile
docker build --rm -t $NAME-build:$BV .
docker run --rm $NAME-build:$BV > $NAME-build.tar.gz
docker rmi $NAME-build:$BV

cp dist/Dockerfile.dist Dockerfile
docker build --rm -t sebest/$NAME:$TAG .

rm -rf $TMPD
