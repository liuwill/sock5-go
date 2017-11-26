#!/bin/bash

echo "Testing Start"

cd ./bin
go run main.go &
PID=$!

cd ../
echo `go test`

if ps -p $PID
then
    echo "Sever Still Running"
    kill -9 $PID
fi

ps -p $PID

echo "test ok"
exit 0