@echo off
REM This script copies lvs.exe from the current directory to C:\Windows\System32

SET SRC=%~dp0lvs.exe
SET DEST=C:\Windows\System32\lvs.exe

IF NOT EXIST "%SRC%" (
    echo lvs.exe not found in the current directory.
    exit /b 1
)

copy /Y "%SRC%" "%DEST%"
IF %ERRORLEVEL% EQU 0 (
    echo lvs.exe successfully copied to C:\Windows\System32
) ELSE (
    echo Failed to copy lvs.exe. Try running as Administrator.
)
