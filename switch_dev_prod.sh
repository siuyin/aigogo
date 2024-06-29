#! /bin/bash

if [ -z "$1" ]; then
       echo "Usage $0 [dev|prod]"
       exit
fi

if [ "$1" = "dev" ]; then
	grep --exclude="$0" -l -r ' DEV' . | while read filename; do
		echo $filename
		sed -i '/\/\/ DEV/s/\/\///' $filename
	done
 	grep --exclude="$0" -l -r ' PROD' . | while read filename; do
		echo $filename
		sed -i '/\/\/ PROD/s/^/\/\/ /' $filename
	done

	exit
fi

grep --exclude="$0" -l -r ' PROD' . | while read filename; do
	echo $filename
	sed -i '/\/\/ PROD/s/\/\///' $filename
done
grep --exclude="$0" -l -r ' DEV' . | while read filename; do
	echo $filename
	sed -i '/\/\/ DEV/s/^/\/\/ /' $filename
done
