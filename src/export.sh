#!/bin/bash

DIR="$( cd -P "$( dirname "$0" )" && pwd )"

rm ${DIR}/../log/* 
rm ${DIR}/../run/*
rm ${DIR}/../hash.txt

echo "SHA1= "`openssl sha1 ${DIR}/../realtime | awk '{print $NF}'` >> ${DIR}/../hash.txt

cd ${DIR}/../ && git-archive-all $1 --extra hash.txt --extra bin/realtime --extra log/ --extra run/ --prefix RealTime/ -v
