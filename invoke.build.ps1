Param(
    $VersionMajor   = (property VERSION_MAJOR "1"),
    $VersionMinor   = (property VERSION_MINOR "0"),
    $BuildNumber    = (property BUILD_NUMBER  "0"),
    $PatchString    = (property PATCH_NUMBER  "-alpha1"),
    $RegistryUser   = (property REGISTRY_USER "quay.io/rajware"),
    $ImagePlatforms = (property IMAGE_PLATFORMS "linux/amd64,linux/arm64,linux/ppc64le,linux/s390x")
)

$VersionString = "$($VersionMajor).$($VersionMinor).$($BuildNumber)$($PatchString)"
$ImageName = "$($RegistryUser)/expensetracker-go"
$ImageTag = "$($ImageName):$($VersionString)"
$ImageTagLatest = "$($ImageName):latest"

# Synopsis: Runs tests on the domain package
task test-domain {
    exec {
        go test -v ./internal/domain
    }
}
