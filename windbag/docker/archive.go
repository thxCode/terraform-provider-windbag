package docker

import (
	"archive/zip"
	"bufio"
	"io"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/docker/cli/cli/command/image/build"
	"github.com/docker/docker/pkg/fileutils"
	"github.com/docker/docker/pkg/pools"
	"github.com/docker/docker/pkg/system"
	"github.com/mitchellh/go-homedir"
	"github.com/pkg/errors"
)

// GetBuildpathArchive retrieves the context to build.
func GetBuildpathArchive(path string, dockerfile string) (io.ReadCloser, error) {
	var excludes, err = build.ReadDockerignore(path)
	if err != nil {
		return nil, errors.Wrap(err, "failed ot get docker build ignored files")
	}
	excludes = build.TrimBuildFilesFromExcludes(excludes, dockerfile, false)

	path, err = homedir.Expand(path)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to expand docker build path %s", path)
	}

	return ZipWithOptions(path, &ZipOptions{
		ExcludePatterns: excludes,
	})
}

type ZipOptions struct {
	IncludeFiles     []string
	ExcludePatterns  []string
	IncludeSourceDir bool
}

// ZipWithOptions creates an archive from the directory at `path`, only including files whose relative
// paths are included in `options.IncludeFiles` (if non-nil) or not in `options.ExcludePatterns`.
func ZipWithOptions(srcPath string, options *ZipOptions) (io.ReadCloser, error) {

	// Fix the source path to work with long path names. This is a no-op
	// on platforms other than Windows.
	srcPath = fixVolumePathPrefix(srcPath)

	pm, err := fileutils.NewPatternMatcher(options.ExcludePatterns)
	if err != nil {
		return nil, err
	}

	pipeReader, pipeWriter := io.Pipe()

	go func() {
		var za = newZipAppender(pipeWriter)

		defer func() {
			// Make sure to check the error on Close.
			if err := za.ZipWriter.Close(); err != nil {
				log.Printf("[ERROR] Cannot close zip writer: %v\n", err)
			}
			if err := pipeWriter.Close(); err != nil {
				log.Printf("[ERROR] Cannot close pipe writer: %v\n", err)
			}
		}()

		// this buffer is needed for the duration of this piped stream
		defer pools.BufioWriter32KPool.Put(za.Buffer)

		// In general we log errors here but ignore them because
		// during e.g. a diff operation the container can continue
		// mutating the filesystem and we can see transient errors
		// from this

		var stat, err = os.Lstat(srcPath)
		if err != nil {
			return
		}

		if !stat.IsDir() {
			// We can't later join a non-dir with any includes because the
			// 'walk' will error if "file/." is stat-ed and "file" is not a
			// directory. So, we must split the source path and use the
			// basename as the include.
			if len(options.IncludeFiles) > 0 {
				log.Printf("[WARN] Zip: cannot archive a file with includes\n")
			}

			var dir, base = splitPathDirEntry(srcPath)
			srcPath = dir
			options.IncludeFiles = []string{base}
		}

		if len(options.IncludeFiles) == 0 {
			options.IncludeFiles = []string{"."}
		}

		var seen = make(map[string]struct{})

		for _, include := range options.IncludeFiles {
			var walkRoot = getWalkRoot(srcPath, include)
			_ = filepath.Walk(walkRoot, func(filePath string, f os.FileInfo, err error) error {
				if err != nil {
					log.Printf("[ERROR] Zip: Cannot stat file %s to zip: %v\n", srcPath, err)
					return nil
				}

				relFilePath, err := filepath.Rel(srcPath, filePath)
				if err != nil || (!options.IncludeSourceDir && relFilePath == "." && f.IsDir()) {
					// Error getting relative path OR we are looking
					// at the source directory path. Skip in both situations.
					return nil
				}

				if options.IncludeSourceDir && include == "." && relFilePath != "." {
					relFilePath = strings.Join([]string{".", relFilePath}, string(filepath.Separator))
				}

				var skip bool

				// If "include" is an exact match for the current file
				// then even if there's an "excludePatterns" pattern that
				// matches it, don't skip it. IOW, assume an explicit 'include'
				// is asking for that file no matter what - which is true
				// for some files, like .dockerignore and Dockerfile (sometimes)
				if include != relFilePath {
					skip, err = pm.Matches(relFilePath)
					if err != nil {
						log.Printf("[ERROR] Error matching %s: %v\n", relFilePath, err)
						return err
					}
				}

				if skip {
					// If we want to skip this file and its a directory
					// then we should first check to see if there's an
					// excludes pattern (e.g. !dir/file) that starts with this
					// dir. If so then we can't skip this dir.

					// Its not a dir then so we can just return/skip.
					if !f.IsDir() {
						return nil
					}

					// No exceptions (!...) in patterns so just skip dir
					if !pm.Exclusions() {
						return filepath.SkipDir
					}

					var dirSlash = relFilePath + string(filepath.Separator)

					for _, pat := range pm.Patterns() {
						if !pat.Exclusion() {
							continue
						}
						if strings.HasPrefix(pat.String()+string(filepath.Separator), dirSlash) {
							// found a match - so can't skip this dir
							return nil
						}
					}

					// No matching exclusion dir so just skip dir
					return filepath.SkipDir
				}

				if _, ok := seen[relFilePath]; ok {
					return nil
				}
				seen[relFilePath] = struct{}{}

				if err := za.addZipFile(filePath, relFilePath); err != nil {
					log.Printf("[ERROR] Cannot add file %s to zip: %v\n", filePath, err)
					// if pipe is broken, stop writing zip stream to it
					if err == io.ErrClosedPipe {
						return err
					}
				}
				return nil
			})
		}
	}()

	return pipeReader, nil
}

