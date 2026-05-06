[CmdletBinding()]
param (
    [Parameter(Mandatory = $true)]
    [string]$Variant,

    [Parameter(Mandatory = $true)]
    [string]$Version,

    [Parameter(Mandatory = $true)]
    [string]$Output
)

# Get the doc.txt file for the variant
$DocPath = "deploy/kubernetes/$Variant/doc.txt"
$Header = if (Test-Path $DocPath) { Get-Content -Raw $DocPath } else { "" }

# Define file sets based on Variant
$VariantFiles = switch ($Variant) {
    'tracker-sqlite' { "pvc", "secret", "dep", "svc", "ingress" }
    'tracker-postgres' { "pvc", "secret", "db-dep", "db-svc", "fe-dep", "fe-svc", "ingress" }
    Default {
        Write-Error "Invalid variant: $Variant"
        exit 1
    }
}

# Process the files
$Content = foreach ($i in $VariantFiles) {
    $FilePath = "deploy/kubernetes/$Variant/$Variant-$i.yaml"
    
    if (Test-Path $FilePath) {
        # Return the separator and the content to the pipeline
        "---"
        (Get-Content -Raw $FilePath).Trim()
    }
    else {
        Write-Warning "File not found: $FilePath"
    }
}

# 1. Output $Header first
# 2. Process $Content as follows, and output:
#    2.1.  Skip(1) removes the very first '---'
#    2.2. -replace performs the regex substitution
# 3. Set-Content writes the combined output to the file
$(
    $Header.Trim()
    $Content | Select-Object -Skip 1 | ForEach-Object {
        $_ -replace 'image: quay.io/rajware/expensetracker-go:latest', "image: quay.io/rajware/expensetracker-go:$Version"
    }
) | Set-Content -Path $Output

Write-Host "Successfully generated $Output for $Variant ($Version)" -ForegroundColor Cyan