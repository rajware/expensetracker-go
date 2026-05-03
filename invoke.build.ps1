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

# Synopsis: Builds the tracker-web executable
task out-tracker-web -Outputs out/tracker-web -Inputs (Get-ChildItem -Recurse -File ./cmd/tracker-web, ./internal/) {
    exec {
        $env:CGO_ENABLED=0
        go build -o $($Outputs) -ldflags "-X main.version=$($VersionString)" ./cmd/tracker-web
    }
}

# Synopsis: Runs all tests
task test test-domain, test-repository-sqlite, test-auth-cookie, test-rest-api, {

}

# Synopsis: Runs tests for domain package
task test-domain {
    exec {
        go test -v ./internal/domain
    }
}

# Synopsis: Runs tests for repository/sqlite package
task test-repository-sqlite {
    exec {
        go test -v ./internal/repository/sqlite
    }
}

# Synopsis: Runs tests for auth/cookie package
task test-auth-cookie {
    exec {
        go test -v ./internal/auth/cookie
    }
}

# Synopsis: Runs tests for api/rest package
task test-rest-api {
    exec {
        go test -v ./internal/api/rest
    }
}

# Synopsis: Cleans up all output
task clean clean-out, clean-data, {

}

# Synopsis: Cleans up output directory
task clean-out {
    Remove-Item -Recurse -Force ./out -ErrorAction Ignore
}

# Synopsis: Cleans up data directory
task clean-data {
    Remove-Item -Recurse -Force ./data -ErrorAction Ignore
}
