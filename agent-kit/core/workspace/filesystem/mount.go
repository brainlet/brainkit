// Ported from: packages/core/src/workspace/filesystem/mount.ts
package filesystem

// =============================================================================
// Filesystem Mount Configuration
// =============================================================================

// FilesystemMountConfig holds the configuration returned by a filesystem
// that supports mounting into a sandbox.
//
// The Type field determines how the sandbox should mount the filesystem:
//   - "local": Direct bind mount from a local directory
//   - "s3": Mount via s3fs (requires s3fs-fuse)
//   - "gcs": Mount via gcsfuse (requires gcsfuse)
//   - Custom strings for provider-specific implementations
type FilesystemMountConfig struct {
	// Type identifies the mount strategy.
	Type string

	// LocalPath is the host path for "local" type mounts.
	LocalPath string

	// Bucket is the bucket name for cloud storage mounts.
	Bucket string

	// Region is the cloud region (e.g., for S3).
	Region string

	// Prefix is the key prefix within the bucket.
	Prefix string

	// Credentials holds authentication credentials for cloud mounts.
	Credentials map[string]string

	// Extra holds additional provider-specific configuration.
	Extra map[string]interface{}
}

// MountResult holds the result of a mount operation.
type MountResult struct {
	// Success indicates if the mount succeeded.
	Success bool
	// MountPath is the path where the filesystem was mounted.
	MountPath string
	// Error holds the error message if the mount failed.
	Error string
	// Unavailable indicates the mount tool is not installed (warning, not error).
	Unavailable bool
}

// FilesystemIcon represents an icon identifier for UI display.
type FilesystemIcon string

// Predefined filesystem icon constants.
const (
	FilesystemIconFolder     FilesystemIcon = "folder"
	FilesystemIconCloud      FilesystemIcon = "cloud"
	FilesystemIconDatabase   FilesystemIcon = "database"
	FilesystemIconGlobe      FilesystemIcon = "globe"
	FilesystemIconLock       FilesystemIcon = "lock"
	FilesystemIconStar       FilesystemIcon = "star"
	FilesystemIconCode       FilesystemIcon = "code"
	FilesystemIconFile       FilesystemIcon = "file"
	FilesystemIconGit        FilesystemIcon = "git"
	FilesystemIconTerminal   FilesystemIcon = "terminal"
	FilesystemIconSettings   FilesystemIcon = "settings"
	FilesystemIconPackage    FilesystemIcon = "package"
	FilesystemIconImage      FilesystemIcon = "image"
	FilesystemIconMusic      FilesystemIcon = "music"
	FilesystemIconVideo      FilesystemIcon = "video"
	FilesystemIconArchive    FilesystemIcon = "archive"
	FilesystemIconBookmark   FilesystemIcon = "bookmark"
	FilesystemIconClipboard  FilesystemIcon = "clipboard"
	FilesystemIconDownload   FilesystemIcon = "download"
	FilesystemIconUpload     FilesystemIcon = "upload"
	FilesystemIconTrash      FilesystemIcon = "trash"
	FilesystemIconHardDrive  FilesystemIcon = "hard-drive"
	FilesystemIconServer     FilesystemIcon = "server"
	FilesystemIconShield     FilesystemIcon = "shield"
	FilesystemIconZap        FilesystemIcon = "zap"
	FilesystemIconPuzzle     FilesystemIcon = "puzzle"
	FilesystemIconBrain      FilesystemIcon = "brain"
	FilesystemIconS3         FilesystemIcon = "s3"
	FilesystemIconGCS        FilesystemIcon = "gcs"
	FilesystemIconAzureBlob  FilesystemIcon = "azure-blob"
	FilesystemIconR2         FilesystemIcon = "r2"
)
