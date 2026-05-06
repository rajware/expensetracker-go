Param(
    $VersionMajor   = (property VERSION_MAJOR "1"),
    $VersionMinor   = (property VERSION_MINOR "0"),
    $BuildNumber    = (property BUILD_NUMBER  "0"),
    $PatchString    = (property PATCH_NUMBER  "-alpha1"),
    $RegistryUser   = (property REGISTRY_USER "quay.io/rajware"),
    $ImagePlatforms = (property IMAGE_PLATFORMS "linux/amd64,linux/arm64,linux/ppc64le,linux/s390x"),
    $AddLatestTag   = $true
)

$VersionString = "$($VersionMajor).$($VersionMinor).$($BuildNumber)$($PatchString)"
$ImageName = "$($RegistryUser)/expensetracker-go"
$ImageTag = "$($ImageName):$($VersionString)"
$ImageTagLatest = "$($ImageName):latest"

$ComposePostgresTest = "deploy/compose/postgrestest.yaml"

$SourceFiles = (Get-ChildItem -Recurse -File ./cmd/tracker-web, ./internal/)

$ReleaseTargets = "linux_amd64", "linux_arm64", "darwin_amd64", "darwin_arm64", "windows_amd64", "windows_arm64"

# Synopsis: Builds the tracker-web executable
task default -Outputs out/tracker-web -Inputs $SourceFiles {
    exec {
        $env:CGO_ENABLED = 0
        go build -o $($Outputs) -ldflags "-X main.version=$($VersionString)" ./cmd/tracker-web
    }
}

# Build tasks dynamically for all release targets
foreach($target in $ReleaseTargets) {
    $outputName = "out/tracker-web_$target"
    if($target -match "windows_.*") {
        $outputName += ".exe"
    }

    # Synopsis: Builds executable for specified platform
    task $target -Outputs $outputName -Inputs $SourceFiles {
        $targetParts = $Task.Name.Split("_")
        $os = $targetParts[0]
        $arch = $targetParts[1]

        $oldCGoEnabled = $env:CGO_ENABLED
        $oldGoOS = $env:GOOS
        $oldGoArch = $env:GOARCH

        try {
            exec {
                $env:CGO_ENABLED=0
                $env:GOOS=$os
                $env:GOARCH=$arch

                Write-Host "Building for $($Task.Name) ==> $($env:GOOS)/$($env:GOARCH)..." -ForegroundColor Yellow
                go build -o $($Outputs) -ldflags "-X main.version=$($VersionString)" ./cmd/tracker-web
            }
        }
        finally {
            $env:CGO_ENABLED=$oldCGoEnabled
            $env:GOOS=$oldGoOS
            $env:GOARCH=$oldGoArch
        }
    }
}

# Synopsis: Builds the release k8s manifest for tracker sqlite variant
task release-k8smanifest-tracker-sqlite -Inputs (Get-ChildItem ./deploy/kubernetes/tracker-sqlite/*.yaml, ./deploy/kubernetes/tracker-sqlite/doc.txt) -Outputs ./out/tracker-sqlite.k8s.yaml {
    exec {
        ./scripts/build-k8smanifest.ps1 -Variant "tracker-sqlite" -Version $VersionString -Output $Outputs
    }
}

# Synopsis: Builds the release k8s manifest for tracker postgres variant
task release-k8smanifest-tracker-postgres -Inputs (Get-ChildItem ./deploy/kubernetes/tracker-postgres/*.yaml, ./deploy/kubernetes/tracker-postgres/doc.txt) -Outputs ./out/tracker-postgres.k8s.yaml {
    exec {
        ./scripts/build-k8smanifest.ps1 -Variant "tracker-postgres" -Version $VersionString -Output $Outputs
    }
}

# Synopsis: Builds the release compose manifest for tracker sqlite variant
task release-compose-tracker-sqlite -Inputs (Get-ChildItem ./deploy/compose/tracker-sqlite.yaml) -Outputs ./out/tracker-sqlite.compose.yaml {
    exec {
        ./scripts/build-composemanifest.ps1 -Variant "tracker-sqlite" -Version $VersionString -Output $Outputs
    }
}

# Synopsis: Builds the release compose manifest for tracker postgres variant
task release-compose-tracker-postgres -Inputs (Get-ChildItem ./deploy/compose/tracker-postgres.yaml) -Outputs ./out/tracker-postgres.compose.yaml {
    exec {
        ./scripts/build-composemanifest.ps1 -Variant "tracker-postgres" -Version $VersionString -Output $Outputs
    }
}

# Synopsis: Builds all release executables
task release  {
    $ReleaseTargets | ForEach-Object { Invoke-Build $_ }
    $manifestTargets = "release-k8smanifest-tracker-sqlite", "release-k8smanifest-tracker-postgres", "release-compose-tracker-sqlite", "release-compose-tracker-postgres"
    $manifestTargets | ForEach-Object { Invoke-Build $_ }
}

# Synopsis: Runs all tests
task test test-domain, test-auth-cookie, test-rest-api, test-repo-sqlite, test-repo-postgres, {

}

# Synopsis: Runs tests for domain package
task test-domain {
    exec {
        go test -v ./internal/domain
    }
}

# Synopsis: Runs tests for repository/sqlite package
task test-repo-sqlite {
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

# Synopsis: Runs tests for repository/postgres package
task test-repo-postgres compose-up-postgrestest, {
    exec {
        go test -v ./internal/repository/postgres
    }
}

# Synopsis: Brings up postgres on port 15432
task compose-up-postgrestest {
    exec {
        docker compose -p test -f $($ComposePostgresTest) up -d
    }
}

# Synopsis: Brings down postgres on port 15432
task compose-down-postgrestest {
    exec {
        docker compose -p test -f $($ComposePostgresTest) down
    }
}

# Synopsis: Brings down postgres on port 15432, deletes volumes
task compose-down-volumes-postgrestest {
    exec {
        docker compose -p test -f $($ComposePostgresTest) down --volumes
    }
}

# Synopsis: Builds local docker image
task local-image {
    exec {
        docker buildx build --load `
            -f package/docker/Dockerfile `
            --build-arg VERSION_STRING=$($VersionString) `
            -t $($ImageTagLatest) `
            .
    }
}

# Synopsis: Builds and pushes final multi-arch docker image
task final-image {
    exec {
        docker buildx build --push `
            --platform $($ImagePlatforms) `
            -f package/docker/Dockerfile `
            --build-arg VERSION_STRING=$($VersionString) `
            -t $($ImageTag) `
            .
    }

    If ($AddLatestTag) {
        exec {
            docker buildx build --push `
                --platform $($ImagePlatforms) `
                -f package/docker/Dockerfile `
                --build-arg VERSION_STRING=$($VersionString) `
                -t $($ImageTagLatest) `
                .
        }
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

# Synopsis: Cleans up local docker image
task clean-local-image {
    exec {
        docker image rm $($ImageTagLatest)
    }
}
