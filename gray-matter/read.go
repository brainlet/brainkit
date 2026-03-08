package graymatter

import (
	"os"
)

// Read reads a file from the filesystem and parses its front-matter.
// It returns a File with the parsed content and the filepath set.
func Read(filepath string, opts ...Options) (File, error) {
	content, err := os.ReadFile(filepath)
	if err != nil {
		return File{}, err
	}

	file, err := Parse(string(content), opts...)
	if err != nil {
		return File{}, err
	}

	file.Path = filepath
	return file, nil
}
