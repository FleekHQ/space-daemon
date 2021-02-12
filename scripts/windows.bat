@ECHO off

REM Delete credentials associated with space daemon
FOR /F "tokens=2" %%c IN ('@CMDKEY /list ^| @FINDSTR space:space') DO (@CMDKEY /delete:%%c)

REM Delete folders created by space daemon
@RD /S /Q "C:\\Users\\%USERNAME%\\.fleek-ipfs"
@RD /S /Q "C:\\Users\\%USERNAME%\\.fleek-space"
@RD /S /Q "C:\\Users\\%USERNAME%\\.buckd"

@ECHO Removed credentials and deleted space daemon associated folders successfully.