func newZipAppender(writer io.Writer) *zipAppender {
	return &zipAppender{
		ZipWriter: zip.NewWriter(writer),
		Buffer:    pools.BufioWriter32KPool.Get(nil),
	}
}

type zipAppender struct {
	ZipWriter *zip.Writer
	Buffer    *bufio.Writer
}

// addZipFile adds to the zip archive a file from `path` as `name`
func (za *zipAppender) addZipFile(path, name string) error {
	var fi, err = os.Lstat(path)
	if err != nil {
		return err
	}

	hdr, err := zip.FileInfoHeader(fi)
	if err != nil {
		return err
	}

	hdr.Name = name
	if fi.IsDir() {
		hdr.Name += "/" // required - strangely no mention of this in zip spec? but is in godoc...
		hdr.Method = zip.Store
	} else {
		hdr.Method = zip.Deflate
	}

	w, err := za.ZipWriter.CreateHeader(hdr)
	if err != nil {
		return err
	}

	if !fi.Mode().IsRegular() {
		// directories have no contents
		return nil
	}

	var link string
	if fi.Mode()&os.ModeSymlink != 0 {
		link, err = os.Readlink(path)
		if err != nil {
			return err
		}
	}
	if link != "" {
		_, err = w.Write([]byte(filepath.ToSlash(link)))
		if err != nil {
			return err
		}
		return nil
	}

	// We use system.OpenSequential to ensure we use sequential file
	// access on Windows to avoid depleting the standby list.
	// On Linux, this equates to a regular os.Open.
	file, err := system.OpenSequential(path)
	if err != nil {
		return err
	}

	za.Buffer.Reset(w)
	defer za.Buffer.Reset(nil)
	_, err = io.Copy(za.Buffer, file)
	_ = file.Close()
	if err != nil {
		return err
	}
	return za.Buffer.Flush()
}

// splitPathDirEntry splits the given path between its directory name and its
// basename by first cleaning the path but preserves a trailing "." if the
// original path specified the current directory.
func splitPathDirEntry(path string) (dir, base string) {
	cleanedPath := filepath.Clean(filepath.FromSlash(path))

	if specifiesCurrentDir(path) {
		cleanedPath += string(os.PathSeparator) + "."
	}

	return filepath.Dir(cleanedPath), filepath.Base(cleanedPath)
}

// specifiesCurrentDir returns whether the given path specifies
// a "current directory", i.e., the last path segment is `.`.
func specifiesCurrentDir(path string) bool {
	return filepath.Base(path) == "."
}
