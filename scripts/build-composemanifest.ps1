[CmdletBinding()]
param (
    [Parameter(Mandatory = $true)]
    [string]$Variant,

    [Parameter(Mandatory = $true)]
    [string]$Version,

    [Parameter(Mandatory = $true)]
    [string]$Output
)


# Process the files
 
$FilePath = "deploy/compose/$Variant.yaml"
    
if (-Not (Test-Path $FilePath)) {
    Write-Error "Input File not found: $FilePath"
    exit 1
}
# Return the content to the pipeline
$Content = (Get-Content -Raw $FilePath).Trim()

# 1. -replace performs the regex substitution
# 2. Set-Content writes it to the file
$Content | ForEach-Object {
    $_ -replace 'image: quay.io/rajware/expensetracker-go:latest', "image: quay.io/rajware/expensetracker-go:$Version"
} | Set-Content -Path $Output

Write-Host "Successfully generated $Output for $Variant ($Version)" -ForegroundColor Cyan
