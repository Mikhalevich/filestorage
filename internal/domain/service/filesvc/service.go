package filesvc

type Service struct {
	rootDir string
}

// New creates file service. All paths passed to the service are resolved relative to rootDir.
func New(rootDir string) *Service {
	return &Service{
		rootDir: rootDir,
	}
}
