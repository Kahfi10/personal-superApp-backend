# Script untuk menjalankan API dengan mudah
Set-Location $PSScriptRoot
& .\.venv\Scripts\Activate.ps1
uvicorn main:app --reload
