@echo off
setlocal

rem Clean up previous server executable
if exist server.exe del server.exe

rem Build the Go server
go build -o server.exe

rem Start the server instances
start /B server.exe -port=8001
start /B server.exe -port=8002 -api=1
start /B server.exe -port=8003 

rem Wait for a few seconds
timeout /t 2
@REM echo >>> start test

rem Send test requests
start /B curl "http://localhost:9999/api?key=Tom"
start /B curl "http://localhost:9999/api?key=Tom"
start /B curl "http://localhost:9999/api?key=Tom"

rem Wait for a few seconds before exiting
@REM timeout /t 5