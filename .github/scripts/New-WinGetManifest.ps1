[CmdletBinding()]
param(
    [Parameter(Mandatory)]
    [ValidatePattern('^[0-9]+(?:\.[0-9]+)*(?:[-+][0-9A-Za-z.-]+)?$')]
    [string] $Version,

    [Parameter(Mandatory)]
    [uri] $ReleaseUrl,

    [Parameter(Mandatory)]
    [ValidateScript({ Test-Path $_ -PathType Container })]
    [string] $AssetDirectory,

    [Parameter(Mandatory)]
    [string] $OutputDirectory
)

$packageIdentifier = 'Aicage.acp-wsl'
$packageName = 'acp-wsl'
$publisher = 'Aicage'
$manifestVersion = '1.12.0'
$releaseBaseUrl = $ReleaseUrl.ToString().Replace('/tag/', '/download/')
$assets = @(
    @{ Architecture = 'x64'; FileName = 'acp-wsl-windows-amd64.zip' }
    @{ Architecture = 'arm64'; FileName = 'acp-wsl-windows-arm64.zip' }
)

New-Item -ItemType Directory -Force -Path $OutputDirectory | Out-Null

$installerEntries = foreach ($asset in $assets) {
    $assetPath = Join-Path $AssetDirectory $asset.FileName
    if (-not (Test-Path $assetPath -PathType Leaf)) {
        throw "Release asset not found: $assetPath"
    }
    $hash = (Get-FileHash -Algorithm SHA256 -LiteralPath $assetPath).Hash
    @"
  - Architecture: $($asset.Architecture)
    InstallerUrl: $releaseBaseUrl/$($asset.FileName)
    InstallerSha256: $hash
"@
}
$installerEntries = $installerEntries -join [Environment]::NewLine

$versionManifest = @"
# yaml-language-server: `$schema=https://aka.ms/winget-manifest.version.$manifestVersion.schema.json
PackageIdentifier: $packageIdentifier
PackageVersion: $Version
DefaultLocale: en-US
ManifestType: version
ManifestVersion: $manifestVersion
"@

$defaultLocaleManifest = @"
# yaml-language-server: `$schema=https://aka.ms/winget-manifest.defaultLocale.$manifestVersion.schema.json
PackageIdentifier: $packageIdentifier
PackageVersion: $Version
PackageLocale: en-US
Publisher: $publisher
PublisherUrl: https://github.com/aicage
PublisherSupportUrl: https://github.com/aicage/acp-wsl/issues
PackageName: $packageName
PackageUrl: https://github.com/aicage/acp-wsl
License: MIT
LicenseUrl: https://github.com/aicage/acp-wsl/blob/main/LICENSE
ShortDescription: Converts ACP paths between Windows clients and WSL agents.
ReleaseNotesUrl: $ReleaseUrl
ManifestType: defaultLocale
ManifestVersion: $manifestVersion
"@

$installerManifest = @"
# yaml-language-server: `$schema=https://aka.ms/winget-manifest.installer.$manifestVersion.schema.json
PackageIdentifier: $packageIdentifier
PackageVersion: $Version
InstallerType: zip
NestedInstallerType: portable
NestedInstallerFiles:
  - RelativeFilePath: acp-wsl.exe
    PortableCommandAlias: acp-wsl
Installers:
$installerEntries
ManifestType: installer
ManifestVersion: $manifestVersion
"@

Set-Content `
    -LiteralPath (Join-Path $OutputDirectory "$packageIdentifier.yaml") `
    -Value $versionManifest `
    -Encoding utf8
Set-Content `
    -LiteralPath (Join-Path $OutputDirectory "$packageIdentifier.locale.en-US.yaml") `
    -Value $defaultLocaleManifest `
    -Encoding utf8
Set-Content `
    -LiteralPath (Join-Path $OutputDirectory "$packageIdentifier.installer.yaml") `
    -Value $installerManifest `
    -Encoding utf8
